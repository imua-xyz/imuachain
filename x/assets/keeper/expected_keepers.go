package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
)

// this keeper interface is defined here to avoid a circular dependency
type delegationKeeper interface {
	GetDelegationInfo(ctx sdk.Context, stakerID, assetID string) (*delegationtype.QueryDelegationInfoResponse, error)
	TotalDelegatedAmountForStakingAsset(ctx sdk.Context, stakerID, assetID string) (amount sdkmath.Int, err error)
	GetPendingUndelegationFinalAmount(ctx sdk.Context, stakerID, assetID string) (sdkmath.Int, error)
}
