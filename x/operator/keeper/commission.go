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
	// check last update time
	if ctx.BlockTime().Sub(operatorInfo.Commission.UpdateTime) < k.GetMinCommissionUpdateInterval(ctx) {
		return stakingtypes.ErrCommissionUpdateTime.Wrapf(
			"commission update time is less than the minimum commission update interval: %s < %s",
			ctx.BlockTime().Sub(operatorInfo.Commission.UpdateTime).String(),
			k.GetMinCommissionUpdateInterval(ctx).String(),
		)
	}
	// now validate the entire commission. we do not use the `ValidateNewRate`
	// method because that has the duration of 24h hardcoded.
	commission := stakingtypes.NewCommission(rate, operatorInfo.Commission.CommissionRates.MaxRate, operatorInfo.Commission.CommissionRates.MaxChangeRate)
	if err := commission.Validate(); err != nil {
		return err
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
