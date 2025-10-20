package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// ValidateAndUpdateCommissionRate validates the commission rate and updates the operator info.
func (k Keeper) ValidateAndUpdateCommissionRate(
	ctx sdk.Context, addr sdk.AccAddress, rate sdk.Dec,
) error {
	operatorInfo, err := k.OperatorInfo(ctx, addr.String())
	if err != nil {
		return err
	}
	// check rate exceeds min commission rate
	minCommissionRate := k.GetMinCommissionRate(ctx)
	if rate.LT(minCommissionRate) {
		return stakingtypes.ErrCommissionLTMinRate.Wrapf(
			"commission rate is less than the minimum commission rate: %s < %s",
			rate.String(), minCommissionRate.String(),
		)
	}
	// check last update time - condition 1
	if ctx.BlockTime().Sub(operatorInfo.Commission.UpdateTime) < k.GetMinCommissionUpdateInterval(ctx) {
		return stakingtypes.ErrCommissionUpdateTime.Wrapf(
			"commission update time is less than the minimum commission update interval: %s < %s",
			ctx.BlockTime().Sub(operatorInfo.Commission.UpdateTime).String(),
			k.GetMinCommissionUpdateInterval(ctx).String(),
		)
	}
	// ease of access var
	commission := operatorInfo.Commission.CommissionRates
	// check rate is non-negative - condition 2
	if rate.IsNegative() {
		return stakingtypes.ErrCommissionNegative.Wrapf(
			"commission rate is negative: %s", rate.String(),
		)
	}
	// check rate is less than the max rate - condition 3
	if rate.GT(commission.MaxRate) {
		return stakingtypes.ErrCommissionGTMaxRate.Wrapf(
			"commission rate is greater than the max rate: %s > %s",
			rate.String(), commission.MaxRate.String(),
		)
	}
	// check rate change is less than the max change rate - condition 4
	if rate.Sub(commission.Rate).GT(commission.MaxChangeRate) {
		return stakingtypes.ErrCommissionGTMaxChangeRate.Wrapf(
			"commission change rate is greater than the max change rate: %s > %s",
			rate.Sub(commission.Rate).String(), commission.MaxChangeRate.String(),
		)
	}
	// finally, store it
	operatorInfo.Commission.CommissionRates.Rate = rate
	operatorInfo.Commission.UpdateTime = ctx.BlockTime()
	k.setOperatorInfo(ctx, addr, operatorInfo)
	// inform the indexer
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeEditOperator,
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, addr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyCommissionRate, rate.String()),
		),
	)
	return nil
}
