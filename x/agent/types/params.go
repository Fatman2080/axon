package types

func DefaultParams() Params {
	return Params{
		MinRegisterStake:  100,
		RegisterBurnAmount: 20,
		ContractDeployBurn: 10,
		InitialReputation: 10,
		MaxReputation:     100,
		HeartbeatInterval: 100,   // blocks
		HeartbeatTimeout:  720,   // blocks (~1 hour)
		EpochLength:       720,   // blocks (~1 hour)
		AiChallengeWindow: 50,    // blocks (~4 minutes)
	}
}

func (p Params) Validate() error {
	if p.MinRegisterStake == 0 {
		return ErrInsufficientStake
	}
	return nil
}
