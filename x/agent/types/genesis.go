package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Agents: []Agent{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
