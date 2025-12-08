package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ValidatorMissCount(goCtx context.Context, req *types.QueryValidatorMissCountRequest) (*types.QueryValidatorMissCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if _, err := sdk.ConsAddressFromBech32(req.Validator); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid validator consensus address")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	info, found := k.GetValidatorReportInfo(ctx, req.Validator)
	if !found {
		return nil, status.Error(codes.NotFound, "validator report info not found")
	}
	params := k.GetParams(ctx)
	reportedRoundsWindow := params.Slashing.ReportedRoundsWindow

	return &types.QueryValidatorMissCountResponse{
		MissCount:  info.MissedRoundsCounter,
		WindowSize: reportedRoundsWindow,
		MaxMiss:    reportedRoundsWindow - params.Slashing.MinReportedPerWindow.MulInt64(reportedRoundsWindow).RoundInt64(),
	}, nil
}
