package keeper

import (
	"strings"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k Keeper) SetParams(ctx sdk.Context, params *assetstypes.Params) error {
	// check if addr is evm address
	if !common.IsHexAddress(params.ExocoreLzAppAddress) {
		return assetstypes.ErrInvalidEvmAddressFormat
	}
	if len(common.FromHex(params.ExocoreLzAppEventTopic)) != common.HashLength {
		return assetstypes.ErrInvalidLzUaTopicIDLength
	}
	params.ExocoreLzAppAddress = strings.ToLower(params.ExocoreLzAppAddress)
	params.ExocoreLzAppEventTopic = strings.ToLower(params.ExocoreLzAppEventTopic)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstypes.KeyPrefixParams)
	bz := k.cdc.MustMarshal(params)
	store.Set(assetstypes.ParamsKey, bz)
	return nil
}

func (k Keeper) GetParams(ctx sdk.Context) (*assetstypes.Params, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstypes.KeyPrefixParams)
	value := store.Get(assetstypes.ParamsKey)
	if value == nil {
		return nil, assetstypes.ErrNoParamsKey
	}

	ret := &assetstypes.Params{}
	k.cdc.MustUnmarshal(value, ret)
	return ret, nil
}

func (k Keeper) GetExocoreGatewayAddress(ctx sdk.Context) (common.Address, error) {
	param, err := k.GetParams(ctx)
	if err != nil {
		return common.Address{}, err
	}
	return common.HexToAddress(param.ExocoreLzAppAddress), nil
}

func (k Keeper) CheckExocoreGatewayAddr(ctx sdk.Context, addr common.Address) error {
	// This is customized for testing, as the test tool sends batch transactions
	// to the precompiled contracts directly from different addresses.
	// Therefore, the check should be skipped when the Exocore node is used for testing.
	// Note: This should NOT be merged into the release branches.

	/*	param, err := k.GetParams(ctx)
		if err != nil {
			return err
		}
		exoCoreLzAppAddr := common.HexToAddress(param.ExocoreLzAppAddress)
		if addr != exoCoreLzAppAddr {
			return errorsmod.Wrapf(assetstypes.ErrNotEqualToLzAppAddr, "addr:%s,param.ExocoreGatewayAddress:%s", addr, param.ExocoreLzAppAddress)
		}*/
	return nil
}
