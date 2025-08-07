package types

import (
	errorsmod "cosmossdk.io/errors"
)

const DefaultInstantUnbondingPenalty = uint32(25)

// NewParams creates a new Params instance.
func NewParams(instantUnbondingPenalty uint32) Params {
	return Params{
		InstantUndelegationPenalty: instantUnbondingPenalty,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultInstantUnbondingPenalty)
}

func (p Params) Validate() error {
	if p.InstantUndelegationPenalty > 100 {
		return errorsmod.Wrap(ErrInvalidParams, "instant undelegation penalty cannot be greater than 100")
	}
	return nil
}
