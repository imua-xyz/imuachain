package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/assets/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	if err := k.SetParams(ctx, &data.Params); err != nil {
		panic(err)
	}
	// TODO(mm): is it possible to optimize / speed up this process?
	// client_chain.go
	for i := range data.ClientChains {
		info := data.ClientChains[i]
		if err := k.SetClientChainInfo(ctx, &info); err != nil {
			panic(errorsmod.Wrap(err, "failed to set client chain info"))
		}
	}
	// client_chain_asset.go
	for i := range data.Tokens {
		info := data.Tokens[i]
		if err := k.SetStakingAssetInfo(ctx, &info); err != nil {
			panic(errorsmod.Wrap(err, "failed to set staking asset info"))
		}
	}
	// staker_asset.go (deposits)
	// we set the assets state related to deposits
	// it constructs the stakerID and the assetID, which we have validated previously.
	// it checks that the deposited amount is not negative, which we have already done.
	// and that the asset is registered, which we have also already done.
	for _, deposit := range data.Deposits {
		stakerID := deposit.StakerID
		for _, depositsByStaker := range deposit.Deposits {
			assetID := depositsByStaker.AssetID
			info := depositsByStaker.Info
			infoAsChange := types.DeltaStakerSingleAsset(info)
			// set the deposited and free values for the staker
			if err := k.UpdateStakerAssetState(
				ctx, stakerID, assetID, infoAsChange,
			); err != nil {
				panic(errorsmod.Wrap(err, "failed to set deposit info"))
			}
		}
	}

	for _, assets := range data.OperatorAssets {
		for _, assetInfo := range assets.AssetsState {
			// #nosec G703 // already validated
			accAddress, _ := sdk.AccAddressFromBech32(assets.Operator)
			infoAsChange := types.DeltaOperatorSingleAsset(assetInfo.Info)
			err := k.UpdateOperatorAssetState(ctx, accAddress, assetInfo.AssetID, infoAsChange)
			if err != nil {
				panic(errorsmod.Wrap(err, "failed to set operator asset info"))
			}
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	res := types.GenesisState{}
	var err error
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get parameter").Error())
	}
	res.Params = *params

	res.ClientChains, err = k.GetAllClientChainInfo(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all client chains").Error())
	}

	res.Tokens, err = k.GetAllStakingAssetsInfo(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all staking assets info").Error())
	}

	res.Deposits, err = k.AllDeposits(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all deposits").Error())
	}

	res.OperatorAssets, err = k.AllOperatorAssets(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all assets info for the operators").Error())
	}
	return &res
}
