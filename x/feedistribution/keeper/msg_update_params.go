package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	// For test purpose, skip for now
	// if k.GetAuthority() != req.Authority {
	// 	 return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	// }

	ctx := sdk.UnwrapSDKContext(goCtx)
	epochIdentifier := req.Params.EpochIdentifier

	_, found := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !found {
		return &types.MsgUpdateParamsResponse{}, errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s", epochIdentifier))
	}
	k.SetParams(ctx, req.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
