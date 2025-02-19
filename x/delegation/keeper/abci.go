package keeper

import (
	"strings"

	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EndBlock : completed Undelegation events according to the canCompleted blockHeight
// This function will be triggered at the end of every block, it will query the undelegation state to get the records that need to be handled and try to complete the undelegation task.
func (k *Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	records, err := k.GetPendingUndelegationRecords(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		panic(err)
	}
	if len(records) == 0 {
		return []abci.ValidatorUpdate{}
	}
	for _, record := range records {
		// check if the operator has been slashed or frozen
		operatorAccAddress := sdk.MustAccAddressFromBech32(record.OperatorAddr)
		// todo: don't think about freezing the operator in current implementation
		/*		if k.slashKeeper.IsOperatorFrozen(ctx, operatorAccAddress) {
				// reSet the completed height if the operator is frozen
				record.CompleteBlockNumber = k.operatorKeeper.GetUnbondingExpirationBlockNumber(ctx, operatorAccAddress, record.BlockNumber)
				if record.CompleteBlockNumber <= uint64(ctx.BlockHeight()) {
					panic(fmt.Sprintf("the reset completedHeight isn't in future,setHeight:%v,curHeight:%v", record.CompleteBlockNumber, ctx.BlockHeight()))
				}
				_, innerError = k.SetSingleUndelegationRecord(ctx, record)
				if innerError != nil {
					panic(innerError)
				}
				continue
			}*/

		recordID := types.GetUndelegationRecordKey(record.BlockNumber, record.LzTxNonce, record.TxHash, record.OperatorAddr)
		if k.GetUndelegationHoldCount(ctx, recordID) > 0 {
			// store it again with the next block and move on
			// #nosec G701
			record.CompleteBlockNumber = uint64(ctx.BlockHeight()) + 1
			// we need to store two things here: one is the updated record in itself
			recordKey, err := k.SetSingleUndelegationRecord(ctx, record)
			if err != nil {
				panic(err)
			}
			// and the other is the fact that it matures at the next block
			if err = k.StorePendingUndelegationRecord(ctx, recordKey, record); err != nil {
				// TODO: remove previous index with currentHeight for pendingUndelegationRecord
				panic(err)
			}
			continue
		}
		// TODO(mike): ensure that operator is required to perform self delegation to match above.

		recordAmountNeg := record.Amount.Neg()
		// update delegation state
		deltaAmount := &types.DeltaDelegationAmounts{
			WaitUndelegationAmount: recordAmountNeg,
		}
		_, err = k.UpdateDelegationState(ctx, record.StakerID, record.AssetID, record.OperatorAddr, deltaAmount)
		if err != nil {
			// todo: using cached context to remove the panic
			panic(err)
		}

		// update the staker state
		if record.AssetID == assetstypes.ExocoreAssetID {
			parsedStakerID := strings.Split(record.StakerID, "_")
			stakerAddr := sdk.AccAddress(hexutil.MustDecode(parsedStakerID[0]))
			if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.DelegatedPoolName, stakerAddr, sdk.NewCoins(sdk.NewCoin(assetstypes.ExocoreAssetDenom, record.ActualCompletedAmount))); err != nil {
				panic(err)
			}
		} else {
			err = k.assetsKeeper.UpdateStakerAssetState(ctx, record.StakerID, record.AssetID, assetstypes.DeltaStakerSingleAsset{
				WithdrawableAmount:        record.ActualCompletedAmount,
				PendingUndelegationAmount: recordAmountNeg,
			})
			if err != nil {
				panic(err)
			}
		}

		// update the operator state
		err = k.assetsKeeper.UpdateOperatorAssetState(ctx, operatorAccAddress, record.AssetID, assetstypes.DeltaOperatorSingleAsset{
			PendingUndelegationAmount: recordAmountNeg,
		})
		if err != nil {
			panic(err)
		}

		// delete the Undelegation records that have been complemented
		err = k.DeleteUndelegationRecord(ctx, record)
		if err != nil {
			panic(err)
		}
	}
	return []abci.ValidatorUpdate{}
}
