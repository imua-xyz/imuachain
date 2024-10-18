package keeper

import (
	"context"

	"github.com/ExocoreNetwork/exocore/utils"
	"github.com/ExocoreNetwork/exocore/x/slash/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (k Keeper) UpdateParams(ctx context.Context, params *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	if utils.IsMainnet(c.ChainID()) && k.authority != params.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			k.authority, params.Authority,
		)
	}

	k.Logger(c).Info(
		"UpdateParams request",
		"authority", k.authority,
		"params.AUthority", params.Authority,
	)

	err := k.SetParams(c, &params.Params)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
