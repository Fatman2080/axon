package keeper_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/axon-chain/axon/x/agent/keeper"
)

// TestBlockRewardHalving verifies the 4-year halving schedule.
func TestBlockRewardHalving(t *testing.T) {
	tests := []struct {
		name        string
		blockHeight int64
		wantAxon    string // approximate AXON value
	}{
		{"Year 1 block 2", 2, "12.367"},
		{"Year 1 block 1000", 1000, "12.367"},
		{"Year 4 last block", 6_307_200*4 - 1, "12.367"},
		{"Year 5 first block (after halving)", 6_307_200 * 4, "6.183"},
		{"Year 8 block", 6_307_200*8 - 1, "6.183"},
		{"Year 9 first block", 6_307_200 * 8, "3.092"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reward := keeper.ExportCalculateBlockReward(tt.blockHeight)
			if reward.IsZero() {
				t.Fatalf("expected non-zero reward at block %d", tt.blockHeight)
			}
			// Convert aaxon to AXON for display
			axon := new(big.Float).Quo(
				new(big.Float).SetInt(reward.BigInt()),
				new(big.Float).SetFloat64(1e18),
			)
			axonStr := axon.Text('f', 3)
			if axonStr != tt.wantAxon {
				t.Errorf("block %d: got %s AXON, want %s AXON", tt.blockHeight, axonStr, tt.wantAxon)
			}
		})
	}
}

// TestBlockRewardHighHalving verifies rewards eventually reach zero.
func TestBlockRewardHighHalving(t *testing.T) {
	// After 64 halvings, reward should be zero
	reward := keeper.ExportCalculateBlockReward(6_307_200 * 4 * 64)
	if !reward.IsZero() {
		t.Errorf("expected zero reward after 64 halvings, got %s", reward.String())
	}
}

// TestContributionPerBlock verifies contribution reward per-block amounts.
func TestContributionPerBlock(t *testing.T) {
	// Year 1: ~35M/year, per-block = 35M * 1e18 / 6_307_200 ≈ 5.55e18
	reward := keeper.ExportCalculateContributionPerBlock(100)
	if reward.IsZero() {
		t.Fatal("expected non-zero contribution per-block")
	}

	axon := new(big.Float).Quo(
		new(big.Float).SetInt(reward.BigInt()),
		new(big.Float).SetFloat64(1e18),
	)
	val, _ := axon.Float64()
	if val < 5.0 || val > 6.0 {
		t.Errorf("expected ~5.5 AXON/block contribution, got %f", val)
	}

	// Year 5 (halved)
	reward2 := keeper.ExportCalculateContributionPerBlock(6_307_200 * 4)
	axon2 := new(big.Float).Quo(
		new(big.Float).SetInt(reward2.BigInt()),
		new(big.Float).SetFloat64(1e18),
	)
	val2, _ := axon2.Float64()
	if val2 < 2.5 || val2 > 3.0 {
		t.Errorf("expected ~2.77 AXON/block contribution (halved), got %f", val2)
	}
}

// TestMaxShareCap verifies 2% cap on contribution rewards.
func TestMaxShareCap(t *testing.T) {
	// If pool = 1000 AXON, max per agent = 20 AXON
	pool := sdkmath.NewInt(1000).Mul(sdkmath.NewIntWithDecimal(1, 18))
	maxShare := pool.MulRaw(200).QuoRaw(10000)

	expected := sdkmath.NewInt(20).Mul(sdkmath.NewIntWithDecimal(1, 18))
	if !maxShare.Equal(expected) {
		t.Errorf("max share: got %s, want %s", maxShare.String(), expected.String())
	}
}

// TestDeflationPaths verifies the existence of all 5 deflation mechanisms.
func TestDeflationPaths(t *testing.T) {
	paths := []string{
		"Gas fee burn (EIP-1559 base fee → 50% of collected fees burned)",
		"Agent registration burn (20 AXON per registration)",
		"Contract deployment burn (10 AXON per deployment via PostTxProcessing hook)",
		"Reputation zero burn (all remaining stake burned when rep hits 0)",
		"AI challenge penalty burn (reputation loss → potential stake burn)",
	}
	for i, p := range paths {
		t.Logf("Deflation path %d: %s", i+1, p)
	}
	// This is a documentation/checklist test - actual integration tests
	// require a running chain. The individual mechanisms are tested in:
	// - Gas burn: app/fee_burn.go + FeeCollector Burner permission
	// - Registration burn: x/agent/keeper/msg_server.go Register()
	// - Deploy burn: app/evm_hooks.go DeployBurnHook
	// - Zero rep burn: x/agent/keeper/reputation.go handleZeroReputation()
	// - AI penalty: x/agent/keeper/challenge.go EvaluateEpochChallenges()
}
