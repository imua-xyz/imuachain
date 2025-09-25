package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (info *OperatorInfo) ValidateBasic() error {
	// basic check 1
	if info == nil {
		return errorsmod.Wrap(
			ErrParameterInvalid, "ValidateBasic: info is nil",
		)
	}
	// basic check 2
	_, err := sdk.AccAddressFromBech32(info.EarningsAddr)
	if err != nil {
		return errorsmod.Wrap(
			err, "ValidateBasic: error occurred when parse acc address from Bech32",
		)
	}
	// do not allow empty operator info
	if info.OperatorMetaInfo == "" {
		return errorsmod.Wrap(
			ErrParameterInvalid, "ValidateBasic: operator meta info is empty",
		)
	}
	// do not allow operator info to exceed the maximum length
	if len(info.OperatorMetaInfo) > stakingtypes.MaxMonikerLength {
		return errorsmod.Wrapf(
			ErrParameterInvalid,
			"ValidateBasic: info length exceeds %d", stakingtypes.MaxMonikerLength,
		)
	}
	// do not allow empty approve address
	if info.ApproveAddr == "" {
		return errorsmod.Wrap(
			ErrParameterInvalid,
			"ValidateBasic: approve address is empty",
		)
	}
	// TODO(Chuang): should the approve address be bech32 validated?
	// first make sure none of these are nil; otherwise Validate will panic
	commission := info.Commission
	if commission.Rate.IsNil() || commission.MaxRate.IsNil() || commission.MaxChangeRate.IsNil() {
		return errorsmod.Wrap(
			ErrParameterInvalid,
			"ValidateBasic: commission rate is nil",
		)
	}
	if err := commission.Validate(); err != nil {
		return errorsmod.Wrap(err, "ValidateBasic: invalid commission rate")
	}
	return nil
}
