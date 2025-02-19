package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
)

// EndBlock : Completes expired pending undelegation events based on epoch information.
// This function is triggered at the end of every block. It queries the completable
// pending undelegations and completes them.
// We use EndBlock instead of epoch hooks to trigger completion because we want
// expired pending undelegations that are still held to be completed per block,
// rather than per epoch.
// Another reason is that we use a custom NullEpoch to handle pending undelegations that
// are not restricted by any unbonding duration. These undelegations are completed immediately
// at the end of the block after the transaction is committed on-chain. If we were to use an
// epochHook, we might need to consider using minutes as the default unbonding duration for
// these undelegations. However, such an implementation would mandate that the system enable
// minute-based epochs.
func (k *Keeper) EndBlock(
	originalCtx sdk.Context, _ abci.RequestEndBlock,
) []abci.ValidatorUpdate {
	logger := k.Logger(originalCtx)
	records, err := k.GetCompletableUndelegations(originalCtx)
	if err != nil {
		// When encountering an error while retrieving pending undelegation, skip the undelegation at the given height without causing the node to stop running.
		logger.Error("Error in GetCompletableUndelegations during the delegation's EndBlock execution", "error", err)
		return []abci.ValidatorUpdate{}
	}
	if len(records) == 0 {
		return []abci.ValidatorUpdate{}
	}
	for i := range records {
		record := records[i] // avoid implicit memory aliasing
		cc, writeCache := originalCtx.CacheContext()
		// we can use `Must` here because we stored this record ourselves.
		operatorAccAddress := sdk.MustAccAddressFromBech32(record.OperatorAddr)
		// TODO check if the operator has been slashed or frozen

		recordAmountNeg := record.Amount.Neg()
		// update delegation state
		deltaAmount := &types.DeltaDelegationAmounts{
			WaitUndelegationAmount: recordAmountNeg,
		}
		_, err = k.UpdateDelegationState(cc, record.StakerId, record.AssetId, record.OperatorAddr, deltaAmount)
		if err != nil {
			logger.Error("Error in UpdateDelegationState during the delegation's EndBlock execution", "error", err)
			continue
		}

		// update the staker state
		if record.AssetId == assetstypes.ImuachainAssetID {
			stakerAddrHex, _, err := assetstypes.ParseID(record.StakerId)
			if err != nil {
				logger.Error(
					"failed to parse staker ID",
					"error", err,
				)
				continue
			}
			stakerAddrBytes, err := hexutil.Decode(stakerAddrHex)
			if err != nil {
				logger.Error(
					"failed to decode staker address",
					"error", err,
				)
				continue
			}
			stakerAddr := sdk.AccAddress(stakerAddrBytes)
			if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(
				cc, types.DelegatedPoolName, stakerAddr,
				sdk.NewCoins(
					sdk.NewCoin(assetstypes.ImuachainAssetDenom, record.ActualCompletedAmount),
				),
			); err != nil {
				logger.Error(
					"failed to undelegate coins from module to account",
					"error", err,
				)
				continue
			}
		} else {
			_, err = k.assetsKeeper.UpdateStakerAssetState(cc, record.StakerId, record.AssetId, assetstypes.DeltaStakerSingleAsset{
				WithdrawableAmount:        record.ActualCompletedAmount,
				PendingUndelegationAmount: recordAmountNeg,
			})
			if err != nil {
				logger.Error("Error in UpdateStakerAssetState during the delegation's EndBlock execution", "error", err)
				continue
			}
		}

		// update the operator state
		err = k.assetsKeeper.UpdateOperatorAssetState(cc, operatorAccAddress, record.AssetId, assetstypes.DeltaOperatorSingleAsset{
			PendingUndelegationAmount: recordAmountNeg,
		})
		if err != nil {
			logger.Error("Error in UpdateOperatorAssetState during the delegation's EndBlock execution", "error", err)
			continue
		}

		// delete the Undelegation records that have been completed
		err = k.DeleteUndelegationRecord(cc, record)
		if err != nil {
			logger.Error("Error in DeleteUndelegationRecord during the delegation's EndBlock execution", "error", err)
			continue
		}
		// when calling `writeCache`, events are automatically emitted on the parent context
		writeCache()
	}
	return []abci.ValidatorUpdate{}
}
