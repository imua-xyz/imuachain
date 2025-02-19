package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/ExocoreNetwork/exocore/utils"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		if updated, err := k.SetClientChainInfo(ctx, &info); err != nil {
			panic(errorsmod.Wrap(err, "failed to set client chain info"))
		} else if updated {
			// should not happen if validate-genesis has been called.
			panic(errorsmod.Wrapf(types.ErrInvalidGenesisData, "duplicate client chain found: %s", info.Name))
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
			// this will not emit an event.
			if _, err := k.UpdateStakerAssetState(
				ctx, stakerID, assetID, infoAsChange,
			); err != nil {
				panic(errorsmod.Wrap(err, "failed to set deposit info"))
			}
		}
	}

	exoDelegation := sdk.ZeroInt()
	for _, assets := range data.OperatorAssets {
		for _, assetInfo := range assets.AssetsState {
			// #nosec G703 // already validated
			accAddress, _ := sdk.AccAddressFromBech32(assets.Operator)
			infoAsChange := types.DeltaOperatorSingleAsset(assetInfo.Info)
			err := k.UpdateOperatorAssetState(ctx, accAddress, assetInfo.AssetID, infoAsChange)
			if err != nil {
				panic(errorsmod.Wrap(err, "failed to set operator asset info"))
			}
			if assetInfo.AssetID == types.ExocoreAssetID {
				exoDelegation = exoDelegation.Add(assetInfo.Info.TotalAmount)
			}
		}
	}

	// check that this exoDelegation amount is reflected in x/bank
	address := k.ak.GetModuleAccount(ctx, delegationtypes.DelegatedPoolName).GetAddress()
	balance := k.bk.GetBalance(ctx, address, utils.BaseDenom)
	// slashing is lazily applied at the time of withdrawal. it means that
	// exoDelegation may have been slashed, but balance will never be slashed.
	// it implies that exoDelegation should be less than or equal to balance.
	// or conversely, if exoDelegation is greater than balance, we have a problem.
	if exoDelegation.GT(balance.Amount) {
		panic(errorsmod.Wrapf(
			types.ErrInvalidGenesisData,
			"delegated pool account balance is too low, exoDelegation: %s, balance: %s",
			exoDelegation, balance.Amount,
		).Error())
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
