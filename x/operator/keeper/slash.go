package keeper

import (
	"encoding/json"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/utils"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

// GetSlashIDForDogfood It use infractionType+'_'+'infractionHeight' as the slashID,
// because the slash event occurs in dogfood doesn't have a TxID. It isn't submitted
// through an external transaction.
func GetSlashIDForDogfood(infraction stakingtypes.Infraction, infractionHeight int64) string {
	slashIDBytes := utils.AppendMany(
		utils.Uint32ToBigEndian(uint32(infraction)),
		sdk.Uint64ToBigEndian(uint64(infractionHeight)))
	return hexutil.Encode(slashIDBytes)
}

// SlashFromUndelegation executes the slash from an undelegation, reduce the .ActualCompletedAmount from undelegationRecords
func (k *Keeper) SlashFromUndelegation(ctx sdk.Context, undelegation *delegationtype.UndelegationRecord, slashProportion sdkmath.LegacyDec) (*types.SlashFromUndelegation, error) {
	if undelegation.ActualCompletedAmount.IsZero() {
		// do nothing because there isn't amount to be slashed
		return nil, nil
	}
	slashAmount := slashProportion.MulInt(undelegation.Amount).TruncateInt()
	// reduce the actual_completed_amount in the record
	if slashAmount.GTE(undelegation.ActualCompletedAmount) {
		slashAmount = undelegation.ActualCompletedAmount
		undelegation.ActualCompletedAmount = sdkmath.ZeroInt()
	} else {
		undelegation.ActualCompletedAmount = undelegation.ActualCompletedAmount.Sub(slashAmount)
	}

	if undelegation.RewardAsset {
		// handle the reward states if it's a reward asset undelegation
		err := k.distributionKeeper.SlashRewardUndelegation(ctx, undelegation, slashProportion)
		if err != nil {
			return nil, err
		}
	}

	return &types.SlashFromUndelegation{
		StakerID:       undelegation.StakerId,
		AssetID:        undelegation.AssetId,
		Amount:         slashAmount,
		UndelegationId: undelegation.UndelegationId,
		RewardAsset:    undelegation.RewardAsset,
	}, nil
}

func (k *Keeper) CheckSlashParameter(ctx sdk.Context, parameter *types.SlashInputInfo) error {
	height := ctx.BlockHeight()
	if parameter.SlashEventHeight > height {
		return types.ErrSlashOccurredHeight.Wrapf("slashEventHeight:%d,curHeight:%d", parameter.SlashEventHeight, height)
	}

	if parameter.IsDogFood {
		if parameter.Power <= 0 {
			return types.ErrInvalidSlashPower.Wrapf("slash for dogfood, the power is:%v", parameter.Power)
		}
	} else {
		if parameter.Power != 0 {
			return types.ErrInvalidSlashPower.Wrapf("slash for other AVSs, the input power should be zero, power:%v", parameter.Power)
		}
	}
	return nil
}

// SlashAssets slash the assets according to the new calculated proportion
// It slashs the undelegation first, then slash the assets pool of the related operator
// If the remaining amount of the assets pool after slash is zero, the share of related
// stakers should be cleared, because the divisor will be zero when calculating the share
// of new delegation after the slash.
func (k *Keeper) SlashAssets(ctx sdk.Context, snapshotHeight int64, parameter *types.SlashInputInfo) (*types.SlashExecutionInfo, error) {
	// calculate the new slash proportion according to the historical power and current assets state
	slashUSDValue := sdkmath.LegacyNewDec(parameter.Power).Mul(parameter.SlashProportion)
	// calculate the current usd value of all assets pool for the operator
	stakingInfo, err := k.CalculateRealTimeOperatorUSDValue(ctx, true, parameter.Operator.String(), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	// calculate the new slash proportion
	newSlashProportion := slashUSDValue.Quo(stakingInfo.StakingAndWaitUnbonding)
	newSlashProportion = sdkmath.LegacyMinDec(sdkmath.LegacyNewDec(1), newSlashProportion)

	executionInfo := &types.SlashExecutionInfo{
		SlashProportion:          newSlashProportion,
		SlashValue:               slashUSDValue,
		SlashUndelegations:       make([]types.SlashFromUndelegation, 0),
		SlashAssetsPool:          make([]types.SlashAssetAmount, 0),
		UndelegationFilterHeight: snapshotHeight,
		HistoricalVotingPower:    parameter.Power,
	}
	// slash from the unbonding stakers
	if parameter.SlashEventHeight < ctx.BlockHeight() {
		// get the undelegations that are submitted after the slash.
		opFunc := func(undelegation *delegationtype.UndelegationRecord) error {
			slashFromUndelegation, err := k.SlashFromUndelegation(ctx, undelegation, newSlashProportion)
			if err != nil {
				return err
			}
			if slashFromUndelegation != nil {
				executionInfo.SlashUndelegations = append(executionInfo.SlashUndelegations, *slashFromUndelegation)
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeUndelegationSlashed,
						sdk.NewAttribute(types.AttributeKeyRecordID, hexutil.Encode(undelegation.GetKey())),
						// amount left after slashing has been performed
						sdk.NewAttribute(types.AttributeKeyAmount, undelegation.ActualCompletedAmount.String()),
						// slashed quantity
						sdk.NewAttribute(types.AttributeKeySlashAmount, slashFromUndelegation.Amount.String()),
					),
				)

			}
			return nil
		}
		// #nosec G701
		heightFilter := uint64(snapshotHeight)
		err = k.delegationKeeper.IterateUndelegationsByOperator(ctx, parameter.Operator.String(), &heightFilter, true, opFunc)
		if err != nil {
			return nil, err
		}
	}

	// slash from the assets pool of the operator, emits operator asset info status event.
	opFuncToIterateAssets := func(assetID string, state *assetstype.OperatorAssetInfo) error {
		// iterate over each operator + asset and reduce the total amount by the slash amount
		slashAmount := newSlashProportion.MulInt(state.TotalAmount).TruncateInt()
		if !slashAmount.IsPositive() {
			// do nothing if the slash amount isn't positive.
			return nil
		}
		remainingAmount := state.TotalAmount.Sub(slashAmount)
		// todo: consider slash all assets if the remaining amount is too small,
		// which can avoid the unbalance between share and amount

		// all shares need to be cleared if the asset amount is slashed to zero,
		// otherwise there will be a problem in updating the shares when handling
		// the new delegations.
		if remainingAmount.IsZero() && k.delegationKeeper.HasStakerList(ctx, parameter.Operator.String(), assetID) {
			// clear the share of other stakers
			stakerList, err := k.delegationKeeper.GetStakersByOperator(ctx, parameter.Operator.String(), assetID)
			if err != nil {
				return err
			}
			err = k.delegationKeeper.SetStakerShareToZero(ctx, parameter.Operator.String(), assetID, stakerList)
			if err != nil {
				return err
			}
			err = k.delegationKeeper.DeleteStakersListForOperator(ctx, parameter.Operator.String(), assetID)
			if err != nil {
				return err
			}
			state.TotalShare = sdkmath.LegacyZeroDec()
			state.OperatorShare = sdkmath.LegacyZeroDec()
		}
		state.TotalAmount = remainingAmount
		// TODO: check if pendingUndelegation also zero => delete this item, and this operator should be opted out if
		// all assets falls to 0 since the miniself is not satisfied then.
		executionInfo.SlashAssetsPool = append(executionInfo.SlashAssetsPool, types.SlashAssetAmount{
			AssetID: assetID,
			Amount:  slashAmount,
		})
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeOperatorAssetSlashed,
				sdk.NewAttribute(types.AttributeKeyOperator, parameter.Operator.String()),
				sdk.NewAttribute(types.AttributeKeyAssetID, assetID),
				sdk.NewAttribute(types.AttributeKeyAmount, slashAmount.String()),
			),
		)
		return nil
	}
	err = k.assetsKeeper.IterateAssetsForOperator(ctx, true, parameter.Operator.String(), nil, opFuncToIterateAssets)
	if err != nil {
		return nil, err
	}

	// slash the compounding rewards
	slashFromUnclaimedRewards, err := k.distributionKeeper.SlashOperatorUnclaimedRewards(ctx, parameter.Operator.String(), stakingInfo.CompoundingUSDValueSources, newSlashProportion)
	if err != nil {
		return nil, err
	}
	executionInfo.SlashUnclaimedRewards = slashFromUnclaimedRewards
	return executionInfo, nil
}

// Slash performs all slash events and stores the execution result
func (k *Keeper) Slash(ctx sdk.Context, parameter *types.SlashInputInfo) error {
	err := k.CheckSlashParameter(ctx, parameter)
	if err != nil {
		return err
	}
	slashEventEpochStartHeight, snapshot, err := k.LoadVotingPowerSnapshot(ctx, parameter.AVSAddr, parameter.SlashEventHeight)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("execute slashing", "eventHeight", parameter.SlashEventHeight, "avsAddr", parameter.AVSAddr, "operator", parameter.Operator, "slashID", parameter.SlashID, "slashType", parameter.SlashType)
	// Marshal the snapshot to improve the user experience when printing the voting power decimal through the logger
	// so we don't have to address the error here.
	snapshotJSON, _ := json.Marshal(snapshot)
	k.Logger(ctx).Info("the voting power snapshot info is:", "filter_height", slashEventEpochStartHeight, "snapshot", string(snapshotJSON))
	// get the historical voting power from the snapshot for the other AVSs
	if !parameter.IsDogFood {
		votingPower := types.GetSpecifiedVotingPower(parameter.Operator.String(), snapshot.OperatorVotingPowers)
		if votingPower == nil {
			return types.ErrFailToGetHistoricalVP.Wrapf("slash: the operator isn't in the voting power set, addr:%s", parameter.Operator)
		}
		parameter.Power = votingPower.VotingPower.TruncateInt64()
		if parameter.Power < 0 {
			return types.ErrInvalidSlashPower.Wrapf("slash: invalid voting power, power:%v", parameter.Power)
		}
	}
	if parameter.Power == 0 {
		k.Logger(ctx).Info("don't execute the slash if the historical voting power is zero")
		return nil
	}

	// slash assets according to the input information
	// using cache context to ensure the atomicity of slash execution.
	cc, writeFunc := ctx.CacheContext()
	executionInfo, err := k.SlashAssets(cc, slashEventEpochStartHeight, parameter)
	if err != nil {
		return err
	}

	// update the voting power and save the snapshot for all affected AVSs
	affectedAVSList, affectedEpochList, err := k.GetImpactfulEpochsAndAVSsForOperator(ctx, parameter.Operator.String())
	if err != nil {
		return err
	}
	// update the operator asset USD value first
	err = k.UpdateAllOperatorAssetUSDValues(cc, affectedEpochList)
	if err != nil {
		return err
	}
	for _, avs := range affectedAVSList {
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avs)
		if err != nil {
			return err
		}
		err = k.UpdateVotingPower(cc, avs, epochInfo.Identifier, epochInfo.CurrentEpoch, true)
		if err != nil {
			return err
		}
	}
	k.hooks.AfterSlash(cc, parameter.Operator, executionInfo.SlashProportion, affectedAVSList,
		executionInfo.SlashAssetsPool, executionInfo.SlashUnclaimedRewards)

	// store the slash information
	height := ctx.BlockHeight()
	slashInfo := types.OperatorSlashInfo{
		SlashType:       parameter.SlashType,
		SlashContract:   parameter.SlashContract,
		SubmittedHeight: height,
		EventHeight:     parameter.SlashEventHeight,
		SlashProportion: parameter.SlashProportion,
		ExecutionInfo:   executionInfo,
	}
	err = k.UpdateOperatorSlashInfo(cc, parameter.Operator.String(), parameter.AVSAddr, parameter.SlashID, slashInfo)
	if err != nil {
		return err
	}
	writeFunc()
	return nil
}

// SlashWithInfractionReason is an expected slash interface for the dogfood module.
func (k Keeper) SlashWithInfractionReason(
	ctx sdk.Context, addr sdk.AccAddress, infractionHeight, power int64,
	slashFactor sdk.Dec, infraction stakingtypes.Infraction,
) sdkmath.Int {
	if slashFactor.IsNil() || slashFactor.IsNegative() {
		k.Logger(ctx).Error("invalid slash factor, expected non-nil and non-negative", "slashFactor", slashFactor)
		return sdkmath.ZeroInt()
	} else if slashFactor.IsZero() {
		k.Logger(ctx).Info("slash factor is zero, do nothing for the slash execution")
		return sdkmath.ZeroInt()
	}

	chainID := utils.ChainIDWithoutRevision(ctx.ChainID())
	isAVS, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAVS {
		k.Logger(ctx).Error("the chainID is not supported by AVS", "chainID", chainID)
		return sdkmath.ZeroInt()
	}
	slashID := GetSlashIDForDogfood(infraction, infractionHeight)
	slashParam := &types.SlashInputInfo{
		IsDogFood:        true,
		Power:            power,
		SlashType:        uint32(infraction),
		Operator:         addr,
		AVSAddr:          avsAddr,
		SlashID:          slashID,
		SlashEventHeight: infractionHeight,
		SlashProportion:  slashFactor,
	}
	err := k.Slash(ctx, slashParam)
	if err != nil {
		k.Logger(ctx).Error("error when executing slash", "error", err, "avsAddr", avsAddr)
		return sdkmath.ZeroInt()
	}
	// The returned value should represent the amount of IMUA burned in the Cosmos SDK.
	// In the IMUA chain implementation, we do not actually burn any slashed assets;
	// instead, we lock these assets, which better supports the slashing veto mechanism.
	return sdkmath.ZeroInt()
}

// IsOperatorJailedForChainID returns whether an operator is jailed for a specific chainID.
func (k Keeper) IsOperatorJailedForChainID(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) bool {
	found, operatorAddr := k.GetOperatorAddressForChainIDAndConsAddr(ctx, chainID, consAddr)
	if !found {
		k.Logger(ctx).Info("couldn't find operator by consensus address and chainID", "consAddr", consAddr, "chainID", chainID)
		return false
	}

	isAVS, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAVS {
		k.Logger(ctx).Error("the chainID is not supported by AVS", chainID)
		return false
	}
	optInfo, err := k.GetOptedInfo(ctx, operatorAddr.String(), avsAddr)
	if err != nil {
		k.Logger(ctx).Error(err.Error(), operatorAddr, avsAddr)
		return false
	}
	return optInfo.Jailed
}

func (k *Keeper) SetJailedState(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string, jailed bool) {
	found, operatorAddr := k.GetOperatorAddressForChainIDAndConsAddr(ctx, chainID, consAddr)
	if !found {
		k.Logger(ctx).Info("couldn't find operator by consensus address and chainID", "consAddr", consAddr, "chainID", chainID)
		return
	}

	isAVS, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAVS {
		k.Logger(ctx).Error("the chainID is not supported by AVS", "chainID", chainID)
		return
	}

	handleFunc := func(info *types.OptedInfo) {
		info.Jailed = jailed
		height := ctx.BlockHeight()
		// append jail or junjail height
		info.JailToggleHeights = append(info.JailToggleHeights, uint64(height))
	}
	err := k.HandleOptedInfo(ctx, operatorAddr.String(), avsAddr, handleFunc)
	if err != nil {
		k.Logger(ctx).Error(err.Error(), chainID)
	}

	// TODO: Should jailing by one AVS apply to all AVSs the operator is serving, similar to slashing?
	// Or should it only affect the AVS that jailed the operator, allowing them to continue serving the others?
	affectedAVSList, _, err := k.GetImpactfulEpochsAndAVSsForOperator(ctx, operatorAddr.String())
	if err != nil {
		return
	}
	k.hooks.AfterJail(ctx, operatorAddr, !jailed, affectedAVSList)
}

// Jail an operator
func (k Keeper) Jail(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) {
	k.SetJailedState(ctx, consAddr, chainID, true)
}

// Unjail an operator
func (k Keeper) Unjail(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) {
	k.SetJailedState(ctx, consAddr, chainID, false)
}
