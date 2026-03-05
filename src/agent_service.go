package main

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func (s *Server) syncByPublicKey(publicKey string) (AgentMarketItem, error) {
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	if publicKey == "" {
		return AgentMarketItem{}, errors.New("empty public key")
	}

	item := AgentMarketItem{
		PublicKey: publicKey,
	}

	if s.hyperliquid != nil {
		if data, err := s.hyperliquid.FetchAccountData(publicKey); err == nil {
			item.AccountValue = data.AccountValue
			item.LastSyncedAt = data.LastSyncAt
			_, _ = s.store.saveAgentSnapshot(publicKey, data.AccountValue, data.UnrealizedPNL, "hyperliquid")
		}
	}

	snapshots, err := s.store.listAgentSnapshots(publicKey, 120, "ALL")
	if err != nil {
		return AgentMarketItem{}, err
	}
	if len(snapshots) == 0 {
		return item, nil
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})

	latest := snapshots[len(snapshots)-1]
	item.AccountValue = latest.AccountValue
	item.LastSyncedAt = latest.CreatedAt

	first := snapshots[0]
	item.TotalPnL = latest.AccountValue - first.AccountValue

	// EVM vault data (non-blocking: log errors but don't fail)
	evm := s.getEVMClient()
	if evm != nil {
		addr := common.HexToAddress(publicKey)
		infos, err := evm.GetAgentsInfo([]common.Address{addr})
		if err != nil {
			logWarn("sync", "evm vault read for %s: %v", publicKey, err)
		} else if info, ok := infos[addr]; ok {
			if info.VaultAddress != (common.Address{}) {
				item.VaultAddress = info.VaultAddress.Hex()
			}
			item.EVMBalance = info.EVMBalance
			item.AgentStatus = info.Status
			_ = s.store.updateVaultData(publicKey, item.VaultAddress, info.EVMBalance, info.Status)
		}
	}

	return item, nil
}

func (s *Server) syncAllAssignedAgents() {
	keys, err := s.store.listAssignedPublicKeys()
	if err != nil {
		logError("sync", "failed to list assigned keys: %v", err)
		return
	}
	if len(keys) == 0 {
		logInfo("sync", "no assigned agents to sync")
		return
	}

	synced, failed := 0, 0
	for _, key := range keys {
		if _, err := s.syncByPublicKey(key); err != nil {
			logError("sync", "failed to sync %s: %v", key, err)
			failed++
		} else {
			synced++
		}
		time.Sleep(2 * time.Second)
	}
	logInfo("sync", "done — %d synced, %d failed, %d total", synced, failed, len(keys))
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
		s.syncAllAssignedAgents()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.syncAllAssignedAgents()
			case <-stop:
				return
			}
		}
	}()
	logInfo("sync", "auto sync started (interval: %ds)", intervalSeconds)
}

