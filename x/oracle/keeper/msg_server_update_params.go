package keeper

import (
	"context"

	utils "github.com/ExocoreNetwork/exocore/utils"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/cache"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if utils.IsMainnet(ctx.ChainID()) && ms.Keeper.authority != msg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			ms.Keeper.authority, msg.Authority,
		)
	}

	ms.Keeper.Logger(ctx).Info(
		"UpdateParams request",
		"authority", ms.Keeper.authority,
		"params.AUthority", msg.Authority,
	)

	p := ms.GetParams(ctx)
	var err error
	defer func() {
		if err != nil {
			ms.Logger(ctx).Error("UpdateParams failed", "error", err)
		}
	}()
	height := uint64(ctx.BlockHeight())
	// add sources
	if p, err = p.AddSources(msg.Params.Sources...); err != nil {
		return nil, err
	}
	// add chains
	if p, err = p.AddChains(msg.Params.Chains...); err != nil {
		return nil, err
	}
	// add tokens
	if p, err = p.UpdateTokens(height, msg.Params.Tokens...); err != nil {
		return nil, err
	}
	// add rules
	if p, err = p.AddRules(msg.Params.Rules...); err != nil {
		return nil, err
	}
	// update max size of price
	if p, err = p.UpdateMaxPriceCount(msg.Params.MaxSizePrices); err != nil {
		return nil, err
	}
	// udpate tokenFeeders
	for _, tokenFeeder := range msg.Params.TokenFeeders {
		if p, err = p.UpdateTokenFeeder(tokenFeeder, height); err != nil {
			return nil, err
		}
	}
	// validate params
	if err = p.Validate(); err != nil {
		return nil, err
	}
	// set updated new params
	ms.SetParams(ctx, p)
	_ = GetAggregatorContext(ctx, ms.Keeper)
	cs.AddCache(cache.ItemP(p))
	return &types.MsgUpdateParamsResponse{}, nil
}
