package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

func NewGenesisState(p Params) *GenesisState {
	return &GenesisState{
		Params: p,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.FeePool.CommunityPool.Validate(); err != nil {
		return fmt.Errorf("invalid FeePool: %s", err)
	}
	// Check val address and Commission
	for _, e := range gs.ValidatorAccumulatedCommissions {
		_, err := sdk.AccAddressFromBech32(e.ValAddr)
		if err != nil {
			return fmt.Errorf("invalid addr: %s", err)
		}
		err = e.Commission.Commission.Validate()
		if err != nil {
			return fmt.Errorf("invalid commission: %s", err)
		}
	}
	for _, e := range gs.ValidatorCurrentRewardsList {
		_, err := sdk.AccAddressFromBech32(e.ValAddr)
		if err != nil {
			return fmt.Errorf("invalid addr: %s", err)
		}
		err = e.CurrentRewards.Rewards.Validate()
		if err != nil {
			return fmt.Errorf("invalid Rewards: %s", err)
		}
	}
	for _, e := range gs.ValidatorOutstandingRewardsList {
		_, err := sdk.AccAddressFromBech32(e.ValAddr)
		if err != nil {
			return fmt.Errorf("invalid addr: %s", err)
		}
		err = e.OutstandingRewards.Rewards.Validate()
		if err != nil {
			return fmt.Errorf("invalid Rewards: %s", err)
		}
	}
	for _, e := range gs.StakerOutstandingRewardsList {

		err := e.StakerOutstandingRewards.Rewards.Validate()
		if err != nil {
			return fmt.Errorf("invalid Rewards: %s", err)
		}
	}
	return gs.Params.Validate()
}
