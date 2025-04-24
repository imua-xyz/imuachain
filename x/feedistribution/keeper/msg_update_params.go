package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/imua-xyz/imuachain/utils"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k Keeper) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if utils.IsMainnet(ctx.ChainID()) && k.authority != req.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			k.authority, req.Authority,
		)
	}

	k.Logger().Info(
		"UpdateParams request",
		"authority", k.authority,
		"params.Authority", req.Authority,
	)

	if err := req.Params.Validate(); err != nil {
		return &types.MsgUpdateParamsResponse{}, err
	}
	k.SetParams(ctx, req.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
