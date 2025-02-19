package keeper

import (
	"slices"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

func (k Keeper) SetParams(ctx sdk.Context, params *assetstypes.Params) error {
	// lower case all gateway addresses
	params.Normalize()
	// check if addr is evm address
	for _, gateway := range params.Gateways {
		if !common.IsHexAddress(gateway) {
			return assetstypes.ErrInvalidEvmAddressFormat
		}
	}
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

func (k Keeper) GetGatewayAddresses(ctx sdk.Context) ([]common.Address, error) {
	param, err := k.GetParams(ctx)
	if err != nil {
		return []common.Address{}, err
	}
	gateways := []common.Address{}
	for _, gateway := range param.Gateways {
		gateways = append(gateways, common.HexToAddress(gateway))
	}
	return gateways, nil
}

func (k Keeper) IsAuthorizedGateway(ctx sdk.Context, addr common.Address) (bool, error) {
	param, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	authorized := slices.Contains(param.Gateways, strings.ToLower(addr.Hex()))
	return authorized, nil
}
