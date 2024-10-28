package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/ExocoreNetwork/exocore/utils"
	"github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
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

	// validate the existence of the epoch (stateful)
	epochIdentifier := req.Params.EpochIdentifier
	_, found := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !found {
		return &types.MsgUpdateParamsResponse{}, errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s", epochIdentifier))
	}
	k.SetParams(ctx, req.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
