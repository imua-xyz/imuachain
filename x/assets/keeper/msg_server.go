package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

var _ assetstype.MsgServer = &Keeper{}

// UpdateParams This function should be triggered by the governance in the future
func (k Keeper) UpdateParams(ctx context.Context, params *assetstype.MsgUpdateParams) (*assetstype.MsgUpdateParamsResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	if utils.IsMainnet(c.ChainID()) && k.authority != params.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			k.authority, params.Authority,
		)
	}
	c.Logger().Info(
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
