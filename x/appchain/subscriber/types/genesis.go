package types

import (
	commontypes "github.com/imua-xyz/imuachain/x/appchain/common/types"
)

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return NewGenesis(commontypes.DefaultSubscriberParams())
}

// NewGenesis creates a new genesis state with the provided parameters and
// data.
func NewGenesis(params commontypes.SubscriberParams) *GenesisState {
	return &GenesisState{Params: params}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	return nil
}
