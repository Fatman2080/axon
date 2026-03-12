package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var syncRoundCounter int64

func (s *Server) runSyncRound() {
	if !atomic.CompareAndSwapInt32(&s.syncRoundRunning, 0, 1) {
		logWarn("sync", "round skipped — previous round still running")
		return
	}
	defer atomic.StoreInt32(&s.syncRoundRunning, 0)

	round := atomic.AddInt64(&syncRoundCounter, 1)

	evm := s.getEVMClient()
	if evm == nil {
		logWarn("sync", "round #%d skipped — EVM client not ready", round)
		return
	}

	assignedKeys, _ := s.store.listAssignedPublicKeys()
	logInfo("sync", "=== round #%d started === agents_assigned: %d, allocator: %s", round, len(assignedKeys), s.contractAllocator)

	start := time.Now()

	// --- Phase 1: EVM contract discovery ---
	total, err := evm.GetVaultCount()
	if err != nil {
		logError("sync", "evm: vaultCount failed: %v", err)
		return
	}
	logInfo("sync", "evm: discovering vaults... total: %d", total)

	const batchSize = 50
	var allVaultAddrs []common.Address
	batches := (total + batchSize - 1) / batchSize
	for b := 0; b < batches; b++ {
		bStart := b * batchSize
		bCount := batchSize
		if bStart+bCount > total {
			bCount = total - bStart
		}
		addrs, err := evm.GetVaultsByRange(bStart, bCount)
		if err != nil {
			logError("sync", "evm: getVaultsByRange(%d, %d) failed: %v", bStart, bCount, err)
			continue
		}
		allVaultAddrs = append(allVaultAddrs, addrs...)
	}

	// Fetch info in batches
	var records []VaultRecord
	activeCount := 0
	for b := 0; b < len(allVaultAddrs); b += batchSize {
		end := b + batchSize
		if end > len(allVaultAddrs) {
			end = len(allVaultAddrs)
		}
		batch := allVaultAddrs[b:end]

		infos, err := evm.GetVaultsInfo(batch)
		if err != nil {
			logError("sync", "evm: getVaultsInfo batch %d/%d failed: %v", b/batchSize+1, (len(allVaultAddrs)+batchSize-1)/batchSize, err)
			continue
		}
		logInfo("sync", "evm: fetched info for %d vaults (batch %d/%d)", len(infos), b/batchSize+1, (len(allVaultAddrs)+batchSize-1)/batchSize)

		for i, info := range infos {
			vaultAddr := batch[i]
			userAddr := ""
			if info.user != (common.Address{}) {
				userAddr = strings.ToLower(info.user.Hex())
			}
			if userAddr == "" || userAddr == "0x0000000000000000000000000000000000000000" {
				userAddr = ""
			} else if info.valid {
				activeCount++
			}
			records = append(records, VaultRecord{
				VaultAddress:     strings.ToLower(vaultAddr.Hex()),
				UserAddress:      userAddr,
				EVMBalance:       info.balance,
				InitialCapital:   info.capital,
				Valid:            info.valid,
				AllocatorAddress: s.contractAllocator,
			})
		}
	}

	if len(records) > 0 {
		if err := s.store.batchUpsertAgentVaults(records); err != nil {
			logError("sync", "evm: batch upsert failed: %v", err)
		}
	}
	if err := s.store.syncAgentVaultState(records); err != nil {
		logError("sync", "evm: sync agent vault state failed: %v", err)
	}
	logInfo("sync", "evm: %d vaults upserted, %d with active users", len(records), activeCount)

	// --- Phase 2: Hyperliquid data (concurrent) ---
	// Query by vault_address — the vault is the trading entity on Hyperliquid L1
	hlConcurrency := 5
	if v := s.store.getSettingDefault("sync_hl_concurrency", "5"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hlConcurrency = n
		}
	}

	type hlTask struct {
		vaultAddress string
		userAddress  string
	}
	var tasks []hlTask
	for _, r := range records {
		if r.UserAddress != "" && r.Valid {
			tasks = append(tasks, hlTask{vaultAddress: r.VaultAddress, userAddress: r.UserAddress})
		}
	}

	// Atomic accumulators for vault-level treasury data
	var vaultPerpsTotal, vaultSpotTotal, vaultPnlTotal int64 // stored as cents (×100)

	if len(tasks) > 0 && s.hyperliquid != nil {
		logInfo("sync", "hl: syncing %d vaults (concurrency: %d)...", len(tasks), hlConcurrency)

		var okCount, failCount int64
		sem := make(chan struct{}, hlConcurrency)
		var wg sync.WaitGroup

		for _, task := range tasks {
			wg.Add(1)
			sem <- struct{}{}
			go func(t hlTask) {
				defer wg.Done()
				defer func() { <-sem }()

				// Use vault_address to query Hyperliquid — vault is the trading account
				data, err := s.hyperliquid.FetchAccountData(t.vaultAddress)
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					logWarn("sync", "hl: vault %s failed: %v", t.vaultAddress[:10]+"..."+t.vaultAddress[len(t.vaultAddress)-4:], err)
					_ = s.store.updateVaultSyncError(t.vaultAddress, err.Error())
					return
				}
				atomic.AddInt64(&okCount, 1)
				_ = s.store.updateVaultHyperliquidData(t.vaultAddress, data.AccountValue, data.UnrealizedPNL)

				// Accumulate perps + pnl
				atomic.AddInt64(&vaultPerpsTotal, int64(math.Round(data.AccountValue*100)))
				atomic.AddInt64(&vaultPnlTotal, int64(math.Round(data.UnrealizedPNL*100)))

				// Fetch spot balance
				if sb, err := s.hyperliquid.FetchSpotBalance(t.vaultAddress); err == nil && sb > 0 {
					atomic.AddInt64(&vaultSpotTotal, int64(math.Round(sb*100)))
				}

				// Snapshot account value follows L1 perps accountValue for TVL consistency.
				_, _ = s.store.saveAgentSnapshot(t.userAddress, data.AccountValue, data.UnrealizedPNL, "hyperliquid")

				// Persist fills for performance/leaderboard metrics.
				if fills, err := s.hyperliquid.FetchUserFills(t.vaultAddress); err == nil {
					limit := len(fills)
					if limit > 200 {
						limit = 200
					}
					for i := 0; i < limit; i++ {
						fill := fills[i]
						raw, marshalErr := json.Marshal(fill)
						if marshalErr != nil {
							logError("sync", "hl: marshal fill failed (user=%s, i=%d): %v", t.userAddress, i, marshalErr)
							continue
						}
						fillID := strings.TrimSpace(fill.Hash)
						if fillID == "" {
							fillID = fmt.Sprintf("%s_%d_%d", t.userAddress, fill.Time, i)
						}
						if err := s.store.upsertAgentVaultOrder(t.vaultAddress, t.userAddress, fillID, fill.Time, string(raw)); err != nil {
							logError("sync", "hl: upsertAgentVaultOrder failed (user=%s, fill=%s): %v", t.userAddress, fillID, err)
						}
					}
				}
			}(task)
		}
		wg.Wait()
		logInfo("sync", "hl: done — %d ok, %d failed", okCount, failCount)
	}

	// --- Phase 2.5: Treasury snapshot ---
	s.collectTreasurySnapshot(evm, records, activeCount,
		float64(vaultPerpsTotal)/100, float64(vaultSpotTotal)/100, float64(vaultPnlTotal)/100)

	// --- Phase 2.6: Platform snapshot ---
	s.collectPlatformSnapshot(records, activeCount,
		float64(vaultPerpsTotal)/100, float64(vaultSpotTotal)/100, float64(vaultPnlTotal)/100)

	// --- Phase 3: Consistency check ---
	if len(assignedKeys) > 0 {
		vaultUsers := make(map[string]bool)
		for _, r := range records {
			if r.UserAddress != "" && r.Valid {
				vaultUsers[r.UserAddress] = true
			}
		}

		lostCount := 0
		for _, key := range assignedKeys {
			if !vaultUsers[strings.ToLower(key)] {
				lostCount++
				logWarn("sync", "consistency: agent %s no longer has vault — on-chain access revoked", key)
			}
		}
		logInfo("sync", "consistency: %d assigned accounts checked, %d lost vault access", len(assignedKeys), lostCount)
	}

	// --- Phase 4: Refresh public API cache ---
	s.refreshPublicCache()
	s.reconcileAgentArenaState()

	elapsed := time.Since(start).Seconds()
	logInfo("sync", "=== round #%d completed in %.1fs === next in %ds", round, elapsed, s.syncIntervalSecs)
}

func (s *Server) startAutoSync(intervalSeconds int) {
	if s.syncStop != nil {
		close(s.syncStop)
		s.syncStop = nil
	}

	s.syncIntervalSecs = intervalSeconds

	if intervalSeconds <= 0 {
		logInfo("sync", "auto sync disabled (intervalSeconds <= 0)")
		return
	}
	interval := time.Duration(intervalSeconds) * time.Second
	stop := make(chan struct{})
	s.syncStop = stop
	go func() {
		// Wait for EVM client to be ready (up to 60 seconds)
		for i := 0; i < 60; i++ {
			if s.getEVMClient() != nil {
				break
			}
			select {
			case <-stop:
				return
			case <-time.After(1 * time.Second):
			}
		}

		s.runSyncRound()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runSyncRound()
			case <-stop:
				return
			}
		}
	}()
	logInfo("sync", "auto sync started (interval: %ds)", intervalSeconds)
}

// collectTreasurySnapshot gathers vault + allocator + owner balances and saves a treasury snapshot.
func (s *Server) collectTreasurySnapshot(evm *EVMClient, records []VaultRecord, activeCount int,
	vaultPerps, vaultSpot, vaultPnl float64) {

	// Vault EVM + Capital from records
	var vaultEvm, vaultCapital float64
	for _, r := range records {
		vaultEvm += r.EVMBalance
		vaultCapital += r.InitialCapital
	}

	snap := TreasurySnapshot{
		VaultEvm:         vaultEvm,
		VaultPerps:       vaultPerps,
		VaultSpot:        vaultSpot,
		VaultPnl:         vaultPnl,
		VaultCapital:     vaultCapital,
		VaultCount:       len(records),
		ActiveVaultCount: activeCount,
		AllocatorAddress: s.contractAllocator,
	}

	// Get allocator owner address
	ownerAddr, err := evm.GetAllocatorOwner()
	if err != nil {
		logWarn("sync", "treasury: failed to get allocator owner: %v", err)
	} else {
		snap.OwnerAddress = strings.ToLower(ownerAddr.Hex())
	}

	// Concurrent fetch: allocator + owner balances (EVM/Perps/Spot)
	type balResult struct {
		evmBal, perpsBal, spotBal float64
	}
	var allocBal, ownerBal balResult
	var wg sync.WaitGroup

	// Allocator balances
	wg.Add(1)
	go func() {
		defer wg.Done()
		bal, err := evm.GetUSDCBalance(evm.allocatorAddress)
		if err != nil {
			logWarn("sync", "treasury: allocator EVM balance failed: %v", err)
		} else {
			allocBal.evmBal = bal
		}
		if s.hyperliquid != nil {
			addr := strings.ToLower(evm.allocatorAddress.Hex())
			if data, err := s.hyperliquid.FetchAccountData(addr); err == nil {
				allocBal.perpsBal = data.AccountValue
			}
			if spotBal, err := s.hyperliquid.FetchSpotBalance(addr); err == nil {
				allocBal.spotBal = spotBal
			}
		}
	}()

	// Owner balances
	if ownerAddr != (common.Address{}) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bal, err := evm.GetUSDCBalance(ownerAddr)
			if err != nil {
				logWarn("sync", "treasury: owner EVM balance failed: %v", err)
			} else {
				ownerBal.evmBal = bal
			}
			if s.hyperliquid != nil {
				addr := strings.ToLower(ownerAddr.Hex())
				if data, err := s.hyperliquid.FetchAccountData(addr); err == nil {
					ownerBal.perpsBal = data.AccountValue
				}
				if spotBal, err := s.hyperliquid.FetchSpotBalance(addr); err == nil {
					ownerBal.spotBal = spotBal
				}
			}
		}()
	}
	wg.Wait()

	snap.AllocatorEvm = allocBal.evmBal
	snap.AllocatorPerps = allocBal.perpsBal
	snap.AllocatorSpot = allocBal.spotBal
	snap.OwnerEvm = ownerBal.evmBal
	snap.OwnerPerps = ownerBal.perpsBal
	snap.OwnerSpot = ownerBal.spotBal

	snap.TotalFunds = (vaultEvm + vaultPerps + vaultSpot) +
		(allocBal.evmBal + allocBal.perpsBal + allocBal.spotBal) +
		(ownerBal.evmBal + ownerBal.perpsBal + ownerBal.spotBal)

	if err := s.store.saveTreasurySnapshot(snap); err != nil {
		logError("sync", "treasury: save snapshot failed: %v", err)
	}

	vaultTotal := vaultEvm + vaultPerps + vaultSpot
	allocTotal := allocBal.evmBal + allocBal.perpsBal + allocBal.spotBal
	ownerTotal := ownerBal.evmBal + ownerBal.perpsBal + ownerBal.spotBal
	logInfo("sync", "treasury: total=$%.2f (vault=$%.2f alloc=$%.2f owner=$%.2f)",
		snap.TotalFunds, vaultTotal, allocTotal, ownerTotal)
}

// collectPlatformSnapshot gathers platform-level stats and saves a platform snapshot.
func (s *Server) collectPlatformSnapshot(records []VaultRecord, activeCount int,
	vaultPerps, vaultSpot, vaultPnl float64) {

	// TVL = sum of all vault account_values from agent_vaults + EVM balances
	var totalTVL, totalCapital float64
	for _, r := range records {
		totalTVL += r.EVMBalance
		totalCapital += r.InitialCapital
	}
	// Add perps (L1 value)
	totalTVL += vaultPerps + vaultSpot

	// Also add allocator + owner from latest treasury snapshot for total assets
	tsnap, err := s.store.getLatestTreasurySnapshot()
	if err == nil && tsnap != nil {
		totalTVL = tsnap.TotalFunds // total assets = vault + allocator + owner (all 3 balances)
	}

	var userCount, totalAgentCount, totalTrades int
	s.store.db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&userCount)
	s.store.db.QueryRow(`SELECT COUNT(1) FROM agent_accounts`).Scan(&totalAgentCount)
	s.store.db.QueryRow(`SELECT COUNT(1) FROM agent_vault_orders`).Scan(&totalTrades)

	snap := PlatformSnapshot{
		TotalTVL:         totalTVL,
		TotalPnL:         vaultPnl,
		TotalCapital:     totalCapital,
		UserCount:        userCount,
		ActiveAgentCount: activeCount,
		TotalAgentCount:  totalAgentCount,
		TotalTrades:      totalTrades,
	}

	if err := s.store.savePlatformSnapshot(snap); err != nil {
		logError("sync", "platform: save snapshot failed: %v", err)
	}

	_ = s.store.setSetting("last_sync_at", nowISO())
	_ = s.store.setSetting("last_sync_round", strconv.FormatInt(atomic.LoadInt64(&syncRoundCounter), 10))

	logInfo("sync", "platform: tvl=$%.2f users=%d agents=%d/%d trades=%d",
		snap.TotalTVL, userCount, activeCount, totalAgentCount, totalTrades)
}
