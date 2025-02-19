package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// func (k Keeper) PricesAll(goCtx context.Context, req *types.QueryAllPricesRequest) (*types.QueryAllPricesResponse, error) {
//	if req == nil {
//		return nil, status.Error(codes.InvalidArgument, "invalid request")
//	}
//
//	var pricess []types.Prices
//	ctx := sdk.UnwrapSDKContext(goCtx)
//
//	store := ctx.KVStore(k.storeKey)
//	pricesStore := prefix.NewStore(store, types.KeyPrefix(types.PricesKeyPrefix))
//	pricesTokenStore := prefix.NewStore(pricesStore, types.PricesKey(tokenID))
//
//	pageRes, err := query.Paginate(pricesTokenStore, req.Pagination, func(key []byte, value []byte) error {
//		var prices types.Prices
//		if err := k.cdc.Unmarshal(value, &prices); err != nil {
//			return err
//		}
//
//		pricess = append(pricess, prices)
//		return nil
//	})
//
//	if err != nil {
//		return nil, status.Error(codes.Internal, err.Error())
//	}
//
//	return &types.QueryAllPricesResponse{Prices: pricess, Pagination: pageRes}, nil
//}

// Prices return all prices for a specific token TODO: pagination
func (k Keeper) Prices(goCtx context.Context, req *types.QueryGetPricesRequest) (*types.QueryGetPricesResponse, error) {
	if req == nil || req.TokenId < 1 {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetPrices(
		ctx,
		req.TokenId,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetPricesResponse{Prices: val}, nil
}

// LatestPrice return the latest price for a specific token
func (k Keeper) LatestPrice(goCtx context.Context, req *types.QueryGetLatestPriceRequest) (*types.QueryGetLatestPriceResponse, error) {
	if req == nil || req.TokenId < 1 {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetPriceTRLatest(ctx, req.TokenId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetLatestPriceResponse{Price: val}, nil
}
