package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/immint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) EpochMintInfo(goCtx context.Context, req *types.QueryEpochMintInfoRequest) (*types.QueryEpochMintInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	epochMintAmount, annualInflation, err := k.GetEpochMintInfo(ctx)
	if err != nil {
		return nil, err
	}
	if annualInflation.IsNil() {
		params := k.GetParams(ctx)
		epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, params.EpochIdentifier)
		if !exist {
			return nil, types.ErrInvalidParams.Wrapf("invalid epoch identifier:%s", params.EpochIdentifier)
		}
		epochDurationSeconds := int64(epochInfo.Duration.Seconds())
		if epochDurationSeconds <= 0 {
			return nil, types.ErrInvalidParams.Wrapf("invalid epoch duration: %v", epochInfo.Duration)
		}
		epochNumberInYear := SecondsInYear / epochDurationSeconds
		// calculate the annual inflation ratio
		totalSupply := k.bankKeeper.GetSupply(ctx, params.MintDenom).Amount
		annualProvisions := epochMintAmount.Mul(sdk.NewInt(epochNumberInYear))
		annualInflation = sdk.NewDecFromInt(annualProvisions).QuoInt(totalSupply)
	}
	return &types.QueryEpochMintInfoResponse{
		EpochMintInfo: types.EpochMintInfo{
			EpochMintAmount: epochMintAmount,
			AnnualInflation: annualInflation,
		},
	}, nil
}
