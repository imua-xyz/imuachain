package keeper

import (
	"bytes"
	"fmt"

	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
)

// DelegateTo : It doesn't need to check the active status of the operator in middlewares when
// delegating assets to the operator. This is because it adds assets to the operator's amount.
// But it needs to check if operator has been slashed or frozen.
func (k Keeper) DelegateTo(ctx sdk.Context, params *delegationtype.DelegationOrUndelegationParams) (sdkmath.LegacyDec, sdkmath.Int, error) {
	return k.delegateTo(ctx, params, true)
}

// delegateTo is the internal private version of DelegateTo. if the notGenesis parameter is
// false, the operator keeper and the delegation hooks are not called.
func (k *Keeper) delegateTo(
	ctx sdk.Context,
	params *delegationtype.DelegationOrUndelegationParams,
	notGenesis bool,
) (sdkmath.LegacyDec, sdkmath.Int, error) {
	if !params.OpAmount.IsPositive() {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, delegationtype.ErrAmountIsNotPositive
	}
	// check if the delegatedTo address is an operator
	if !k.operatorKeeper.IsOperator(ctx, params.OperatorAddress) {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, errorsmod.Wrap(delegationtype.ErrOperatorNotExist, fmt.Sprintf("input operatorAddr is:%s", params.OperatorAddress))
	}

	// check if the operator has been slashed or frozen
	// skip the check if not notgenesis (or chain restart)
	if notGenesis && k.slashKeeper.IsOperatorFrozen(ctx, params.OperatorAddress) {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, delegationtype.ErrOperatorIsFrozen
	}

	var stakerID, assetID string
	if params.RewardAsset {
		// handle the reward asset delegation
		stakerID, assetID = params.RewardStakerID, params.RewardAssetID
		// The delegated rewards come from the distribution module,
		// so the rewards state will be updated there when this function is called.
	} else {
		stakerID, assetID = assetstype.GetStakerIDAndAssetID(params.ClientChainID, params.StakerAddress, params.AssetsAddress)
		if assetID != assetstype.ImuachainAssetID {
			// check if the staker asset has been deposited and the canWithdraw amount is bigger than the delegation amount
			info, err := k.assetsKeeper.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
			if err != nil {
				return sdkmath.LegacyDec{}, sdkmath.Int{}, err
			}

			if info.WithdrawableAmount.LT(params.OpAmount) {
				return sdkmath.LegacyDec{}, sdkmath.Int{}, errorsmod.Wrap(delegationtype.ErrDelegationAmountTooBig, fmt.Sprintf("the opAmount is:%s the WithdrawableAmount amount is:%s", params.OpAmount, info.WithdrawableAmount))
			}

			// update staker asset state
			_, err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, assetstype.DeltaStakerSingleAsset{
				WithdrawableAmount: params.OpAmount.Neg(),
			})
			if err != nil {
				return sdkmath.LegacyDec{}, sdkmath.Int{}, err
			}
		} else {
			coins := sdk.NewCoins(sdk.NewCoin(assetstype.ImuachainAssetDenom, params.OpAmount))
			// transfer the delegation amount from the staker account to the delegated pool
			if err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, params.StakerAddress, delegationtype.DelegatedPoolName, coins); err != nil {
				return sdkmath.LegacyDec{}, sdkmath.Int{}, err
			}
			// auto associate it, if there is a match. note that both are byte versions of bech32
			// AccAddress. there is no need to check for an existing association because:
			// (1) at this point, the `params.ClientChainID` is 0 and such a `stakerID` ending with
			// this clientChainID can not be associated with an operator using the standard
			// precompile method due to the `ClientChainExists` check.
			// (2) an existing association will be overwritten by the exact same association due to
			// the equality check below.
			if bytes.Equal(params.StakerAddress, params.OperatorAddress[:]) {
				// always returns nil.
				err := k.SetAssociatedOperator(ctx, stakerID, params.OperatorAddress.String())
				if err != nil {
					return sdkmath.LegacyDec{}, sdkmath.Int{}, err
				}
			}
			// this emitted event is not the total amount; it is the additional amount.
			// indexers must add it to the last known amount to get the total amount.
			// non-native case handled within UpdateStakerAssetState
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					delegationtype.EventTypeImuaAssetDelegation,
					sdk.NewAttribute(delegationtype.AttributeKeyStakerID, sdk.AccAddress(params.StakerAddress).String()),
					sdk.NewAttribute(delegationtype.AttributeKeyOperator, params.OperatorAddress.String()),
					sdk.NewAttribute(delegationtype.AttributeKeyAmount, params.OpAmount.String()),
				),
			)
		}
	}
	// calculate the share from the delegation amount
	share, err := k.CalculateShare(ctx, params.OperatorAddress, assetID, params.OpAmount)
	if err != nil {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, err
	}
	deltaOperatorAsset := assetstype.DeltaOperatorSingleAsset{
		TotalAmount: params.OpAmount,
		TotalShare:  share,
	}
	// Check if the staker belongs to the delegated operator. Increase the operator's share if yes.
	if k.GetAssociatedOperator(ctx, stakerID) == params.OperatorAddress.String() {
		deltaOperatorAsset.OperatorShare = share
	}

	prevAssetState, err := k.assetsKeeper.UpdateOperatorAssetState(ctx, params.OperatorAddress, assetID, deltaOperatorAsset)
	if err != nil {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, err
	}

	deltaAmount := &delegationtype.DeltaDelegationAmounts{}
	if params.RewardAsset {
		deltaAmount.RewardUndelegatableShare = share
	} else {
		deltaAmount.UndelegatableShare = share
	}
	_, preDelegationState, err := k.UpdateDelegationState(ctx, stakerID, assetID, params.OperatorAddress.String(), deltaAmount)
	if err != nil {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, err
	}
	err = k.AppendStakerForOperator(ctx, params.OperatorAddress.String(), assetID, stakerID)
	if err != nil {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, err
	}

	// calculate the previous delegation amount
	totalDelegatableShare := preDelegationState.UndelegatableShare.Add(preDelegationState.RewardUndelegatableShare)
	preDelegatedAmount, err := TokensFromShares(totalDelegatableShare, prevAssetState.TotalShare, prevAssetState.TotalAmount)
	if err != nil {
		return sdkmath.LegacyDec{}, sdkmath.Int{}, err
	}
	if notGenesis {
		// call the hooks registered by the other modules
		err = k.Hooks().AfterDelegation(ctx, stakerID, assetID, params.OperatorAddress, preDelegatedAmount, prevAssetState)
		if err != nil {
			return sdkmath.LegacyDec{}, sdkmath.Int{}, err
		}
	}
	return share, preDelegatedAmount, nil
}

// UndelegateFrom handles normal and instant undelegation.
// The undelegation needs to consider whether the operator's opted-in assets can exit from the AVS.
// Because only after the operator has served the AVS can the staking asset be undelegated.
// So we use two steps to handle the undelegation. Fist,record the undelegation request and the corresponding exit time which needs to be obtained from the operator opt-in module. Then,we handle the record when the exit time has expired.
func (k *Keeper) UndelegateFrom(ctx sdk.Context, params *delegationtype.DelegationOrUndelegationParams) error {
	if !params.OpAmount.IsPositive() {
		return delegationtype.ErrAmountIsNotPositive
	}
	// check if the UndelegatedFrom address is an operator
	if !k.operatorKeeper.IsOperator(ctx, params.OperatorAddress) {
		return delegationtype.ErrOperatorNotExist
	}
	// get staker delegation state, then check the validation of Undelegation amount
	var stakerID, assetID string
	if params.RewardAsset {
		stakerID, assetID = params.RewardStakerID, params.RewardAssetID
	} else {
		stakerID, assetID = assetstype.GetStakerIDAndAssetID(params.ClientChainID, params.StakerAddress, params.AssetsAddress)
	}

	// verify the undelegation amount
	share, err := k.ValidateUndelegationAmount(ctx, params.RewardAsset, params.OperatorAddress, stakerID, assetID, params.OpAmount)
	if err != nil {
		return err
	}

	// get the previous operator asset state before update
	prevAssetState, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, params.OperatorAddress, assetID)
	if err != nil {
		return err
	}
	preDelegationState, err := k.GetSingleDelegationInfo(ctx, stakerID, assetID, params.OperatorAddress.String())
	if err != nil {
		return err
	}
	// calculate the previous delegation amount
	totalDelegatableShare := preDelegationState.UndelegatableShare.Add(preDelegationState.RewardUndelegatableShare)
	preDelegatedAmount, err := TokensFromShares(totalDelegatableShare,
		prevAssetState.TotalShare, prevAssetState.TotalAmount)
	if err != nil {
		return err
	}

	// remove share
	removeToken, err := k.RemoveShare(ctx, true, params.RewardAsset, params.OperatorAddress, stakerID, assetID, share)
	if err != nil {
		return err
	}
	undelegationID := k.GetLastUndelegationID(ctx)
	// record Undelegation event
	r := delegationtype.UndelegationRecord{
		StakerId:              stakerID,
		AssetId:               assetID,
		OperatorAddr:          params.OperatorAddress.String(),
		TxHash:                params.TxHash.String(),
		UndelegationId:        undelegationID,
		BlockNumber:           uint64(ctx.BlockHeight()),
		Amount:                removeToken,
		ActualCompletedAmount: removeToken,
		RewardAsset:           params.RewardAsset,
		RewardUndelegations:   params.RewardUndelegations,
		InstantPenaltyAmount:  sdkmath.ZeroInt(),
	}

	var completedEpochID string
	var completedEpochNumber int64
	var applySlash bool
	instantSlashRatio := sdkmath.LegacyZeroDec()
	if params.InstantUnbonding {
		r.InstantUnbonding = params.InstantUnbonding
		applySlash, completedEpochID, completedEpochNumber, err = k.operatorKeeper.GetInstantUnbondingExpiration(ctx, params.OperatorAddress)
		if err != nil {
			return err
		}
		// Apply instant penalty
		if applySlash {
			// Get the instant undelegation penalty
			penalty := k.GetInstantUndelegationPenalty(ctx)
			instantSlashRatio = sdkmath.LegacyNewDec(int64(penalty)).QuoInt64(100)
			penaltyAmount := instantSlashRatio.MulInt(params.OpAmount).TruncateInt()
			r.ActualCompletedAmount = r.ActualCompletedAmount.Sub(penaltyAmount)
			r.ApplySlash = true
			r.InstantPenaltyAmount = penaltyAmount
		}
	} else {
		completedEpochID, completedEpochNumber, _, err = k.operatorKeeper.GetUnbondingExpiration(ctx, params.OperatorAddress)
		if err != nil {
			return err
		}
	}

	if params.RewardAsset {
		if params.ReduceDelegationShare == nil {
			return delegationtype.ErrInvalidInputParameter.Wrap("ReduceDelegationShare function is required for reward undelegation")
		}
		// decrease the reward delegation share if it's an reward undelegation
		undelegationAmounts, _, err := params.ReduceDelegationShare(ctx, params.RewardStakerID, params.RewardAssetID, params.OperatorAddress, instantSlashRatio, params.OpAmount, *prevAssetState)
		if err != nil {
			return err
		}
		r.RewardUndelegations = undelegationAmounts
	}

	r.CompletedEpochIdentifier = completedEpochID
	r.CompletedEpochNumber = completedEpochNumber
	// the hold count is relevant to async AVSs instead of sync AVSs. for example, the dogfood AVS is sync since it
	// runs only on this chain. meanwhile, x/appchain-based AVSs are async because of the IBC's in-built communication
	// lag. the hold count is used to ensure that the undelegation is not processed until the AVS has completed its
	// unbonding period.
	// TODO: remove the hold count increment for x/dogfood AVS.
	err = k.SetUndelegationRecords(ctx, false, []delegationtype.UndelegationAndHoldCount{
		{
			Undelegation: &r,
		},
	})
	if err != nil {
		return err
	}
	err = k.IncrementLastUndelegationID(ctx)
	if err != nil {
		return err
	}

	recordKey := r.GetKey()
	// call the hooks registered by the other modules
	err = k.Hooks().AfterUndelegationStarted(ctx, stakerID, assetID, params.OperatorAddress,
		recordKey, preDelegatedAmount, *prevAssetState)
	if err != nil {
		return err
	}

	// emit an event to track the undelegation record identifiers.
	// for the ImuachainAssetID undelegation, this event is used to track asset state as well.
	// for other undelegations, it is instead tracked from the staker asset state.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeUndelegationStarted,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, r.StakerId),
			sdk.NewAttribute(delegationtype.AttributeKeyAssetID, r.AssetId),
			sdk.NewAttribute(delegationtype.AttributeKeyOperator, r.OperatorAddr),
			sdk.NewAttribute(delegationtype.AttributeKeyRecordID, hexutil.Encode(recordKey)),
			// the amount and ActualCompletedAmount are the same unless slashed, which does not happen within this function.
			sdk.NewAttribute(delegationtype.AttributeKeyAmount, r.Amount.String()),
			sdk.NewAttribute(delegationtype.AttributeKeyCompletedEpochID, r.CompletedEpochIdentifier),
			sdk.NewAttribute(delegationtype.AttributeKeyCompletedEpochNumber, fmt.Sprintf("%d", r.CompletedEpochNumber)),
			sdk.NewAttribute(delegationtype.AttributeKeyUndelegationID, fmt.Sprintf("%d", r.UndelegationId)),
			sdk.NewAttribute(delegationtype.AttributeKeyTxHash, params.TxHash.String()),
			sdk.NewAttribute(delegationtype.AttributeKeyBlockNumber, fmt.Sprintf("%d", r.BlockNumber)),
			sdk.NewAttribute(delegationtype.AttributeKeyInstantUnbonding, fmt.Sprintf("%t", params.InstantUnbonding)),
			sdk.NewAttribute(delegationtype.AttributeKeyApplyInstantSlash, fmt.Sprintf("%t", applySlash)),
		),
	)
	return nil
}

// AssociateOperatorWithStaker marks that a staker is claiming to be associated with an operator.
// In other words, the staker's delegations will be marked as self-delegations for the operator.
// Each stakerID can associate, at most, to one operator. To change that operator, the staker must
// first call DissociateOperatorFromStaker.
// However, each operator may be associated with multiple stakers.
// This function is not available for end users to call directly, and such calls must be made
// via the ImuachainGateway. The gateway associates the `msg.sender` of the call, along with the
// chain id, with the operator. Since it associates `msg.sender`, we do not need to check that
// this request is signed by the staker.
// TODO: It may be possible that an address, which is an EOA staker on a client chain, is a
// contract on Imuachain. When that happens, the contract may be able to call the Gateway to
// maliciously associate the staker with an operator. The probability of this, however, is
// infinitesimal (10^-60) so we will not do anything about it for now.
// The solution could be to require that such associations originate from the client chain.
func (k *Keeper) AssociateOperatorWithStaker(
	ctx sdk.Context,
	clientChainID uint64,
	operatorAddress sdk.AccAddress,
	stakerAddress []byte,
) error {
	if !k.assetsKeeper.ClientChainExists(ctx, clientChainID) {
		return delegationtype.ErrClientChainNotExist
	}
	if !k.operatorKeeper.IsOperator(ctx, operatorAddress) {
		return delegationtype.ErrOperatorNotExist
	}

	stakerID, _ := assetstype.GetStakerIDAndAssetID(clientChainID, stakerAddress, nil)
	if k.GetAssociatedOperator(ctx, stakerID) != "" {
		return delegationtype.ErrOperatorAlreadyAssociated
	}

	var err error
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		// increase the share of new marked operator
		if keys.OperatorAddr == operatorAddress.String() {
			_, err = k.assetsKeeper.UpdateOperatorAssetState(ctx, operatorAddress, keys.AssetId, assetstype.DeltaOperatorSingleAsset{
				OperatorShare: amounts.UndelegatableShare,
			})
		}
		if err != nil {
			return true, err
		}
		return false, nil
	}
	err = k.IterateDelegationsForStaker(ctx, stakerID, opFunc)
	if err != nil {
		return err
	}

	// update the marking information
	err = k.SetAssociatedOperator(ctx, stakerID, operatorAddress.String())
	if err != nil {
		return err
	}

	return nil
}

// DissociateOperatorFromStaker deletes the associative relationship between a staker
// (address + chain id combination) and an operator. See AssociateOperatorWithStaker for more
// information on the relationship and restrictions.
func (k *Keeper) DissociateOperatorFromStaker(
	ctx sdk.Context,
	clientChainID uint64,
	stakerAddress []byte,
) error {
	stakerID, _ := assetstype.GetStakerIDAndAssetID(clientChainID, stakerAddress, nil)
	associatedOperator := k.GetAssociatedOperator(ctx, stakerID)
	if associatedOperator == "" {
		return delegationtype.ErrNoAssociatedOperatorByStaker
	}
	oldOperatorAccAddr, err := sdk.AccAddressFromBech32(associatedOperator)
	if err != nil {
		return delegationtype.ErrOperatorAddrIsNotAccAddr
	}

	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		// decrease the share of old operator
		if keys.OperatorAddr == associatedOperator {
			_, err = k.assetsKeeper.UpdateOperatorAssetState(ctx, oldOperatorAccAddr, keys.AssetId, assetstype.DeltaOperatorSingleAsset{
				OperatorShare: amounts.UndelegatableShare.Neg(),
			})
		}
		if err != nil {
			return true, err
		}
		return false, nil
	}
	err = k.IterateDelegationsForStaker(ctx, stakerID, opFunc)
	if err != nil {
		return err
	}

	// delete the marking information
	err = k.DeleteAssociatedOperator(ctx, stakerID)
	if err != nil {
		return err
	}

	return nil
}
