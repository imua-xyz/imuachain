package keeper

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	types "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrationParams(ctx sdk.Context, k keeper.Keeper) {
	pV1 := k.GetV1Params(ctx)
	var pV2 types.Params
	if len(pV1.Chains) < 1 {
		panic("params.Chain should not be empty when do migrations")
	}

	pV2.Chains = pV1.Chains
	pV2.Tokens = pV1.Tokens
	pV2.Sources = pV1.Sources
	pV2.Rules = pV1.Rules
	pV2.TokenFeeders = pV1.TokenFeeders
	pV2.MaxNonce = pV1.MaxNonce
	pV2.ThresholdA = pV1.ThresholdA
	pV2.ThresholdB = pV1.ThresholdB
	pV2.Mode = types.ConsensusMode(pV1.Mode)
	pV2.MaxSizePrices = pV1.MaxSizePrices

	// this is the new field(break change) added to params
	pV2.NewMigrationInfo = "params upgraded to new structure"

	// save new params into kvstore
	k.SetParams(ctx, pV2)
}
