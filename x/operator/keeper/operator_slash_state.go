package keeper

import (
	"strings"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// UpdateOperatorSlashInfo This is a function to store the slash info related to an operator
// The stored state is: operator + '/' + AVSAddr + '/' + slashId -> OperatorSlashInfo
// Now this function will be called by `slash` function implemented in 'state_update.go' when there is a slash event occurs.
func (k *Keeper) UpdateOperatorSlashInfo(ctx sdk.Context, operatorAddr, avsAddr, slashID string, slashInfo operatortypes.OperatorSlashInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)

	// check operator address validation
	_, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return assetstype.ErrInvalidOperatorAddr
	}
	slashInfoKey := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr), slashID)
	if store.Has(slashInfoKey) {
		return errorsmod.Wrapf(operatortypes.ErrSlashInfoExist, "slashInfoKey:%s", slashInfoKey)
	}
	// check the validation of slash info
	slashContract, err := k.avsKeeper.GetAVSSlashContract(ctx, avsAddr)
	if err != nil {
		return err
	}
	if slashInfo.SlashContract != slashContract {
		return errorsmod.Wrapf(operatortypes.ErrSlashInfo, "err slashContract:%s, stored contract:%s", slashInfo.SlashContract, slashContract)
	}
	if slashInfo.EventHeight > slashInfo.SubmittedHeight {
		return errorsmod.Wrapf(operatortypes.ErrSlashInfo, "err SubmittedHeight:%v,EventHeight:%v", slashInfo.SubmittedHeight, slashInfo.EventHeight)
	}

	if slashInfo.SlashProportion.IsNil() || slashInfo.SlashProportion.IsNegative() || slashInfo.SlashProportion.GT(sdkmath.LegacyNewDec(1)) {
		return errorsmod.Wrapf(operatortypes.ErrSlashInfo, "err SlashProportion:%v", slashInfo.SlashProportion)
	}

	// save single operator delegation state
	bz := k.cdc.MustMarshal(&slashInfo)
	store.Set(slashInfoKey, bz)
	// TODO: add an event for the slash info
	return nil
}

// GetOperatorSlashInfo This is a function to retrieve the slash info related to an operator
// Now this function hasn't been called. In the future, it might be called by the grpc query.
// Additionally, it might be used when implementing the veto function
func (k *Keeper) GetOperatorSlashInfo(ctx sdk.Context, avsAddr, operatorAddr, slashID string) (changeState *operatortypes.OperatorSlashInfo, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	slashInfoKey := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr), slashID)
	value := store.Get(slashInfoKey)
	if value == nil {
		return nil, errorsmod.Wrapf(operatortypes.ErrNoKeyInTheStore, "GetOperatorSlashInfo: key is %s", slashInfoKey)
	}
	operatorSlashInfo := operatortypes.OperatorSlashInfo{}
	k.cdc.MustUnmarshal(value, &operatorSlashInfo)
	return &operatorSlashInfo, nil
}

// AllOperatorSlashInfo return all slash information for the specified operator and AVS
func (k *Keeper) AllOperatorSlashInfo(ctx sdk.Context, avsAddr, operatorAddr string) (map[string]*operatortypes.OperatorSlashInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	prefix := assetstype.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr))

	ret := make(map[string]*operatortypes.OperatorSlashInfo, 0)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var slashInfo operatortypes.OperatorSlashInfo
		k.cdc.MustUnmarshal(iterator.Value(), &slashInfo)
		keys, err := assetstype.ParseJoinedKey(iterator.Key())
		if err != nil {
			return nil, err
		}
		ret[keys[2]] = &slashInfo
	}
	return ret, nil
}

// UpdateSlashAssetsState This is a function to update the assets amount that need to be slashed
// The stored state is:
// KeyPrefixSlashAssetsState key-value:
// processedSlashHeight + '/' + assetID -> SlashAmount
// processedSlashHeight + '/' + assetID + '/' + stakerID -> SlashAmount
// processedSlashHeight + '/' + assetID + '/' + operatorAddr -> SlashAmount
// The slashed assets info won't be sent to the client chain immediately after the slash event being processed, env if
// the asset amounts of related operator and staker have been decreased. This is because we need to wait a veto period.
// The state updated by this function will be sent to the client chain once the veto period has expired.
// This function will be called by `SlashStaker` and `SlashOperator` implemented in the 'state_update.go' file.
func (k *Keeper) UpdateSlashAssetsState(ctx sdk.Context, assetID, stakerOrOperator string, processedHeight uint64, opAmount sdkmath.Int) error {
	if opAmount.IsNil() || opAmount.IsZero() {
		return nil
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashAssetsState)
	var key []byte
	if stakerOrOperator == "" || assetID == "" {
		return errorsmod.Wrapf(operatortypes.ErrParameterInvalid, "assetID:%s,stakerOrOperator:%s", assetID, stakerOrOperator)
	}

	key = assetstype.GetJoinedStoreKey(hexutil.EncodeUint64(processedHeight), assetID, stakerOrOperator)
	slashAmount := assetstype.ValueField{Amount: sdkmath.ZeroInt()}
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &slashAmount)
	}

	err := assetstype.UpdateAssetValue(&slashAmount.Amount, &opAmount)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&slashAmount)
	store.Set(key, bz)

	key = assetstype.GetJoinedStoreKey(hexutil.EncodeUint64(processedHeight), assetID)
	totalSlashAmount := assetstype.ValueField{Amount: sdkmath.ZeroInt()}
	value = store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &totalSlashAmount)
	}

	err = assetstype.UpdateAssetValue(&totalSlashAmount.Amount, &opAmount)
	if err != nil {
		return err
	}
	bz = k.cdc.MustMarshal(&slashAmount)
	store.Set(key, bz)
	return nil
}

// GetSlashAssetsState This is a function to retrieve the assets awaiting transfer to the client chain for slashing.
// Now this function hasn't been called, it might be called by the grpc query in the future.
// Additionally, this function might be called in the schedule function `EndBlock` to send the slash info to client chain.
// todo: It's to be determined about how to send the slash info to client chain. If we send them in `EndBlock`, then the native code needs to call the gateway contract deployed in imua. This seems a little bit odd.
func (k *Keeper) GetSlashAssetsState(ctx sdk.Context, assetID, stakerOrOperator string, processedHeight uint64) (sdkmath.Int, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashAssetsState)
	var key []byte
	if stakerOrOperator == "" {
		key = assetstype.GetJoinedStoreKey(hexutil.EncodeUint64(processedHeight), assetID)
	} else {
		key = assetstype.GetJoinedStoreKey(hexutil.EncodeUint64(processedHeight), assetID, stakerOrOperator)
	}
	value := store.Get(key)
	if value == nil {
		return sdkmath.Int{}, errorsmod.Wrapf(operatortypes.ErrNoKeyInTheStore, "GetSlashAssetsState: key is %s", key)
	}
	var ret assetstype.ValueField
	k.cdc.MustUnmarshal(value, &ret)

	return ret.Amount, nil
}

func (k *Keeper) SetAllSlashStates(ctx sdk.Context, slashStates []operatortypes.OperatorSlashState) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	for i := range slashStates {
		state := slashStates[i]
		bz := k.cdc.MustMarshal(&state.Info)
		store.Set([]byte(state.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllSlashStates(ctx sdk.Context) ([]operatortypes.OperatorSlashState, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.OperatorSlashState, 0)
	for ; iterator.Valid(); iterator.Next() {
		var slashInfo operatortypes.OperatorSlashInfo
		k.cdc.MustUnmarshal(iterator.Value(), &slashInfo)
		ret = append(ret, operatortypes.OperatorSlashState{
			Key:  string(iterator.Key()),
			Info: slashInfo,
		})
	}
	return ret, nil
}
