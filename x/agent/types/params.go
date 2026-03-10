package types

const (
	// DeregisterCooldownBlocks is 7 days at 5s/block = 120960 blocks.
	// For local testing, use a smaller value via genesis override.
	DeregisterCooldownBlocks int64 = 120960

	// ReputationGainEpochOnline is the reputation gained per epoch of continuous online status.
	ReputationGainEpochOnline int64 = 1
	// ReputationGainHeartbeatStreak is gained every 1000 blocks of continuous heartbeats.
	ReputationGainHeartbeatStreak int64 = 1
	// ReputationGainActiveEpoch is gained for >=10 tx in an epoch (stored as x10 internally).
	ReputationGainActiveEpoch int64 = 1
	// ReputationLossOffline is lost when going offline.
	ReputationLossOffline int64 = -5
	// ReputationLossSlashing is lost on double-sign.
	ReputationLossSlashing int64 = -50
	// ReputationLossNoHeartbeatEpoch is lost per epoch without heartbeat.
	ReputationLossNoHeartbeatEpoch int64 = -1
)

func DefaultParams() Params {
	return Params{
		MinRegisterStake:   100,
		RegisterBurnAmount: 20,
		ContractDeployBurn: 10,
		InitialReputation:  10,
		MaxReputation:      100,
		HeartbeatInterval:  100,
		HeartbeatTimeout:   720,
		EpochLength:        720,
		AiChallengeWindow:  50,
	}
}

func (p Params) Validate() error {
	if p.MinRegisterStake == 0 {
		return ErrInsufficientStake
	}
	return nil
}
