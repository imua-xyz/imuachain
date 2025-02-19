package keeper

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(
	ctx sdk.Context,
	genState types.GenesisState,
) []abci.ValidatorUpdate {
	// the `params` validator is not super useful to validate state level information
	// so, it must be done here. by extension, the `InitGenesis` of the epochs module
	// should be called before that of this module.
	epochID := genState.Params.EpochIdentifier
	epochInfo, found := k.epochsKeeper.GetEpochInfo(ctx, epochID)
	if !found {
		// the panic is suitable here because it is being done at genesis, when the node
		// is not running. it means that the genesis file is malformed.
		panic(fmt.Sprintf("epoch info not found %s", epochID))
	}
	// apply the same logic to the staking assets.
	for _, assetID := range genState.Params.AssetIDs {
		if !k.restakingKeeper.IsStakingAsset(ctx, assetID) {
			panic(fmt.Errorf("staking param %s not found in assets module", assetID))
		}
	}
	k.SetParams(ctx, genState.Params)
	// create the AVS
	var avsAddr common.Address
	var err error
	// the avs module will remove the revision by itself, but we do it here anyway because we need it
	// to look up operator registration status after this - which is keyed by chainID without revision.
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	if avsAddr, err = k.avsKeeper.RegisterAVSWithChainID(ctx, &avstypes.AVSRegisterOrDeregisterParams{
		AvsName:           chainIDWithoutRevision,
		AssetID:           genState.Params.AssetIDs,
		UnbondingPeriod:   uint64(genState.Params.EpochsUntilUnbonded),
		MinSelfDelegation: genState.Params.MinSelfDelegation.Uint64(),
		EpochIdentifier:   epochID,
		ChainID:           chainIDWithoutRevision,
	}); err != nil {
		panic(fmt.Errorf("could not create the dogfood AVS: %s", err))
	}
	avsAddrString := avsAddr.String()
	k.Logger(ctx).Info(
		"created dogfood avs",
		"avsAddrString", avsAddrString,
		"chainIDWithoutRevision", chainIDWithoutRevision,
	)
	// create the validators
	out := make([]abci.ValidatorUpdate, 0, len(genState.ValSet))
	for _, val := range genState.ValSet {
		// wrappedKey can never be nil
		wrappedKey := operatortypes.NewWrappedConsKeyFromHex(val.PublicKey)
		// check that an operator exists
		if found, _ := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
			ctx, chainIDWithoutRevision, wrappedKey.ToConsAddr(),
		); !found {
			panic(fmt.Sprintf("operator not found for key %s", val.PublicKey))
		}
		out = append(out, abci.ValidatorUpdate{
			PubKey: *wrappedKey.ToTmProtoKey(),
			Power:  val.Power,
		})
	}
	for i := range genState.OptOutExpiries {
		obj := genState.OptOutExpiries[i]
		epoch := obj.Epoch
		if epoch < epochInfo.CurrentEpoch {
			panic(fmt.Sprintf("epoch %d is in the past", epoch))
		}
		for _, addr := range obj.OperatorAccAddrs {
			// #nosec G703 // already validated
			operatorAddr, _ := sdk.AccAddressFromBech32(addr)
			k.AppendOptOutToFinish(ctx, epoch, operatorAddr)
			k.SetOperatorOptOutFinishEpoch(ctx, operatorAddr, epoch)
		}
	}
	for i := range genState.ConsensusAddrsToPrune {
		obj := genState.ConsensusAddrsToPrune[i]
		epoch := obj.Epoch
		if epoch < epochInfo.CurrentEpoch {
			panic(fmt.Sprintf("epoch %d is in the past", epoch))
		}
		for _, addr := range obj.ConsAddrs {
			// #nosec G703 // already validated
			accAddr, _ := sdk.ConsAddressFromBech32(addr)
			k.AppendConsensusAddrToPrune(ctx, epoch, accAddr)
		}
	}
	for i := range genState.UndelegationMaturities {
		obj := genState.UndelegationMaturities[i]
		epoch := obj.Epoch
		if epoch < epochInfo.CurrentEpoch {
			panic(fmt.Sprintf("epoch %d is in the past", epoch))
		}
		for _, recordKey := range obj.UndelegationRecordKeys {
			// #nosec G703 // already validated
			recordKeyBytes, _ := hexutil.Decode(recordKey)
			k.AppendUndelegationToMature(ctx, epoch, recordKeyBytes)
			k.SetUndelegationMaturityEpoch(ctx, recordKeyBytes, epoch)
		}
	}
	// ApplyValidatorChanges only gets changes and hence the vote power must be set here.
	k.SetLastTotalPower(ctx, genState.LastTotalPower)

	// ApplyValidatorChanges will sort it internally
	return k.ApplyValidatorChanges(
		ctx, out,
	)
}

// ExportGenesis returns the module's exported genesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetDogfoodParams(ctx)
	validators := []types.GenesisValidator{}
	k.IterateBondedValidatorsByPower(ctx, func(_ int64, val stakingtypes.ValidatorI) bool {
		// #nosec G703 // already validated
		pubKey, _ := val.ConsPubKey()
		// #nosec G703 // already validated
		convKey, _ := cryptocodec.ToTmPubKeyInterface(pubKey)
		validators = append(validators,
			types.GenesisValidator{
				PublicKey: hexutil.Encode(convKey.Bytes()),
				Power:     val.GetConsensusPower(sdk.DefaultPowerReduction),
			},
		)
		return false // stop == false => continue iteration
	})
	return types.NewGenesis(
		k.GetDogfoodParams(ctx),
		validators,
		k.GetAllOptOutsToFinish(ctx),
		k.GetAllConsAddrsToPrune(ctx),
		k.GetAllUndelegationsToMature(ctx),
		k.GetLastTotalPower(ctx),
	)
}
