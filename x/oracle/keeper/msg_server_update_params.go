package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	utils "github.com/imua-xyz/imuachain/utils"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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
	updated := false
	defer func() {
		if err != nil {
			ms.Logger(ctx).Error("UpdateParams failed", "error", err)
		}
	}()
	// #nosec G115 - blockHeight will never be negative
	height := uint64(ctx.BlockHeight())
	// add sources
	if p, err = p.AddSources(msg.Params.Sources...); err == nil {
		updated = true
	} else if err != types.ErrNoOp {
		return nil, err
	}
	// add chains
	if p, err = p.AddChains(msg.Params.Chains...); err == nil {
		updated = true
	} else if err != types.ErrNoOp {
		return nil, err
	}
	// add tokens
	if p, err = p.UpdateTokens(height, msg.Params.Tokens...); err == nil {
		updated = true
	} else if err != types.ErrNoOp {
		return nil, err
	}
	// add rules
	if p, err = p.AddRules(msg.Params.Rules...); err == nil {
		updated = true
	} else if err != types.ErrNoOp {
		return nil, err
	}
	// update max size of price
	if p, err = p.UpdateMaxPriceCount(msg.Params.MaxSizePrices); err == nil {
		updated = true
	} else if err != types.ErrNoOp {
		return nil, err
	}
	// udpate tokenFeeders
	for _, tokenFeeder := range msg.Params.TokenFeeders {
		if p, err = p.UpdateTokenFeeder(tokenFeeder, height); err == nil {
			updated = true
		} else if err != types.ErrNoOp {
			return nil, err
		}
	}

	if !updated {
		return nil, types.ErrNoOp
	}
	// validate params
	if err = p.Validate(); err != nil {
		return nil, err
	}
	// set updated new params
	ms.SetParams(ctx, p)
	if !ctx.IsCheckTx() {
		// mark params updated for memory cache
		ms.SetParamsUpdated()
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
