package keeper

import (
	"fmt"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationkeeper "github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	"github.com/ethereum/go-ethereum/common/hexutil"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AllDeposits
func (k Keeper) AllDeposits(ctx sdk.Context) (deposits []assetstype.DepositsByStaker, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakerAssetInfos)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]assetstype.DepositsByStaker, 0)
	var previousStakerID string
	for ; iterator.Valid(); iterator.Next() {
		var stateInfo assetstype.StakerAssetInfo
		k.cdc.MustUnmarshal(iterator.Value(), &stateInfo)
		keyList, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		stakerID, assetID := keyList[0], keyList[1]
		if previousStakerID != stakerID {
			depositsByStaker := assetstype.DepositsByStaker{
				StakerID: stakerID,
				Deposits: make([]assetstype.DepositByAsset, 0),
			}
			ret = append(ret, depositsByStaker)
		}
		index := len(ret) - 1
		ret[index].Deposits = append(ret[index].Deposits, assetstype.DepositByAsset{
			AssetID: assetID,
			Info:    stateInfo,
		})
		previousStakerID = stakerID
	}
	return ret, nil
}

func (k Keeper) GetStakerAssetInfos(ctx sdk.Context, stakerID string) (assetsInfo []assetstype.DepositByAsset, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakerAssetInfos)
	iterator := sdk.KVStorePrefixIterator(store, []byte(stakerID))
	defer iterator.Close()

	ret := make([]assetstype.DepositByAsset, 0)
	for ; iterator.Valid(); iterator.Next() {
		var stateInfo assetstype.StakerAssetInfo
		k.cdc.MustUnmarshal(iterator.Value(), &stateInfo)
		keyList, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		assetID := keyList[1]
		ret = append(ret, assetstype.DepositByAsset{
			AssetID: assetID,
			Info:    stateInfo,
		})
	}
	// add exo-native-token info
	info, err := k.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetstype.ExocoreAssetID)
	if err != nil {
		return nil, err
	}
	ret = append(ret, assetstype.DepositByAsset{
		AssetID: assetstype.ExocoreAssetID,
		Info:    *info,
	})
	return ret, nil
}

func (k Keeper) GetStakerSpecifiedAssetInfo(ctx sdk.Context, stakerID string, assetID string) (info *assetstype.StakerAssetInfo, err error) {
	if assetID == assetstype.ExocoreAssetID {
		stakerAddrStr, _, err := assetstype.ParseID(stakerID)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to parse stakerID")
		}
		stakerAccDecode, err := hexutil.Decode(stakerAddrStr)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to decode staker address")
		}
		stakerAcc := sdk.AccAddress(stakerAccDecode)
		balance := k.bk.GetBalance(ctx, stakerAcc, assetstype.ExocoreAssetDenom)
		info := &assetstype.StakerAssetInfo{
			TotalDepositAmount:        balance.Amount,
			WithdrawableAmount:        balance.Amount,
			PendingUndelegationAmount: math.NewInt(0),
		}

		delegationInfoRecords, err := k.dk.GetDelegationInfo(ctx, stakerID, assetID)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to GetDelegationInfo")
		}
		for operator, record := range delegationInfoRecords.DelegationInfos {
			operatorAssetInfo, err := k.GetOperatorSpecifiedAssetInfo(ctx, sdk.MustAccAddressFromBech32(operator), assetID)
			if err != nil {
				return nil, errorsmod.Wrap(err, "failed to GetOperatorSpecifiedAssetInfo")
			}
			// the `undelegatableTokens` are currently delegated tokens. they are post-slashing, if any is applied.
			// this is because slashing is applied to an operator's total amount, of which, the share of a staker is kept
			// unchanged.
			undelegatableTokens, err := delegationkeeper.TokensFromShares(record.UndelegatableShare, operatorAssetInfo.TotalShare, operatorAssetInfo.TotalAmount)
			if err != nil {
				return nil, errorsmod.Wrap(err, "failed to get shares from token")
			}
			// this amount is post-slashing, as explained above.
			info.TotalDepositAmount = info.TotalDepositAmount.Add(undelegatableTokens).Add(record.WaitUndelegationAmount)
			info.PendingUndelegationAmount = info.PendingUndelegationAmount.Add(record.WaitUndelegationAmount)
		}
		return info, nil
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakerAssetInfos)
	key := assetstype.GetJoinedStoreKey(stakerID, assetID)
	value := store.Get(key)
	if value == nil {
		return nil, errorsmod.Wrap(assetstype.ErrNoStakerAssetKey, fmt.Sprintf("the key is:%s", key))
	}
	// when there is a slashing, we do not modify `StakerAssetInfo`.
	// hence, all the amounts below are pre-slashing. however, when
	// an undelegation is matured, the post-slashing amount is added
	// to the withdrawable amount and the pre-slashed amount is removed
	// from the amount pending undelegation.
	// if a staker were to exit the system, they would leave behind
	// `TotalDepositAmount` == lifetime slashing amount.
	ret := assetstype.StakerAssetInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// UpdateStakerAssetState is used to update the staker asset state
// The input `changeAmount` represents the values that you want to add or decrease,using positive or negative values for increasing and decreasing,respectively. The function will calculate and update new state after a successful check.
// The function will be called when there is deposit or withdraw related to the specified staker.
func (k Keeper) UpdateStakerAssetState(
	ctx sdk.Context, stakerID string, assetID string, changeAmount assetstype.DeltaStakerSingleAsset,
) (info *assetstype.StakerAssetInfo, err error) {
	// get the latest state,use the default initial state if the state hasn't been stored
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakerAssetInfos)
	key := assetstype.GetJoinedStoreKey(stakerID, assetID)
	assetState := assetstype.StakerAssetInfo{
		TotalDepositAmount:        math.NewInt(0),
		WithdrawableAmount:        math.NewInt(0),
		PendingUndelegationAmount: math.NewInt(0),
	}
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &assetState)
	}
	// update all states of the specified restaker asset
	err = assetstype.UpdateAssetValue(&assetState.TotalDepositAmount, &changeAmount.TotalDepositAmount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "UpdateStakerAssetState TotalDepositAmount error")
	}
	err = assetstype.UpdateAssetValue(&assetState.WithdrawableAmount, &changeAmount.WithdrawableAmount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "UpdateStakerAssetState CanWithdrawAmountOrWantChangeValue error")
	}
	err = assetstype.UpdateAssetValue(&assetState.PendingUndelegationAmount, &changeAmount.PendingUndelegationAmount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "UpdateStakerAssetState WaitUndelegationAmountOrWantChangeValue error")
	}

	// store the updated state
	bz := k.cdc.MustMarshal(&assetState)
	store.Set(key, bz)

	// emit event with new amount.
	// the indexer can pick this up and update the staker's asset state
	// without needing to know the prior state. it can also use the
	// event type to index a deposit or withdrawal history.
	// this event is only emitted here; callers of this function with
	// other side effects may emit events dedicated to those side effects
	// in addition to this event.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			assetstype.EventTypeUpdatedStakerAsset,
			sdk.NewAttribute(
				assetstype.AttributeKeyStakerID, stakerID,
			),
			sdk.NewAttribute(
				assetstype.AttributeKeyAssetID, assetID,
			),
			sdk.NewAttribute(
				assetstype.AttributeKeyDepositAmount, assetState.TotalDepositAmount.String(),
			),
			sdk.NewAttribute(
				assetstype.AttributeKeyWithdrawableAmount, assetState.WithdrawableAmount.String(),
			),
			sdk.NewAttribute(
				assetstype.AttributeKeyPendingUndelegationAmount, assetState.PendingUndelegationAmount.String(),
			),
		),
	)

	return &assetState, nil
}

func (k Keeper) GetStakerBalanceByAsset(ctx sdk.Context, stakerID string, assetID string) (balance assetstype.StakerBalance, err error) {
	stakerAssetInfo, err := k.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
	if err != nil {
		return assetstype.StakerBalance{}, err
	}

	delegatedAmount, err := k.dk.TotalDelegatedAmountForStakerAsset(ctx, stakerID, assetID)
	if err != nil {
		return assetstype.StakerBalance{}, err
	}

	totalBalance := stakerAssetInfo.WithdrawableAmount.Add(stakerAssetInfo.PendingUndelegationAmount).Add(delegatedAmount)

	balance = assetstype.StakerBalance{
		StakerID:           stakerID,
		AssetID:            assetID,
		Balance:            totalBalance.BigInt(),
		Withdrawable:       stakerAssetInfo.WithdrawableAmount.BigInt(),
		Delegated:          delegatedAmount.BigInt(),
		PendingUndelegated: stakerAssetInfo.PendingUndelegationAmount.BigInt(),
		TotalDeposited:     stakerAssetInfo.TotalDepositAmount.BigInt(),
	}

	return balance, nil
}
