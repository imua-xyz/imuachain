package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

type OracleKeeper interface {
	GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (oracletypes.Price, error)
	RegisterNewTokenAndSetTokenFeeder(ctx sdk.Context, oInfo *oracletypes.OracleInfo) error
	UpdateNSTValidatorListForStaker(ctx sdk.Context, chainID, stakerAddr, validatorPubkey string, amount sdkmath.Int) error
}

type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}
