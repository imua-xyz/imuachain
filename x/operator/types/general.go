package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DeltaOptedInAssetState This is a struct to describe the desired change that matches with the OptedInAssetState
type (
	DeltaOptedInAssetState OptedInAssetState
	DeltaOperatorUSDInfo   OperatorOptedUSDValue
)

type OperatorStakingInfo struct {
	Staking                 sdkmath.LegacyDec
	SelfStaking             sdkmath.LegacyDec
	StakingAndWaitUnbonding sdkmath.LegacyDec
}

type SlashInputInfo struct {
	IsDogFood        bool
	Power            int64
	SlashType        uint32
	Operator         sdk.AccAddress
	AVSAddr          string
	SlashContract    string
	SlashID          string
	SlashEventHeight int64
	SlashProportion  sdkmath.LegacyDec
}
