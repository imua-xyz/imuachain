package keeper

import (
	"strings"

	"github.com/ExocoreNetwork/exocore/utils"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common/hexutil"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetSlashIDForDogfood It use infractionType+'_'+'infractionHeight' as the slashID, because /* the slash  */event occurs in dogfood doesn't have a TxID. It isn't submitted through an external transaction.
func GetSlashIDForDogfood(infraction stakingtypes.Infraction, infractionHeight int64) string {
	// #nosec G701
	return strings.Join([]string{hexutil.EncodeUint64(uint64(infraction)), hexutil.EncodeUint64(uint64(infractionHeight))}, utils.DelimiterForID)
}

// SlashFromUndelegation executes the slash from an undelegation
func SlashFromUndelegation(undelegation *delegationtype.UndelegationRecord, slashProportion sdkmath.LegacyDec) *types.SlashFromUndelegation {
	if undelegation.ActualCompletedAmount.IsZero() {
		return nil
	}
	slashAmount := slashProportion.MulInt(undelegation.Amount).TruncateInt()
	// reduce the actual_completed_amount in the record
	if slashAmount.GTE(undelegation.ActualCompletedAmount) {
		slashAmount = undelegation.ActualCompletedAmount
		undelegation.ActualCompletedAmount = sdkmath.NewInt(0)
	} else {
		undelegation.ActualCompletedAmount = undelegation.ActualCompletedAmount.Sub(slashAmount)
	}

	return &types.SlashFromUndelegation{
		StakerID: undelegation.StakerID,
		AssetID:  undelegation.AssetID,
		Amount:   slashAmount,
	}
}

func (k *Keeper) CheckSlashParameter(ctx sdk.Context, parameter *types.SlashInputInfo) error {
	if parameter.SlashProportion.IsNil() || parameter.SlashProportion.IsNegative() {
		return errorsmod.Wrapf(types.ErrValueIsNilOrZero, "Invalid SlashProportion; expected non-nil and non-negative, got: %+v", parameter.SlashProportion)
	}
	height := ctx.BlockHeight()
	if parameter.SlashEventHeight > height {
		return errorsmod.Wrapf(types.ErrSlashOccurredHeight, "slashEventHeight:%d,curHeight:%d", parameter.SlashEventHeight, height)
	}

	if parameter.IsDogFood {
		if parameter.Power <= 0 {
			return errorsmod.Wrapf(types.ErrInvalidSlashPower, "slash for dogfood, the power is:%v", parameter.Power)
		}
	} else {
		if parameter.Power != 0 {
			return errorsmod.Wrapf(types.ErrInvalidSlashPower, "slash for other AVSs, the power is:%v", parameter.Power)
		}
		// todo: get the historical voting power from the snapshot for the other AVSs
	}
	return nil
}

// SlashAssets slash the assets according to the new calculated proportion
// It slashs the undelegation first, then slash the assets pool of the related operator
// If the remaining amount of the assets pool after slash is zero, the share of related
// stakers should be cleared, because the divisor will be zero when calculating the share
// of new delegation after the slash.
func (k *Keeper) SlashAssets(ctx sdk.Context, parameter *types.SlashInputInfo) (*types.SlashExecutionInfo, error) {
	// calculate the new slash proportion according to the historical power and current assets state
	slashUSDValue := sdkmath.LegacyNewDec(parameter.Power).Mul(parameter.SlashProportion)
	// calculate the current usd value of all assets pool for the operator
	stakingInfo, err := k.CalculateUSDValueForOperator(ctx, true, parameter.Operator.String(), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	// calculate the new slash proportion
	newSlashProportion := slashUSDValue.Quo(stakingInfo.StakingAndWaitUnbonding)
	newSlashProportion = sdkmath.LegacyMinDec(sdkmath.LegacyNewDec(1), newSlashProportion)

	executionInfo := &types.SlashExecutionInfo{
		SlashProportion:    newSlashProportion,
		SlashValue:         slashUSDValue,
		SlashUndelegations: make([]types.SlashFromUndelegation, 0),
		SlashAssetsPool:    make([]types.SlashFromAssetsPool, 0),
	}
	// slash from the unbonding stakers
	if parameter.SlashEventHeight < ctx.BlockHeight() {
		// get the undelegations that are submitted after the slash.
		opFunc := func(undelegation *delegationtype.UndelegationRecord) error {
			slashFromUndelegation := SlashFromUndelegation(undelegation, newSlashProportion)
			if slashFromUndelegation != nil {
				executionInfo.SlashUndelegations = append(executionInfo.SlashUndelegations, *slashFromUndelegation)
			}
			return nil
		}
		// #nosec G701
		heightFilter := uint64(parameter.SlashEventHeight)
		err = k.delegationKeeper.IterateUndelegationsByOperator(ctx, parameter.Operator.String(), &heightFilter, true, opFunc)
		if err != nil {
			return nil, err
		}
	}

	// slash from the assets pool of the operator
	opFuncToIterateAssets := func(assetID string, state *assetstype.OperatorAssetInfo) error {
		slashAmount := newSlashProportion.MulInt(state.TotalAmount).TruncateInt()
		remainingAmount := state.TotalAmount.Sub(slashAmount)
		// todo: consider slash all assets if the remaining amount is too small,
		// which can avoid the unbalance between share and amount

		// all shares need to be cleared if the asset amount is slashed to zero,
		// otherwise there will be a problem in updating the shares when handling
		// the new delegations.
		if remainingAmount.IsZero() &&
			k.delegationKeeper.HasStakerList(ctx, parameter.Operator.String(), assetID) {
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
			state.TotalShare = sdkmath.LegacyNewDec(0)
			state.OperatorShare = sdkmath.LegacyNewDec(0)
		}
		state.TotalAmount = remainingAmount
		executionInfo.SlashAssetsPool = append(executionInfo.SlashAssetsPool, types.SlashFromAssetsPool{
			AssetID: assetID,
			Amount:  slashAmount,
		})
		return nil
	}
	err = k.assetsKeeper.IterateAssetsForOperator(ctx, true, parameter.Operator.String(), nil, opFuncToIterateAssets)
	if err != nil {
		return nil, err
	}
	return executionInfo, nil
}

// Slash performs all slash events and stores the execution result
func (k *Keeper) Slash(ctx sdk.Context, parameter *types.SlashInputInfo) error {
	err := k.CheckSlashParameter(ctx, parameter)
	if err != nil {
		return err
	}

	// slash assets according to the input information
	// using cache context to ensure the atomicity of slash execution.
	cc, writeFunc := ctx.CacheContext()
	executionInfo, err := k.SlashAssets(cc, parameter)
	if err != nil {
		return err
	}
	writeFunc()
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
	err = k.UpdateOperatorSlashInfo(ctx, parameter.Operator.String(), parameter.AVSAddr, parameter.SlashID, slashInfo)
	if err != nil {
		return err
	}
	return nil
}

// SlashWithInfractionReason is an expected slash interface for the dogfood module.
func (k Keeper) SlashWithInfractionReason(
	ctx sdk.Context, addr sdk.AccAddress, infractionHeight, power int64,
	slashFactor sdk.Dec, infraction stakingtypes.Infraction,
) sdkmath.Int {
	chainID := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	isAvs, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAvs {
		k.Logger(ctx).Error("the chainID is not supported by AVS", "chainID", chainID)
		return sdkmath.NewInt(0)
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
		return sdkmath.NewInt(0)
	}
	// todo: The returned value should be the amount of burned Exo if we considering a slash from the reward
	// Now it doesn't slash from the reward, so just return 0
	return sdkmath.NewInt(0)
}

// IsOperatorJailedForChainID returns whether an operator is jailed for a specific chainID.
func (k Keeper) IsOperatorJailedForChainID(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) bool {
	found, operatorAddr := k.GetOperatorAddressForChainIDAndConsAddr(ctx, chainID, consAddr)
	if !found {
		k.Logger(ctx).Info("couldn't find operator by consensus address and chainID", "consAddr", consAddr, "chainID", chainID)
		return false
	}

	isAvs, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAvs {
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

	isAvs, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	if !isAvs {
		k.Logger(ctx).Error("the chainID is not supported by AVS", "chainID", chainID)
		return
	}

	handleFunc := func(info *types.OptedInfo) {
		info.Jailed = jailed
	}
	err := k.HandleOptedInfo(ctx, operatorAddr.String(), avsAddr, handleFunc)
	if err != nil {
		k.Logger(ctx).Error(err.Error(), chainID)
	}
}

// Jail an operator
func (k Keeper) Jail(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) {
	k.SetJailedState(ctx, consAddr, chainID, true)
}

// Unjail an operator
func (k Keeper) Unjail(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) {
	k.SetJailedState(ctx, consAddr, chainID, false)
}
