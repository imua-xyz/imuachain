package types

import (
	evmtype "github.com/evmos/evmos/v16/x/evm/types"
)

// DefaultGenesisState sets default evm genesis state with empty accounts and default params and
// chain config values.
func DefaultGenesisState() *evmtype.GenesisState {
	return &evmtype.GenesisState{
		Accounts: []evmtype.GenesisAccount{},
		Params:   ImuachainEvmDefaultParams(),
	}
}
