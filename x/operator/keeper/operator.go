package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
)

// SetOperatorInfo is used to store the operator's information on the chain.
// There is no current way implemented to delete an operator's registration or edit it.
// TODO: implement operator edit function, which should allow editing:
// approve address?
// commission, subject to limits and once within 24 hours.
// client chain earnings addresses (maybe append only?)
func (k *Keeper) SetOperatorInfo(
	ctx sdk.Context, addr string, info *operatortypes.OperatorInfo,
) (err error) {
	// #nosec G703 // already validated in `ValidateBasic`
	opAccAddr, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return errorsmod.Wrap(err, "SetOperatorInfo: error occurred when parse acc address from Bech32")
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
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, avsAddr)
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
	return nil
}

func (k *Keeper) SetOptedInfo(ctx sdk.Context, operatorAddr, avsAddr string, info *operatortypes.OptedInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)

	// check operator address validation
	_, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return assetstype.ErrInvalidOperatorAddr
	}
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, avsAddr)

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
	infoKey := assetstype.GetJoinedStoreKey(operatorAddr, avsAddr)
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

func (k *Keeper) GetOptedInAVSForOperator(ctx sdk.Context, operatorAddr string) ([]string, error) {
	// get all opted-in info
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte(operatorAddr))
	defer iterator.Close()

	avsList := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		avsList = append(avsList, keys[1])
	}
	return avsList, nil
}

func (k Keeper) GetChainIDsForOperator(ctx sdk.Context, operatorAddr string) ([]string, error) {
	addrs, err := k.GetOptedInAVSForOperator(ctx, operatorAddr)
	if err != nil {
		return nil, err
	}
	chainIDs := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if chainID, found := k.avsKeeper.GetChainIDByAVSAddr(ctx, addr); found {
			chainIDs = append(chainIDs, chainID)
		}
	}
	return chainIDs, nil
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
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.OptedState, 0)
	for ; iterator.Valid(); iterator.Next() {
		var optedInfo operatortypes.OptedInfo
		k.cdc.MustUnmarshal(iterator.Value(), &optedInfo)
		ret = append(ret, operatortypes.OptedState{
			Key:     string(iterator.Key()),
			OptInfo: optedInfo,
		})
	}
	return ret, nil
}

func (k *Keeper) GetOptedInOperatorListByAVS(ctx sdk.Context, avsAddr string) ([]string, error) {
	// get all opted-in info
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorOptedAVSInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	operatorList := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		if avsAddr == keys[1] {
			operatorList = append(operatorList, keys[0])
		}
	}
	return operatorList, nil
}
