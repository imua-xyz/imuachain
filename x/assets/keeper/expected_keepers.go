package keeper

import (
	sdkmath "cosmossdk.io/math"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// this keeper interface is defined here to avoid a circular dependency
type delegationKeeper interface {
	GetDelegationInfo(ctx sdk.Context, stakerID, assetID string) (*delegationtype.QueryDelegationInfoResponse, error)
	TotalDelegatedAmountForStakerAsset(ctx sdk.Context, stakerID, assetID string) (amount sdkmath.Int, err error)
}
