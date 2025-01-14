package keeper

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DelegateTo : It doesn't need to check the active status of the operator in middlewares when
// delegating assets to the operator. This is because it adds assets to the operator's amount.
// But it needs to check if operator has been slashed or frozen.
func (k Keeper) DelegateTo(ctx sdk.Context, params *delegationtype.DelegationOrUndelegationParams) error {
	return k.delegateTo(ctx, params, true)
}

// delegateTo is the internal private version of DelegateTo. if the notGenesis parameter is
// false, the operator keeper and the delegation hooks are not called.
func (k *Keeper) delegateTo(
	ctx sdk.Context,
	params *delegationtype.DelegationOrUndelegationParams,
	notGenesis bool,
) error {
	if !params.OpAmount.IsPositive() {
		return delegationtype.ErrAmountIsNotPositive
	}
	// check if the delegatedTo address is an operator
	if !k.operatorKeeper.IsOperator(ctx, params.OperatorAddress) {
		return errorsmod.Wrap(delegationtype.ErrOperatorNotExist, fmt.Sprintf("input operatorAddr is:%s", params.OperatorAddress))
	}

	// check if the operator has been slashed or frozen
	// skip the check if not genesis (or chain restart)
	if notGenesis && k.slashKeeper.IsOperatorFrozen(ctx, params.OperatorAddress) {
		return delegationtype.ErrOperatorIsFrozen
	}

	stakerID, assetID := assetstype.GetStakerIDAndAssetID(params.ClientChainID, params.StakerAddress, params.AssetsAddress)
	if assetID != assetstype.ExocoreAssetID {
		// check if the staker asset has been deposited and the canWithdraw amount is bigger than the delegation amount
		info, err := k.assetsKeeper.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
		if err != nil {
			return err
		}

		if info.WithdrawableAmount.LT(params.OpAmount) {
			return errorsmod.Wrap(delegationtype.ErrDelegationAmountTooBig, fmt.Sprintf("the opAmount is:%s the WithdrawableAmount amount is:%s", params.OpAmount, info.WithdrawableAmount))
		}

		// update staker asset state
		err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, assetstype.DeltaStakerSingleAsset{
			WithdrawableAmount: params.OpAmount.Neg(),
		})
		if err != nil {
			return err
		}
	} else {
		coins := sdk.NewCoins(sdk.NewCoin(assetstype.ExocoreAssetDenom, params.OpAmount))
		// transfer the delegation amount from the staker account to the delegated pool
		if err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, params.StakerAddress, delegationtype.DelegatedPoolName, coins); err != nil {
			return err
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
				return err
			}
		}
	}
	// calculate the share from the delegation amount
	share, err := k.CalculateShare(ctx, params.OperatorAddress, assetID, params.OpAmount)
	if err != nil {
		return err
	}

	deltaOperatorAsset := assetstype.DeltaOperatorSingleAsset{
		TotalAmount: params.OpAmount,
		TotalShare:  share,
	}
	// Check if the staker belongs to the delegated operator. Increase the operator's share if yes.
	operator, err := k.GetAssociatedOperator(ctx, stakerID)
	if err != nil {
		return err
	}
	if operator == params.OperatorAddress.String() {
		deltaOperatorAsset.OperatorShare = share
	}

	err = k.assetsKeeper.UpdateOperatorAssetState(ctx, params.OperatorAddress, assetID, deltaOperatorAsset)
	if err != nil {
		return err
	}

	deltaAmount := &delegationtype.DeltaDelegationAmounts{
		UndelegatableShare: share,
	}
	_, err = k.UpdateDelegationState(ctx, stakerID, assetID, params.OperatorAddress.String(), deltaAmount)
	if err != nil {
		return err
	}
	err = k.AppendStakerForOperator(ctx, params.OperatorAddress.String(), assetID, stakerID)
	if err != nil {
		return err
	}

	if notGenesis {
		// call the hooks registered by the other modules
		k.Hooks().AfterDelegation(ctx, params.OperatorAddress)
	}
	return nil
}

// UndelegateFrom: The undelegation needs to consider whether the operator's opted-in assets can exit from the AVS.
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
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(params.ClientChainID, params.StakerAddress, params.AssetsAddress)

	// verify the undelegation amount
	share, err := k.ValidateUndelegationAmount(ctx, params.OperatorAddress, stakerID, assetID, params.OpAmount)
	if err != nil {
		return err
	}

	// remove share
	removeToken, err := k.RemoveShare(ctx, true, params.OperatorAddress, stakerID, assetID, share)
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
	}
	completedEpochID, completedEpochNumber, err := k.operatorKeeper.GetUnbondingExpiration(ctx, params.OperatorAddress)
	if err != nil {
		return err
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
	// call the hooks registered by the other modules
	return k.Hooks().AfterUndelegationStarted(ctx, params.OperatorAddress, delegationtype.GetUndelegationRecordKey(r.BlockNumber, r.UndelegationId, r.TxHash, r.OperatorAddr))
}

// AssociateOperatorWithStaker marks that a staker is claiming to be associated with an operator.
// In other words, the staker's delegations will be marked as self-delegations for the operator.
// Each stakerID can associate, at most, to one operator. To change that operator, the staker must
// first call DissociateOperatorFromStaker.
// However, each operator may be associated with multiple stakers.
// This function is not available for end users to call directly, and such calls must be made
// via the ExocoreGateway. The gateway associates the `msg.sender` of the call, along with the
// chain id, with the operator. Since it associates `msg.sender`, we do not need to check that
// this request is signed by the staker.
// TODO: It may be possible that an address, which is an EOA staker on a client chain, is a
// contract on Exocore. When that happens, the contract may be able to call the Gateway to
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
	associatedOperator, err := k.GetAssociatedOperator(ctx, stakerID)
	if err != nil {
		return err
	}
	if associatedOperator != "" {
		return delegationtype.ErrOperatorAlreadyAssociated
	}

	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		// increase the share of new marked operator
		if keys.OperatorAddr == operatorAddress.String() {
			err = k.assetsKeeper.UpdateOperatorAssetState(ctx, operatorAddress, keys.AssetId, assetstype.DeltaOperatorSingleAsset{
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
	associatedOperator, err := k.GetAssociatedOperator(ctx, stakerID)
	if err != nil {
		return err
	}
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
			err = k.assetsKeeper.UpdateOperatorAssetState(ctx, oldOperatorAccAddr, keys.AssetId, assetstype.DeltaOperatorSingleAsset{
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
