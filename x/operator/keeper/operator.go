package keeper

import (
	"fmt"
	"math"
	"strings"
	"time"

	epochtypes "github.com/imua-xyz/imuachain/x/epochs/types"

	"golang.org/x/xerrors"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// SetOperatorInfo is used to store the operator's information on the chain.
// There is no current way implemented to delete an operator's registration or edit it.
// TODO: implement operator edit function, which should allow editing:
// approve address?
// name (meta info)
// commission, subject to limits and once within 24 hours.
// client chain earnings addresses (maybe append only?)
func (k *Keeper) RegisterOperator(
	ctx sdk.Context, addr string, info *operatortypes.OperatorInfo,
) (err error) {
	if info == nil {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "SetOperatorInfo: operator info is nil")
	}
	if err := info.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "SetOperatorInfo: operator info is invalid")
	}
	if info.Commission.UpdateTime.Equal(time.Time{}) {
		info.Commission.UpdateTime = ctx.BlockTime()
	}
	// #nosec G703 // already validated in `ValidateBasic`
	opAccAddr, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return errorsmod.Wrap(err, "SetOperatorInfo: error occurred when parse acc address from Bech32")
	}
	// already checked that addr is valid, so only check match below
	if addr != info.EarningsAddr {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "SetOperatorInfo: operator address does not match earnings address")
	}
	if addr != info.ApproveAddr {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "SetOperatorInfo: operator address does not match approve address")
	}
	// if already registered, this request should go to EditOperator.
	if k.IsOperator(ctx, opAccAddr) {
		return errorsmod.Wrap(
			operatortypes.ErrOperatorAlreadyExists,
			fmt.Sprintf("SetOperatorInfo: operator already exists, address: %s", opAccAddr),
		)
	}
	// check if the operator name already exists
	if has, err := k.HasOperatorName(ctx, info.OperatorMetaInfo); err != nil {
		return errorsmod.Wrap(err, "SetOperatorInfo: error occurred when checking operator name")
	} else if has {
		return errorsmod.Wrap(
			operatortypes.ErrOperatorNameAlreadyExists,
			fmt.Sprintf("SetOperatorInfo: operator name already exists, name: %s", info.OperatorMetaInfo),
		)
	}
	// check if the client chain earning addresses are valid
	if info.ClientChainEarningsAddr != nil {
		for _, data := range info.ClientChainEarningsAddr.EarningInfoList {
			if data.ClientChainEarningAddr == "" {
				return errorsmod.Wrap(
					operatortypes.ErrParameterInvalid,
					"SetOperatorInfo: client chain earning address is empty",
				)
			}
			if !k.assetsKeeper.ClientChainExists(ctx, data.LzClientChainID) {
				return errorsmod.Wrap(
					operatortypes.ErrParameterInvalid,
					"SetOperatorInfo: client chain not found",
				)
			}
		}
	}
	// unchecked data storage
	k.setOperatorInfo(ctx, opAccAddr, info)
	// event for the indexer
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeRegisterOperator,
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, opAccAddr.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyMetaInfo, info.OperatorMetaInfo),
			sdk.NewAttribute(stakingtypes.AttributeKeyCommissionRate, info.Commission.Rate.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyMaxCommissionRate, info.Commission.MaxRate.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyMaxChangeRate, info.Commission.MaxChangeRate.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyCommissionUpdateTime, sdk.FormatTimeString(info.Commission.UpdateTime)),
			// TODO: add ClientChainEarningsAddr.EarningInfoList to the event
		),
	)
	return nil
}

// EditOperator edits an operator's meta info.
func (k *Keeper) EditOperator(
	ctx sdk.Context, opAccAddr sdk.AccAddress, metaInfo string,
) error {
	info, err := k.OperatorInfo(ctx, opAccAddr.String())
	if err != nil {
		return err
	}
	// this prevents resetting to the same name as well
	if has, err := k.HasOperatorName(ctx, metaInfo); err != nil {
		return err
	} else if has {
		return errorsmod.Wrap(
			operatortypes.ErrOperatorNameAlreadyExists,
			fmt.Sprintf("EditOperator: operator name already exists, name: %s", metaInfo),
		)
	}
	info.OperatorMetaInfo = metaInfo
	if err := info.ValidateBasic(); err != nil {
		return err
	}
	k.setOperatorInfo(ctx, opAccAddr, info)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeEditOperator,
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, opAccAddr.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyMetaInfo, metaInfo),
		),
	)
	return nil
}

// setOperatorInfo is used to store the operator's information on the chain.
// It does not validate the operator info.
// It is used by `RegisterOperator` and `UpdateCommissionRate`.
func (k *Keeper) setOperatorInfo(
	ctx sdk.Context, opAccAddr sdk.AccAddress, info *operatortypes.OperatorInfo,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	bz := k.cdc.MustMarshal(info)
	store.Set(opAccAddr, bz)
}

// HasOperatorName checks if the operator name already exists.
func (k *Keeper) HasOperatorName(ctx sdk.Context, name string) (bool, error) {
	res := false
	opFunc := func(_ sdk.AccAddress, operatorInfo *operatortypes.OperatorInfo) (bool, error) {
		if operatorInfo.OperatorMetaInfo == name {
			res = true
			// stop, no error
			return true, nil
		}
		// continue, no error
		return false, nil
	}
	err := k.IterateOperators(ctx, opFunc)
	if err != nil {
		return false, err
	}
	return res, nil
}

func (k *Keeper) OperatorInfo(ctx sdk.Context, addr string) (info *operatortypes.OperatorInfo, err error) {
	opAccAddr, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "GetOperatorInfo: error occurred when parse acc address from Bech32")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	// key := common.HexToAddress(incentive.Contract)
	value := store.Get(opAccAddr)
	if value == nil {
		return nil, errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("GetOperatorInfo: key is %s", opAccAddr))
	}
	ret := operatortypes.OperatorInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// IterateOperators return the list of all operators' detailed information
func (k *Keeper) IterateOperators(ctx sdk.Context, opFunc func(operatorAddr sdk.AccAddress, operatorInfo *operatortypes.OperatorInfo) (bool, error)) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var operatorInfo operatortypes.OperatorInfo
		operatorAddr := sdk.AccAddress(iterator.Key())
		k.cdc.MustUnmarshal(iterator.Value(), &operatorInfo)
		isBreak, err := opFunc(operatorAddr, &operatorInfo)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}
	}
	return nil
}

// AllOperators return the list of all operators' detailed information
func (k *Keeper) AllOperators(ctx sdk.Context) []operatortypes.OperatorDetail {
	ret := make([]operatortypes.OperatorDetail, 0)
	opFunc := func(operatorAddr sdk.AccAddress, operatorInfo *operatortypes.OperatorInfo) (bool, error) {
		ret = append(ret, operatortypes.OperatorDetail{
			OperatorAddress: operatorAddr.String(),
			OperatorInfo:    *operatorInfo,
		})
		return false, nil
	}
	err := k.IterateOperators(ctx, opFunc)
	if err != nil {
		ctx.Logger().Error("error when iterating operators", "err", err)
	}
	return ret
}

func (k Keeper) IsOperator(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	return store.Has(addr)
}

func (k *Keeper) HandleOptedInfo(ctx sdk.Context, operatorAddr, avsAddr string, handleFunc func(info *operatortypes.OptedInfo)) error {
	opAccAddr, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return errorsmod.Wrap(err, "HandleOptedInfo: error occurred when parse acc address from Bech32")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr))
	// get info from the store
	value := store.Get(infoKey)
	if value == nil {
		return errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("HandleOptedInfo: key is %s", opAccAddr))
	}
	info := &operatortypes.OptedInfo{}
	k.cdc.MustUnmarshal(value, info)
	// call the handleFunc
	handleFunc(info)
	// restore the info after handling
	bz := k.cdc.MustMarshal(info)
	store.Set(infoKey, bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeOptInfoUpdated,
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, operatorAddr),
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
			sdk.NewAttribute(operatortypes.AttributeKeySlashContract, info.SlashContract),
			sdk.NewAttribute(operatortypes.AttributeKeyOptInHeight, fmt.Sprintf("%d", info.OptedInHeight)),
			sdk.NewAttribute(operatortypes.AttributeKeyOptOutHeight, fmt.Sprintf("%d", info.OptedOutHeight)),
			sdk.NewAttribute(operatortypes.AttributeKeyJailed, fmt.Sprintf("%t", info.Jailed)),
		),
	)
	return nil
}

func (k *Keeper) SetOptedInfo(ctx sdk.Context, operatorAddr, avsAddr string, info *operatortypes.OptedInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)

	// check operator address validation
	_, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return assetstype.ErrInvalidOperatorAddr
	}
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr))

	bz := k.cdc.MustMarshal(info)
	store.Set(infoKey, bz)
	return nil
}

func (k *Keeper) GetOptedInfo(ctx sdk.Context, operatorAddr, avsAddr string) (info *operatortypes.OptedInfo, err error) {
	opAccAddr, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "GetOptedInfo: error occurred when parse acc address from Bech32")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr))
	value := store.Get(infoKey)
	if value == nil {
		return nil, errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("GetOptedInfo: operator is %s, avs address is %s", opAccAddr, avsAddr))
	}

	ret := operatortypes.OptedInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k *Keeper) IsOptedIn(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	optedInfo, err := k.GetOptedInfo(ctx, operatorAddr, avsAddr)
	if err != nil {
		return false
	}
	return optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight
}

func (k *Keeper) IsJailed(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	optedInfo, err := k.GetOptedInfo(ctx, operatorAddr, avsAddr)
	if err != nil {
		return false
	}
	return optedInfo.Jailed
}

func (k *Keeper) IsOptedOutAndEffective(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	optedInfo, err := k.GetOptedInfo(ctx, operatorAddr, avsAddr)
	if err != nil {
		return true
	}
	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return true
	}
	// The operator will remain active even if it has opted out of the AVS until the voting power
	// is updated at the end of the epoch.
	if optedInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight &&
		optedInfo.OptedOutHeight < uint64(epochInfo.CurrentEpochStartHeight) {
		// opted out and the voting power has been updated at the end of epoch
		return true
	}
	return false
}

func (k *Keeper) IsOptedOutButNotEffective(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	optedInfo, err := k.GetOptedInfo(ctx, operatorAddr, avsAddr)
	if err != nil {
		return false
	}
	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return false
	}
	// The operator will remain active even if it has opted out of the AVS until the voting power
	// is updated at the end of the epoch.
	if optedInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight &&
		optedInfo.OptedOutHeight >= uint64(epochInfo.CurrentEpochStartHeight) {
		// opted out and the voting power hasn't been updated at the end of epoch
		return true
	}
	return false
}

// IsActive is used to check if the operator is serving the AVS, the opted out operator will still serve the
// AVS until the next epoch.
func (k *Keeper) IsActive(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	return !k.IsOptedOutAndEffective(ctx, operatorAddr, avsAddr) && !k.IsJailed(ctx, operatorAddr, avsAddr)
}

// IsOptedInAndNotJailed checks whether the operator is opted in and not jailed.
// Compared to `IsActive`, this returns false immediately after the operator opts out.
// It will be used when updating voting power at the end of an epoch, as we need to exclude
// operators who have opted out in the current epoch. These operators would still be included
// if we used `IsActive`, because the voting power update happens before the epoch info is updated.
func (k *Keeper) IsOptedInAndNotJailed(ctx sdk.Context, operatorAddr, avsAddr string) bool {
	return k.IsOptedIn(ctx, operatorAddr, avsAddr) && !k.IsJailed(ctx, operatorAddr, avsAddr)
}

func (k *Keeper) IterateOptInfo(ctx sdk.Context, iteratePrefix []byte, opFunc func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error)) error {
	// get all opted-in info
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	iterator := sdk.KVStorePrefixIterator(store, iteratePrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var optedInfo operatortypes.OptedInfo
		k.cdc.MustUnmarshal(iterator.Value(), &optedInfo)
		isBreak, err := opFunc(iterator.Key(), &optedInfo)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}
	}
	return nil
}

func (k *Keeper) GetOptedInAVSForOperator(ctx sdk.Context, operatorAddr string) ([]string, error) {
	if _, err := sdk.AccAddressFromBech32(operatorAddr); err != nil {
		return nil, operatortypes.ErrParameterInvalid.Wrapf("invalid operator address,err:%s", err)
	}
	avsList := make([]string, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
			keys, err := assetstype.ParseJoinedStoreKey(key, 2)
			if err != nil {
				return false, err
			}
			avsList = append(avsList, keys[1])
		}
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte(operatorAddr), opFunc)
	if err != nil {
		return nil, err
	}
	return avsList, nil
}

func (k *Keeper) isUnbondingRelated(ctx sdk.Context, avsAddr string, optedInfo *operatortypes.OptedInfo) (bool, uint64, error) {
	unbondingDuration, err := k.avsKeeper.GetAVSUnbondingDuration(ctx, avsAddr)
	if err != nil {
		return false, 0, err
	}
	// add AVS currently opting in to the operator's list.
	if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
		return true, unbondingDuration, nil
	}
	// Add AVS that have opted out but are still within the unbonding duration,
	// and therefore still affect the operator, to the list. This can prevent the operator decrease
	// the unbonding duration through opt-out
	// #nosec G115
	epochNumber, err := k.GetEpochNumberByOptOutHeight(ctx, avsAddr, int64(optedInfo.OptedOutHeight))
	if err != nil {
		return false, unbondingDuration, err
	}
	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return false, unbondingDuration, err
	}
	if epochInfo.CurrentEpoch < epochNumber {
		return false, unbondingDuration, xerrors.Errorf("IsUnbondingRelated: current epoch number is less than the retrieved epoch number by opted out height,avsAddr:%s,epochIdentifier:%s,currentEpochNumber:%d,optedOutEpochNumber:%d", avsAddr, epochInfo.Identifier, epochInfo.CurrentEpoch, epochNumber)
	}
	if epochNumber > 0 && uint64(epochNumber)+unbondingDuration >= uint64(epochInfo.CurrentEpoch) {
		OptOutUnbondingRemaining := uint64(epochNumber) + unbondingDuration - uint64(epochInfo.CurrentEpoch)
		return true, OptOutUnbondingRemaining, nil
	}
	return false, 0, nil
}

// GetUnbondingRelatedAVS return the AVSs that still influence the unbonding duration of operator.
func (k *Keeper) GetUnbondingRelatedAVS(ctx sdk.Context, operatorAddr string) ([]operatortypes.ImpactfulAVSInfo, error) {
	avsList := make([]operatortypes.ImpactfulAVSInfo, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		keys, err := assetstype.ParseJoinedStoreKey(key, 2)
		avsAddr := keys[1]
		if err != nil {
			return false, err
		}
		isUnbondingRelated, unbondingDuration, err := k.isUnbondingRelated(ctx, avsAddr, optedInfo)
		if err != nil {
			return false, err
		}
		if isUnbondingRelated {
			avsList = append(avsList,
				operatortypes.ImpactfulAVSInfo{
					AVSAddr:                  avsAddr,
					OptOutUnbondingRemaining: unbondingDuration,
				})
		}
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte(operatorAddr), opFunc)
	if err != nil {
		return nil, err
	}
	return avsList, nil
}

func (k Keeper) GetUnbondingExpiration(ctx sdk.Context, operator sdk.AccAddress) (string, int64, uint64, error) {
	// get the impactful AVSs for the operator
	avsList, err := k.GetUnbondingRelatedAVS(ctx, operator.String())
	if err != nil {
		return "", 0, 0, err
	}
	// calculate the maximum unbonding expiration
	// Using self-definied NullEpochIdentifier and NullEpochNumber as the default unbonding expiration.
	retEpochIdentifier := epochtypes.NullEpochIdentifier
	retEpochNumber := epochtypes.NullEpochNumber
	maxDurationSeconds := uint64(0)
	for _, avs := range avsList {
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avs.AVSAddr)
		if err != nil {
			return "", 0, 0, err
		}
		unbondingDuration := avs.OptOutUnbondingRemaining
		if unbondingDuration+uint64(epochInfo.CurrentEpoch) > uint64(math.MaxInt64) {
			return "", 0, 0, xerrors.New("the sum of unbondingDuration and the current epoch number exceeds the int64 range.")
		}

		remainingTimeForCurrentEpoch := uint64(epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration).Sub(ctx.BlockTime()))
		unbondingDurationSeconds := unbondingDuration*uint64(epochInfo.Duration) + remainingTimeForCurrentEpoch
		if unbondingDurationSeconds > maxDurationSeconds {
			retEpochIdentifier = epochInfo.Identifier
			retEpochNumber = epochInfo.CurrentEpoch + int64(unbondingDuration)
			maxDurationSeconds = unbondingDurationSeconds
		}
	}
	return retEpochIdentifier, retEpochNumber, maxDurationSeconds, nil
}

// GetInstantUnbondingExpiration determines the correct epoch identifier to use for instant undelegations.
// Instant undelegations are completed at the end of the current epoch, but since an operator might opt into
// multiple AVSs with different epoch configurations, we must choose the appropriate epoch identifier carefully.
//
// Specifically, we select the epoch identifier with the longest remaining time in the current epoch.
// This ensures that the undelegation completes after all other relevant epochs end, so that voting power updates
// and reward distributions for the current epoch remain unaffected.
//
// For example, suppose the operator has opted into three AVSs with the following epoch identifiers and remaining times:
//   - AVS1: minute (remaining: 20s)
//   - AVS2: day (remaining: 7h)
//   - AVS3: week (remaining: 6h)
//
// In this case, "day" should be chosen as the epoch identifier for the undelegation completion,
// because it has the latest epoch end time among all related AVSs. The instant undelegation will be completed
// in 7 hours, at the end of the current "day" epoch.
// The returned epoch number will always correspond to the current epoch of the selected identifier.
// The first return value indicates whether the calculated instant unbonding duration is less than
// the non-instant unbonding duration. This value can be used to determine whether a slash should be
// applied for the submitted instant undelegation.
func (k Keeper) GetInstantUnbondingExpiration(ctx sdk.Context, operator sdk.AccAddress) (bool, string, int64, error) {
	// get the related AVSs for the operator
	avsList, err := k.GetUnbondingRelatedAVS(ctx, operator.String())
	if err != nil {
		return false, "", 0, err
	}

	// calculate the maximum instant unbonding expiration
	// Using self-definied NullEpochIdentifier and NullEpochNumber as the default unbonding expiration.
	instantEpochIdentifier := epochtypes.NullEpochIdentifier
	instantEpochNumber := epochtypes.NullEpochNumber
	maxRemainingTime := uint64(0)
	handledEpochMap := make(map[string]interface{})
	for _, avs := range avsList {
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avs.AVSAddr)
		if err != nil {
			return false, "", 0, err
		}
		if _, isHandled := handledEpochMap[epochInfo.Identifier]; !isHandled {
			handledEpochMap[epochInfo.Identifier] = nil
		} else {
			continue
		}

		// calculate the remaining time of the current epoch
		remainingTime := uint64(epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration).Sub(ctx.BlockTime()))
		if remainingTime > maxRemainingTime {
			instantEpochIdentifier = epochInfo.Identifier
			instantEpochNumber = epochInfo.CurrentEpoch
			maxRemainingTime = remainingTime
		}
	}

	// get the expiration for normal undelegation
	normalUnbondingEpochID, normalUnbondingEpochNumber, unbondingDurationSec, err := k.GetUnbondingExpiration(ctx, operator)
	if err != nil {
		return false, "", 0, err
	}

	if maxRemainingTime < unbondingDurationSec {
		// Use the chosen completion time and apply a slash for instant unbonding
		// only if the calculated instant unbonding duration is shorter than the normal unbonding duration.
		return true, instantEpochIdentifier, instantEpochNumber, nil
	}
	return false, normalUnbondingEpochID, normalUnbondingEpochNumber, nil
}

// GetImpactfulEpochsAndAVSsForOperator gets the impactful epochs and AVSs for an operator.
// In the Imua chain, one operator can opt into multiple AVSs, and different AVSs might have different epoch
// configurations. This function returns the epochs and AVSs still served by the input operator.
func (k Keeper) GetImpactfulEpochsAndAVSsForOperator(ctx sdk.Context, operatorAddr string) ([]string, []string, error) {
	epochsMap := make(map[string]interface{}, 0)
	epochsList := make([]string, 0)
	avsList := make([]string, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		keys, err := assetstype.ParseJoinedStoreKey(key, 2)
		avsAddr := keys[1]
		if err != nil {
			return false, err
		}
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
		if err != nil {
			return false, err
		}
		// If the operator has opted out of an AVS, check whether the opted-out height is
		// less than the start height of the current epoch. If yes, the voting power has been
		// updated, and the operator no longer serves the AVS, so the epoch shouldn't be appended
		// to the list.
		if optedInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight &&
			optedInfo.OptedOutHeight < uint64(epochInfo.CurrentEpochStartHeight) {
			// continue addressing the other AVSs
			return false, nil
		}
		avsList = append(avsList, avsAddr)
		// filter the epoch using a map, because the multiple AVSs might have the save epoch identifier
		if _, ok := epochsMap[epochInfo.Identifier]; !ok {
			epochsMap[epochInfo.Identifier] = nil
			epochsList = append(epochsList, epochInfo.Identifier)
		}
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte(operatorAddr), opFunc)
	if err != nil {
		return nil, nil, err
	}

	return avsList, epochsList, nil
}

func (k Keeper) IsImpactfulEpochForOperator(ctx sdk.Context, epochIdentifier, operatorAddr string) bool {
	var isImpactfulEpoch bool
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		keys, err := assetstype.ParseJoinedStoreKey(key, 2)
		avsAddr := keys[1]
		if err != nil {
			return false, err
		}
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
		if err != nil {
			return false, err
		}

		// If the operator has opted out of an AVS, check whether the opted-out height is
		// less than the start height of the current epoch. If yes, the voting power has been
		// updated, and the operator no longer serves the AVS, so the epoch shouldn't be impactful.
		if optedInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight &&
			optedInfo.OptedOutHeight < uint64(epochInfo.CurrentEpochStartHeight) {
			// continue addressing the other AVSs
			return false, nil
		}
		if epochInfo.Identifier == epochIdentifier {
			// the avs is impactful and the epoch identifier is equal to the input identifier
			// break the iteration and return.
			isImpactfulEpoch = true
			return true, nil
		}
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte(operatorAddr), opFunc)
	if err != nil {
		return false
	}
	return isImpactfulEpoch
}

func (k *Keeper) SetAllOptedInfo(ctx sdk.Context, optedStates []operatortypes.OptedState) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	for i := range optedStates {
		state := optedStates[i]
		bz := k.cdc.MustMarshal(&state.OptInfo)
		store.Set([]byte(state.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllOptedInfo(ctx sdk.Context) ([]operatortypes.OptedState, error) {
	ret := make([]operatortypes.OptedState, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		ret = append(ret, operatortypes.OptedState{
			Key:     string(key),
			OptInfo: *optedInfo,
		})
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte{}, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *Keeper) GetOptedInOperatorListByAVS(ctx sdk.Context, avsAddr string) ([]string, error) {
	operatorList := make([]string, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
			keys, err := assetstype.ParseJoinedStoreKey(key, 2)
			if err != nil {
				return false, err
			}
			if strings.ToLower(avsAddr) == keys[1] {
				operatorList = append(operatorList, keys[0])
			}
		}
		return false, nil
	}
	err := k.IterateOptInfo(ctx, []byte{}, opFunc)
	if err != nil {
		return nil, err
	}
	return operatorList, nil
}

// IsUnbondingRelatedAVS checks whether there are no operators whose unbonding duration
// will be influenced by the AVS. The AVS can be deregistered if the condition is true. .
func (k *Keeper) IsUnbondingRelatedAVS(ctx sdk.Context, avsAddr string) bool {
	var isUnbondingRelatedAVS bool
	var err error
	avsAddr = strings.ToLower(avsAddr)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) (bool, error) {
		keys, err := assetstype.ParseJoinedStoreKey(key, 2)
		if err != nil {
			return false, err
		}
		if keys[1] == avsAddr {
			isUnbondingRelatedAVS, _, err = k.isUnbondingRelated(ctx, avsAddr, optedInfo)
			if err != nil {
				ctx.Logger().Error("IsUnbondingRelatedAVS: failed to check if the avs is unbonding related", "err", err, "avsAddr", avsAddr, "key", string(key))
				return true, err
			}
			if isUnbondingRelatedAVS {
				// return true to break the iteration
				return true, nil
			}
		}
		return false, nil
	}
	err = k.IterateOptInfo(ctx, []byte{}, opFunc)
	if err != nil {
		ctx.Logger().Error("IsUnbondingRelatedAVS: failed to iterate the opt info", "err", err)
		return true
	}
	return isUnbondingRelatedAVS
}
