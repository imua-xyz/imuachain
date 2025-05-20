package types

import (
	errorsmod "cosmossdk.io/errors"
)

func (p Params) Validate() error {
	if p.InstantUndelegationPenalty > 100 {
		return errorsmod.Wrap(ErrInvalidParams, "instant undelegation penalty cannot be greater than 100")
	}
	return nil
}
