package types

import (
	"fmt"

	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (

	// DefaultEpochIdentifier is the epoch identifier which is used, by default, to identify the
	// epoch. Note that the options in the default genesis include minute, week, hour or day.
	DefaultEpochIdentifier = epochstypes.MinuteEpochID
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		EpochIdentifier: DefaultEpochIdentifier,
		CommunityTax:    sdk.NewDecWithPrec(3, 2),
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := epochstypes.ValidateEpochIdentifierInterface(p.EpochIdentifier); err != nil {
		return fmt.Errorf("epoch identifier: %w", err)
	}
	return nil
}
