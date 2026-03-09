package types

const (
	ModuleName = "agent"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	// KV store prefixes
	AgentKeyPrefix        = "Agent/value/"
	AgentCountKey         = "Agent/count"
	ParamsKey             = "Params"
	ChallengeKeyPrefix    = "Challenge/value/"
	AIResponseKeyPrefix   = "AIResponse/value/"
	ContributionKeyPrefix = "Contribution/value/"
)

func KeyAgent(address string) []byte {
	return []byte(AgentKeyPrefix + address)
}

func KeyChallenge(epoch uint64) []byte {
	return append([]byte(ChallengeKeyPrefix), sdk_Uint64ToBytes(epoch)...)
}

func KeyAIResponse(epoch uint64, validator string) []byte {
	return append([]byte(AIResponseKeyPrefix), []byte(validator+"/"+string(sdk_Uint64ToBytes(epoch)))...)
}

func sdk_Uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}
