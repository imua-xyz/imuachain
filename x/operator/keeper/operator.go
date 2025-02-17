package keeper

import (
	"fmt"
	"math"
	"strings"

	epochtypes "github.com/ExocoreNetwork/exocore/x/epochs/types"

	"golang.org/x/xerrors"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
)

// SetOperatorInfo is used to store the operator's information on the chain.
// There is no current way implemented to delete an operator's registration or edit it.
// TODO: implement operator edit function, which should allow editing:
// approve address?
// name (meta info)
// commission, subject to limits and once within 24 hours.
// client chain earnings addresses (maybe append only?)
func (k *Keeper) SetOperatorInfo(
	ctx sdk.Context, addr string, info *operatortypes.OperatorInfo,
) (err error) {
	if info == nil {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "SetOperatorInfo: operator info is nil")
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
	// TODO: EditOperator needs to be implemented.
	if k.IsOperator(ctx, opAccAddr) {
		return errorsmod.Wrap(
			operatortypes.ErrOperatorAlreadyExists,
			fmt.Sprintf("SetOperatorInfo: operator already exists, address: %s", opAccAddr),
		)
	}
	// TODO: add minimum commission rate module parameter and check that commission exceeds it.
	info.Commission.UpdateTime = ctx.BlockTime()

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

	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	bz := k.cdc.MustMarshal(info)
	store.Set(opAccAddr, bz)

	// TODO validate operator name does not already exist

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

// AllOperators return the list of all operators' detailed information
func (k *Keeper) AllOperators(ctx sdk.Context) []operatortypes.OperatorDetail {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]operatortypes.OperatorDetail, 0)
	for ; iterator.Valid(); iterator.Next() {
		var operatorInfo operatortypes.OperatorInfo
		operatorAddr := sdk.AccAddress(iterator.Key())
		k.cdc.MustUnmarshal(iterator.Value(), &operatorInfo)
		ret = append(ret, operatortypes.OperatorDetail{
			OperatorAddress: operatorAddr.String(),
			OperatorInfo:    operatorInfo,
		})
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

func (k *Keeper) IsActive(ctx sdk.Context, operatorAddr sdk.AccAddress, avsAddr string) bool {
	optedInfo, err := k.GetOptedInfo(ctx, operatorAddr.String(), avsAddr)
	if err != nil {
		// not opted in
		return false
	}
	if optedInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight {
		// opted out
		return false
	}
	if optedInfo.Jailed {
		// frozen - either temporarily or permanently
		return false
	}
	return true
}

func (k *Keeper) IterateOptInfo(ctx sdk.Context, isUpdate bool, iteratePrefix []byte, opFunc func(key []byte, optedInfo *operatortypes.OptedInfo) error) error {
	// get all opted-in info
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	iterator := sdk.KVStorePrefixIterator(store, iteratePrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var optedInfo operatortypes.OptedInfo
		k.cdc.MustUnmarshal(iterator.Value(), &optedInfo)
		err := opFunc(iterator.Key(), &optedInfo)
		if err != nil {
			return err
		}
		if isUpdate {
			bz := k.cdc.MustMarshal(&optedInfo)
			store.Set(iterator.Key(), bz)
		}
	}
	return nil
}

func (k *Keeper) GetOptedInAVSForOperator(ctx sdk.Context, operatorAddr string) ([]string, error) {
	avsList := make([]string, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) error {
		if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
			keys, err := assetstype.ParseJoinedStoreKey(key, 2)
			if err != nil {
				return err
			}
			avsList = append(avsList, keys[1])
		}
		return nil
	}
	err := k.IterateOptInfo(ctx, false, []byte(operatorAddr), opFunc)
	if err != nil {
		return nil, err
	}
	return avsList, nil
}

func (k *Keeper) GetImpactfulAVSForOperator(ctx sdk.Context, operatorAddr string) ([]operatortypes.ImpactfulAVSInfo, error) {
	avsList := make([]operatortypes.ImpactfulAVSInfo, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) error {
		keys, err := assetstype.ParseJoinedStoreKey(key, 2)
		avsAddr := keys[1]
		if err != nil {
			return err
		}
		// add AVS currently opting in to the operator's list.
		if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
			avsList = append(avsList, operatortypes.ImpactfulAVSInfo{AVSAddr: avsAddr})
		} else {
			// Add AVS that have opted out but are still within the unbonding duration,
			// and therefore still affect the operator, to the list.
			// #nosec G115
			epochNumber, err := k.GetEpochNumberByOptOutHeight(ctx, avsAddr, int64(optedInfo.OptedOutHeight))
			if err != nil {
				return err
			}
			epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
			if err != nil {
				return err
			}
			if epochInfo.CurrentEpoch < epochNumber {
				return xerrors.Errorf("GetImpactfulAVSForOperator: current epoch number is less than the retrieved epoch number by opted out height,avsAddr:%s,epochIdentifier:%s,currentEpochNumber:%d,optedOutEpochNumber:%d", avsAddr, epochInfo.Identifier, epochInfo.CurrentEpoch, epochNumber)
			}
			unbondingDuration, err := k.avsKeeper.GetAVSUnbondingDuration(ctx, avsAddr)
			if err != nil {
				return err
			}
			if epochNumber > 0 && uint64(epochNumber)+unbondingDuration >= uint64(epochInfo.CurrentEpoch) {
				avsList = append(avsList, operatortypes.ImpactfulAVSInfo{
					AVSAddr:                  avsAddr,
					HasOptedOut:              true,
					OptOutUnbondingRemaining: uint64(epochNumber) + unbondingDuration - uint64(epochInfo.CurrentEpoch),
				})
			}
		}
		return nil
	}
	err := k.IterateOptInfo(ctx, false, []byte(operatorAddr), opFunc)
	if err != nil {
		return nil, err
	}
	return avsList, nil
}

func (k Keeper) GetUnbondingExpiration(ctx sdk.Context, operator sdk.AccAddress) (string, int64, error) {
	// get the impactful AVSs for the operator
	avsList, err := k.GetImpactfulAVSForOperator(ctx, operator.String())
	if err != nil {
		return "", 0, err
	}
	// calculate the maximum unbonding expiration
	// Using self-definied NullEpochIdentifier and NullEpochNumber as the default unbonding expiration.
	retEpochIdentifier := epochtypes.NullEpochIdentifier
	retEpochNumber := epochtypes.NullEpochNumber
	maxDurationSeconds := uint64(0)
	for _, avs := range avsList {
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avs.AVSAddr)
		if err != nil {
			return "", 0, err
		}
		unbondingDuration, err := k.avsKeeper.GetAVSUnbondingDuration(ctx, avs.AVSAddr)
		if err != nil {
			return "", 0, err
		}
		if avs.HasOptedOut {
			unbondingDuration = avs.OptOutUnbondingRemaining
		}
		if unbondingDuration+uint64(epochInfo.CurrentEpoch) > uint64(math.MaxInt64) {
			return "", 0, xerrors.New("the sum of unbondingDuration and the current epoch number exceeds the int64 range.")
		}
		durationSeconds := unbondingDuration * uint64(epochInfo.Duration)
		// address the case that the unbonding time is the end of current epoch.
		if unbondingDuration == 0 {
			durationSeconds = uint64(epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration).Sub(ctx.BlockTime()))
		}
		if durationSeconds > maxDurationSeconds {
			retEpochIdentifier = epochInfo.Identifier
			retEpochNumber = epochInfo.CurrentEpoch + int64(unbondingDuration)
			maxDurationSeconds = durationSeconds
		}
	}
	return retEpochIdentifier, retEpochNumber, nil
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
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) error {
		ret = append(ret, operatortypes.OptedState{
			Key:     string(key),
			OptInfo: *optedInfo,
		})
		return nil
	}
	err := k.IterateOptInfo(ctx, false, []byte{}, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *Keeper) GetOptedInOperatorListByAVS(ctx sdk.Context, avsAddr string) ([]string, error) {
	operatorList := make([]string, 0)
	opFunc := func(key []byte, optedInfo *operatortypes.OptedInfo) error {
		if optedInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
			keys, err := assetstype.ParseJoinedStoreKey(key, 2)
			if err != nil {
				return err
			}
			if strings.ToLower(avsAddr) == keys[1] {
				operatorList = append(operatorList, keys[0])
			}
		}
		return nil
	}
	err := k.IterateOptInfo(ctx, false, []byte{}, opFunc)
	if err != nil {
		return nil, err
	}
	return operatorList, nil
}
