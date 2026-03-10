package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axon-chain/axon/x/agent/types"
)

var challengePool = []struct {
	Question string
	Answer   string
	Category string
}{
	{"What is the time complexity of binary search?", "O(log n)", "algorithms"},
	{"In Ethereum, what opcode is used to transfer ETH to another address?", "CALL", "blockchain"},
	{"What consensus algorithm does CometBFT use?", "PBFT", "blockchain"},
	{"What is the derivative of x^3 with respect to x?", "3x^2", "math"},
	{"In Go, what keyword is used to launch a concurrent goroutine?", "go", "programming"},
	{"What data structure uses LIFO (Last In First Out)?", "stack", "data_structures"},
	{"What is the SHA-256 hash length in bits?", "256", "cryptography"},
	{"What layer of the OSI model does TCP operate at?", "transport", "networking"},
	{"In a Merkle tree, what is stored in leaf nodes?", "hash of data", "data_structures"},
	{"What EIP introduced EIP-1559 fee mechanism?", "EIP-1559", "blockchain"},
	{"What is the base case needed for in recursive functions?", "termination", "algorithms"},
	{"What type of encryption uses the same key for encrypt and decrypt?", "symmetric", "cryptography"},
	{"In SQL, what clause filters groups after aggregation?", "HAVING", "databases"},
	{"What is a smart contract's equivalent of a constructor in Solidity?", "constructor", "blockchain"},
	{"Name the sorting algorithm with best-case O(n) and worst-case O(n^2).", "insertion sort", "algorithms"},
	{"What HTTP method is idempotent and used to update resources?", "PUT", "networking"},
	{"In BFT consensus, what fraction of nodes can be faulty?", "less than 1/3", "blockchain"},
	{"What does CAP theorem state about distributed systems?", "cannot have all three: consistency, availability, partition tolerance", "distributed_systems"},
	{"What is the purpose of a nonce in blockchain transactions?", "prevent replay attacks", "blockchain"},
	{"What Cosmos SDK module handles token transfers?", "bank", "blockchain"},
	{"What is the space complexity of a hash table?", "O(n)", "data_structures"},
	{"Name the pattern where an object notifies dependents of state changes.", "observer", "design_patterns"},
	{"What is the maximum block gas limit set in Axon genesis?", "40000000", "axon"},
	{"In proof of stake, what prevents nothing-at-stake attacks?", "slashing", "blockchain"},
	{"What encoding does Cosmos SDK use for addresses?", "bech32", "blockchain"},
	{"What is the halting problem about?", "undecidability of program termination", "theory"},
	{"What protocol does gRPC use for transport?", "HTTP/2", "networking"},
	{"Name the principle: a class should have only one reason to change.", "single responsibility", "design_patterns"},
	{"What is the gas cost of SSTORE in Ethereum when setting a zero to non-zero value?", "20000", "blockchain"},
	{"What type of database is LevelDB?", "key-value", "databases"},
}

func (k Keeper) GetChallenge(ctx sdk.Context, epoch uint64) (types.AIChallenge, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyChallenge(epoch))
	if bz == nil {
		return types.AIChallenge{}, false
	}
	var challenge types.AIChallenge
	k.cdc.MustUnmarshal(bz, &challenge)
	return challenge, true
}

func (k Keeper) SetChallenge(ctx sdk.Context, challenge types.AIChallenge) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&challenge)
	store.Set(types.KeyChallenge(challenge.Epoch), bz)
}

// GenerateChallenge creates a deterministic challenge for the epoch.
func (k Keeper) GenerateChallenge(ctx sdk.Context, epoch uint64) types.AIChallenge {
	poolSize := uint64(len(challengePool))
	if poolSize == 0 {
		return types.AIChallenge{}
	}

	seed := sha256.Sum256(append(
		ctx.HeaderHash(),
		types.Uint64ToBytes(epoch)...,
	))
	index := types.BytesToUint64(seed[:8]) % poolSize
	selected := challengePool[index]

	questionHash := sha256.Sum256([]byte(selected.Question))
	params := k.GetParams(ctx)

	challenge := types.AIChallenge{
		Epoch:         epoch,
		ChallengeHash: hex.EncodeToString(questionHash[:]),
		ChallengeType: selected.Category,
		ChallengeData: selected.Question,
		DeadlineBlock: ctx.BlockHeight() + params.AiChallengeWindow,
	}

	k.SetChallenge(ctx, challenge)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"ai_challenge_generated",
		sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
		sdk.NewAttribute("category", selected.Category),
		sdk.NewAttribute("question_hash", challenge.ChallengeHash),
		sdk.NewAttribute("deadline_block", fmt.Sprintf("%d", challenge.DeadlineBlock)),
	))

	return challenge
}

// getChallengeAnswer re-derives the answer for the given challenge by finding
// the matching question in the pool.
func getChallengeAnswer(challenge types.AIChallenge) string {
	for _, c := range challengePool {
		if c.Question == challenge.ChallengeData {
			return c.Answer
		}
	}
	return ""
}

func (k Keeper) GetEpochResponses(ctx sdk.Context, epoch uint64) []types.AIResponse {
	store := ctx.KVStore(k.storeKey)
	prefix := types.KeyAIResponsePrefix(epoch)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var responses []types.AIResponse
	for ; iterator.Valid(); iterator.Next() {
		var response types.AIResponse
		k.cdc.MustUnmarshal(iterator.Value(), &response)
		responses = append(responses, response)
	}
	return responses
}

// CheatPenaltyReputation is the reputation penalty for cheating.
const CheatPenaltyReputation = -20

// CheatPenaltyStakePercent is the percentage of stake slashed for cheating.
const CheatPenaltyStakePercent = 20

// EvaluateEpochChallenges scores responses and detects cheating (whitepaper §8.6 path 5).
// Cheating = multiple validators submit identical commit hashes (collusion/copying).
func (k Keeper) EvaluateEpochChallenges(ctx sdk.Context, epoch uint64) {
	challenge, found := k.GetChallenge(ctx, epoch)
	if !found {
		return
	}

	correctAnswer := getChallengeAnswer(challenge)
	responses := k.GetEpochResponses(ctx, epoch)
	respondents := make(map[string]bool)
	cheaters := k.detectCheaters(responses)

	for _, resp := range responses {
		respondents[resp.ValidatorAddress] = true

		if cheaters[resp.ValidatorAddress] {
			k.penalizeCheater(ctx, resp.ValidatorAddress)
			resp.Score = -1
		} else {
			score := scoreResponse(resp, correctAnswer)
			bonus := calculateAIBonus(score)
			k.SetAIBonus(ctx, resp.ValidatorAddress, bonus)

			if score >= 80 {
				k.UpdateReputation(ctx, resp.ValidatorAddress, 2)
			} else if score >= 50 {
				k.UpdateReputation(ctx, resp.ValidatorAddress, 1)
			}
			resp.Score = int64(score)
		}

		store := ctx.KVStore(k.storeKey)
		resp.Evaluated = true
		bz := k.cdc.MustMarshal(&resp)
		store.Set(types.KeyAIResponse(epoch, resp.ValidatorAddress), bz)
	}

	k.IterateAgents(ctx, func(agent types.Agent) bool {
		if agent.Status == types.AgentStatus_AGENT_STATUS_ONLINE && !respondents[agent.Address] {
			k.SetAIBonus(ctx, agent.Address, 0)
		}
		return false
	})

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"ai_challenge_evaluated",
		sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
		sdk.NewAttribute("responses_count", fmt.Sprintf("%d", len(responses))),
		sdk.NewAttribute("cheaters_count", fmt.Sprintf("%d", len(cheaters))),
	))
}

// detectCheaters finds validators that submitted identical commit hashes (collusion).
// If 2+ validators share the same commit hash they are all flagged.
func (k Keeper) detectCheaters(responses []types.AIResponse) map[string]bool {
	commitCounts := make(map[string][]string) // commitHash → list of addresses
	for _, resp := range responses {
		if resp.CommitHash == "" {
			continue
		}
		commitCounts[resp.CommitHash] = append(commitCounts[resp.CommitHash], resp.ValidatorAddress)
	}

	cheaters := make(map[string]bool)
	for _, addrs := range commitCounts {
		if len(addrs) > 1 {
			for _, addr := range addrs {
				cheaters[addr] = true
			}
		}
	}
	return cheaters
}

// penalizeCheater slashes 20% of stake, reputation -20, AIBonus = -5.
func (k Keeper) penalizeCheater(ctx sdk.Context, address string) {
	k.SetAIBonus(ctx, address, -5)
	k.UpdateReputation(ctx, address, CheatPenaltyReputation)

	agent, found := k.GetAgent(ctx, address)
	if !found {
		return
	}

	slashAmount := agent.StakeAmount.Amount.MulRaw(CheatPenaltyStakePercent).QuoRaw(100)
	if slashAmount.IsPositive() {
		slashCoin := sdk.NewCoin("aaxon", slashAmount)
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(slashCoin)); err != nil {
			k.Logger(ctx).Error("failed to slash cheater stake", "address", address, "error", err)
			return
		}
		agent.StakeAmount = agent.StakeAmount.Sub(slashCoin)
		k.SetAgent(ctx, agent)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"ai_challenge_cheat_detected",
		sdk.NewAttribute("address", address),
		sdk.NewAttribute("slashed", slashAmount.String()),
		sdk.NewAttribute("reputation_penalty", fmt.Sprintf("%d", CheatPenaltyReputation)),
	))
}

func scoreResponse(resp types.AIResponse, correctAnswer string) int {
	if resp.RevealData == "" {
		return 0
	}

	normalizedReveal := normalizeAnswer(resp.RevealData)
	normalizedAnswer := normalizeAnswer(correctAnswer)

	if normalizedReveal == normalizedAnswer {
		return 100
	}

	if len(normalizedReveal) > 0 && len(normalizedAnswer) > 0 {
		if stringContains(normalizedReveal, normalizedAnswer) || stringContains(normalizedAnswer, normalizedReveal) {
			return 50
		}
	}

	return 10
}

func calculateAIBonus(score int) int64 {
	switch {
	case score >= 90:
		return 30
	case score >= 70:
		return 20
	case score >= 50:
		return 10
	case score >= 20:
		return 5
	default:
		return 0
	}
}

func normalizeAnswer(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			result = append(result, c)
		}
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
