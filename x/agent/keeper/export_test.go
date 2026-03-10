package keeper

import (
	"github.com/axon-chain/axon/x/agent/types"
)

func GetChallengePoolSize() []struct{} {
	return make([]struct{}, len(challengePool))
}

func NormalizeAnswerForTest(s string) string {
	return normalizeAnswer(s)
}

func ScoreResponseForTest(reveal, answer string) int {
	return scoreResponse(types.AIResponse{RevealData: reveal}, answer)
}

func CalculateAIBonusForTest(score int) int64 {
	return calculateAIBonus(score)
}
