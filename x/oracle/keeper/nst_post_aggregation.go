package keeper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common/hexutil"
	utils "github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// deposit: update staker's totalDeposit
// withdoraw: update staker's totalDeposit
// delegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// undelegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// msg(refund or slash on beaconChain): update staker's price, operator's price

func (k Keeper) SetStakerInfosForAsset(ctx sdk.Context, chainID uint64, stakerInfos []*types.StakerInfo, version uint64) {
	store := ctx.KVStore(k.storeKey)

	lastIndex := uint32(0)
	for _, stakerInfo := range stakerInfos {
		// set staker balances
		keyBalances := types.NSTBalancesKey(chainID, stakerInfo.StakerAddr)
		balances := types.Balances{
			BalanceList: stakerInfo.BalanceList,
		}
		store.Set(keyBalances, k.cdc.MustMarshal(&balances))

		// set staker basic info
		keyStaker := types.NSTStakerKey(chainID, stakerInfo.StakerAddr)
		staker := types.Staker{
			StakerIndex:   stakerInfo.StakerIndex,
			ValidatorList: stakerInfo.ValidatorPubkeyList,
		}
		store.Set(keyStaker, k.cdc.MustMarshal(&staker))
		if stakerInfo.StakerIndex > lastIndex {
			lastIndex = stakerInfo.StakerIndex
		}
	}
	// set indexes for staker
	keyStakerIndex := types.NSTLatestStakerIndexKey(chainID)
	store.Set(keyStakerIndex, sdk.Uint64ToBigEndian(uint64(lastIndex)))

	// set version for assetID
	keyVersion := types.NSTVersionKey(chainID)
	store.Set(keyVersion, sdk.Uint64ToBigEndian(version))
}

// GetStakerInfo returns details about staker for native-restaking under asset of assetID
func (k Keeper) GetStakerInfo(ctx sdk.Context, chainID uint64, stakerAddr string) types.StakerInfo {
	store := ctx.KVStore(k.storeKey)
	stakerAddr = strings.ToLower(stakerAddr)

	keyStaker := types.NSTStakerKey(chainID, stakerAddr)
	value := store.Get(keyStaker)

	if value == nil {
		return types.StakerInfo{}
	}

	staker := &types.Staker{}
	k.cdc.MustUnmarshal(value, staker)

	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	value = store.Get(keyBalances)

	if value == nil {
		return types.StakerInfo{}
	}

	balances := &types.Balances{}
	k.cdc.MustUnmarshal(value, balances)

	return types.StakerInfo{
		StakerAddr:          stakerAddr,
		StakerIndex:         staker.StakerIndex,
		ValidatorPubkeyList: staker.ValidatorList,
		BalanceList:         balances.BalanceList,
	}
}

// GetStakerInfos returns all stakers information
func (k Keeper) GetStakerInfos(ctx sdk.Context, req *types.QueryStakerInfosRequest) (*types.QueryStakerInfosResponse, error) {
	if req.Pagination != nil && req.Pagination.Limit > types.MaxPageLimit {
		return nil, status.Errorf(codes.InvalidArgument, "pagination limit %d exceeds maximum allowed %d", req.Pagination.Limit, types.MaxPageLimit)
	}

	_, chainID, _ := assetstypes.ParseID(strings.ToLower(req.AssetId))
	store := ctx.KVStore(k.storeKey)
	// retrieve version
	bz := store.Get(types.NSTVersionKey(chainID))
	version := uint64(0)
	if bz != nil {
		version = sdk.BigEndianToUint64(bz)
	}
	storePrefix := prefix.NewStore(store, types.NSTStakerKeyChainIDPrefix(chainID))
	retStakerInfos := make([]*types.StakerInfo, 0)
	resPage, err := query.Paginate(storePrefix, req.Pagination, func(key []byte, value []byte) error {
		retStakerInfo, err := k.getStakerInfos(store, types.NSTBalancesKeyChainIDPrefix(chainID), key, value, req.BalancesAll)
		if err != nil {
			return err
		}
		retStakerInfos = append(retStakerInfos, retStakerInfo)
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}
	return &types.QueryStakerInfosResponse{
		// TODO: update type to uint64 to avoid confusion
		// #nosec G115
		Version:     int64(version),
		StakerInfos: retStakerInfos,
		Pagination:  resPage,
	}, nil
}

func (k Keeper) getStakerInfos(store sdk.KVStore, balancesKeyPrefix, key, value []byte, all bool) (*types.StakerInfo, error) {
	staker := &types.Staker{}
	k.cdc.MustUnmarshal(value, staker)
	keyBalances := append(balancesKeyPrefix, key...)
	value = store.Get(keyBalances)
	if value == nil {
		return nil, status.Errorf(codes.NotFound, "staker %s not found", string(key))
	}
	balances := &types.Balances{}
	k.cdc.MustUnmarshal(value, balances)

	stakerInfo := types.StakerInfo{
		StakerAddr:          string(key),
		StakerIndex:         staker.StakerIndex,
		ValidatorPubkeyList: staker.ValidatorList,
	}
	if len(balances.BalanceList) > 0 {
		if all {
			stakerInfo.BalanceList = balances.BalanceList
		} else {
			stakerInfo.BalanceList = balances.BalanceList[len(balances.BalanceList)-1:]
		}
	}
	return &stakerInfo, nil
}

// GetAllStakerInfosAssets returns all stakerInfos combined with assetIDs they belong to, used for genesisstate exporting
func (k Keeper) GetAllStakerInfosAssets(ctx sdk.Context) ([]types.StakerInfosAssets, error) {
	store := ctx.KVStore(k.storeKey)
	storePrefix := prefix.NewStore(store, []byte(types.NSTVersionKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(storePrefix, []byte{})
	ret := make([]types.StakerInfosAssets, 0)
	for ; iterator.Valid(); iterator.Next() {
		chainID, err := hexutil.DecodeUint64(string(iterator.Key()))
		if err != nil {
			return nil, err
		}
		iteratorStakers := sdk.KVStorePrefixIterator(store, types.NSTStakerKeyChainIDPrefix(chainID))
		stakerInfos := make([]*types.StakerInfo, 0)
		for ; iteratorStakers.Valid(); iteratorStakers.Next() {
			stakerInfo, err := k.getStakerInfos(store, types.NSTBalancesKeyChainIDPrefix(chainID), iteratorStakers.Key(), iteratorStakers.Value(), true)
			if err != nil {
				return nil, err
			}
			stakerInfos = append(stakerInfos, stakerInfo)
		}
		version := sdk.BigEndianToUint64(iterator.Value())
		ret = append(ret, types.StakerInfosAssets{
			// #nosec G115
			NstVersion:  int64(version),
			ChainId:     chainID,
			StakerInfos: stakerInfos,
		})
	}

	return ret, nil
}

func (k Keeper) getStakerListNoCache(ctx sdk.Context, assetID string) types.StakerList {
	_, chainID, _ := assetstypes.ParseID(assetID)
	store := ctx.KVStore(k.storeKey)
	keyStakerAddrPrefix := types.NSTStakerAddrKeyChainIDPrefix(chainID)
	iterator := sdk.KVStorePrefixIterator(store, keyStakerAddrPrefix)
	defer iterator.Close()
	stakerList := types.StakerList{
		StakerAddrs: make([]string, 0),
	}
	for ; iterator.Valid(); iterator.Next() {
		stakerAddr := string(iterator.Value())
		if stakerAddr == "" {
			continue
		}
		stakerList.StakerAddrs = append(stakerList.StakerAddrs, stakerAddr)
	}
	return stakerList
}

// GetStakerList return stakerList for native-restaking asset of assetID
// add cache
func (k Keeper) GetStakerList(ctx sdk.Context, assetID string) types.StakerList {
	_, chainID, _ := assetstypes.ParseID(assetID)
	if sl := k.c.GetNSTStakerList(chainID); sl != nil {
		return types.StakerList{
			StakerAddrs: sl,
		}
	}
	stakerList := k.getStakerListNoCache(ctx, assetID)
	// update cache
	k.c.SetNSTStakerList(chainID, stakerList.StakerAddrs)
	return stakerList
}

// handle deposit from assetsModule
func (k Keeper) UpdateNSTValidatorListForStaker(ctx sdk.Context, assetID, stakerID, validatorPubkey string, amount sdkmath.Int) error {
	if amount.LT(sdkmath.ZeroInt()) {
		return errors.New("amount should be positive")
	}
	stakerID = strings.ToLower(stakerID)
	stakerAddr, chainID, _ := assetstypes.ParseID(stakerID)

	feederID, ok := k.GetNSTFeederIDFromClientChainID(chainID)
	if !ok {
		return errors.New("failed to get corresponding feederID from clientChainID")
	}

	amountConverted, err := k.convertDecimal(ctx, assetID, amount, feederID, true)

	if err != nil {
		return err
	}

	_, index, _, err := k.updateStaker(ctx, chainID, 0, amountConverted.Uint64(), stakerAddr, validatorPubkey, types.Action_ACTION_DEPOSIT)
	if err != nil {
		return err
	}

	version := k.GetNSTVersion(ctx, chainID)
	// we use index to sync with client about status of stakerInfo.ValidatorPubkeyList
	eventValue := fmt.Sprintf("%s_%d_%s_%d_%d_%d", types.AttributeValueNativeTokenDeposit, index, validatorPubkey, version, amountConverted.Uint64(), feederID)
	if len(*k.cachedNSTStakersEventValue) > 0 {
		*k.cachedNSTStakersEventValue += types.DelimiterForBase64
	}
	if !ctx.IsCheckTx() {
		*k.cachedNSTStakersEventValue += eventValue
	}
	return nil
}

// SetNSTVersion increases the version of native token for assetID
func (k Keeper) SetNSTVersion(ctx sdk.Context, assetID string, version int64) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NativeTokenVersionKey(assetID)
	// #nosec version is not negative
	store.Set(key, sdk.Uint64ToBigEndian(uint64(version)))
	return version
}

func (k Keeper) GetNSTVersionFromAssetID(ctx sdk.Context, assetID string) uint64 {
	_, chainID, _ := assetstypes.ParseID(strings.ToLower(assetID))
	return k.GetNSTVersion(ctx, chainID)
}

func (k Keeper) GetNSTVersion(ctx sdk.Context, chainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		return 0
	}
	return sdk.BigEndianToUint64(value)
}

func (k Keeper) getDecimal(ctx sdk.Context, assetID string) (int, sdkmath.Int, error) {
	decimalMap, err := k.assetsKeeper.GetAssetsDecimal(ctx, map[string]any{assetID: nil})
	if err != nil {
		return 0, sdkmath.ZeroInt(), err
	}
	decimal := decimalMap[assetID]
	return int(decimal), sdkmath.NewIntWithDecimal(1, int(decimal)), nil
}

// when the balance of staker became zero, we remove it from the staker list and the related data
// return value is the former last staker which had been moved ahead and its updated index
func (k Keeper) removeStaker(ctx sdk.Context, chainID uint64, stakerAddr string) (string, uint32) {
	latestStakerIndex, found := k.GetLatestStakerIndex(ctx, chainID)
	if !found {
		return "", 0
	}

	store := ctx.KVStore(k.storeKey)
	keyStaker := types.NSTStakerKey(chainID, stakerAddr)
	staker := types.Staker{}
	bz := store.Get(keyStaker)
	if bz == nil {
		return "", 0
	}

	k.cdc.MustUnmarshal(bz, &staker)
	removedIndex := staker.StakerIndex
	// remove staker basic info
	store.Delete(keyStaker)

	// remove balanceList
	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	store.Delete(keyBalances)

	// update latest staker index
	// checked staker exists early, so latestStakerIndex must be positive
	var stakerAddrFrom string
	var toIndex int
	if latestStakerIndex > removedIndex {
		// move latest index ahead
		stakerAddrFrom = string(store.Get(types.NSTStakerAddrKey(chainID, latestStakerIndex)))
		keyStaker = types.NSTStakerKey(chainID, stakerAddrFrom)
		bz = store.Get(keyStaker)
		staker = types.Staker{}
		k.cdc.MustUnmarshal(bz, &staker)
		staker.StakerIndex = removedIndex
		store.Set(keyStaker, k.cdc.MustMarshal(&staker))

		// update index
		k.SetStakerIndex(ctx, chainID, removedIndex, stakerAddrFrom)
		toIndex = int(removedIndex)
	} else {
		// delete index
		store.Delete(types.NSTStakerAddrKey(chainID, removedIndex))
		stakerAddrFrom = stakerAddr
		toIndex = -1
	}

	if !k.c.UpdateNSTStakerList(chainID, toIndex, int(latestStakerIndex), stakerAddrFrom, stakerAddr) {
		sl := k.getStakerListNoCache(ctx, types.NSTAssetIDFromClientChainID(chainID))
		if len(sl.StakerAddrs) > 0 {
			k.c.SetNSTStakerList(chainID, sl.StakerAddrs)
		} else {
			k.c.RemoveNSTStakerList(chainID)
		}
	}

	if latestStakerIndex == 0 {
		store.Delete(types.NSTLatestStakerIndexKey(chainID))
	} else {
		latestStakerIndex--
		store.Set(types.NSTLatestStakerIndexKey(chainID), types.Uint32Bytes(latestStakerIndex))
	}

	k.IncreaseVersion(ctx, chainID)
	return stakerAddr, removedIndex
}

// updateStaker updates the staker's info including: validator list, balance, index, version of assets
func (k Keeper) updateStaker(ctx sdk.Context, chainID, roundID, balance uint64, stakerAddr string, validator string, action types.Action) (updatedStakerAddr string, updatedIndex uint32, balanceDelta sdkmath.Int, err error) {
	store := ctx.KVStore(k.storeKey)
	if action == types.Action_ACTION_SLASH_REFUND && balance == 0 {
		// this is a special case, we need to remove the staker from the list
		// and update the index of the last staker
		keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
		bz := store.Get(keyBalances)
		if bz == nil {
			return
		}

		balances := &types.Balances{}
		k.cdc.MustUnmarshal(bz, balances)
		if len(balances.BalanceList) > 0 {
			balanceDelta = sdkmath.NewIntFromUint64(balances.BalanceList[len(balances.BalanceList)-1].Balance)
		}
		updatedStakerAddr, updatedIndex = k.removeStaker(ctx, chainID, stakerAddr)
		return
	}

	if action == types.Action_ACTION_DEPOSIT && len(validator) == 0 {
		err = fmt.Errorf("deposit should have one validator, but got %d", len(validator))
		return
	}

	stakerInfo := k.GetStakerInfo(ctx, chainID, stakerAddr)
	if action != types.Action_ACTION_DEPOSIT && (stakerInfo.StakerAddr == "" || len(stakerInfo.BalanceList) == 0) {
		err = fmt.Errorf("staker or balanceList is not found, stakerAddr is empty: %t, balanceList is empty: %t, action: %s",
			stakerInfo.StakerAddr == "", len(stakerInfo.BalanceList) == 0, action)
		return
	}

	newBalance := &types.BalanceInfo{
		RoundID: roundID,
		Block:   uint64(ctx.BlockHeight()),
		Change:  action,
	}
	staker := &types.Staker{
		ValidatorList: stakerInfo.ValidatorPubkeyList,
	}
	balanceDelta = sdkmath.NewIntFromUint64(balance)
	if stakerInfo.StakerAddr == "" {
		// update latest staker index
		latestIndex := k.IncreaseLatestStakerIndex(ctx, chainID)

		staker.StakerIndex = latestIndex
		// set index for stakerAddress
		k.SetStakerIndex(ctx, chainID, latestIndex, stakerAddr)
	} else {
		// update staker with new validator
		staker.StakerIndex = stakerInfo.StakerIndex
		// this should always be true
		if len(stakerInfo.BalanceList) > 0 {
			newBalance.Index = stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].Index
			if action == types.Action_ACTION_DEPOSIT {
				newBalance.Balance = stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].Balance
				newBalance.RoundID = stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].RoundID
			} else {
				balanceDelta = balanceDelta.Sub(sdkmath.NewIntFromUint64(stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].Balance))
			}
		}
	}

	// set staker
	if action == types.Action_ACTION_DEPOSIT {
		staker.ValidatorList = append(staker.ValidatorList, validator)
	}
	k.SetStaker(ctx, chainID, stakerAddr, staker)

	// set balanceList
	newBalance.Index++
	newBalance.Balance += balance
	balances := &types.Balances{
		BalanceList: append(stakerInfo.BalanceList, newBalance),
	}
	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	bz := k.cdc.MustMarshal(balances)
	store.Set(keyBalances, bz)

	// increase version
	k.IncreaseVersion(ctx, chainID)
	return
}

func (k Keeper) IncreaseVersion(ctx sdk.Context, chainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		store.Set(key, sdk.Uint64ToBigEndian(1))
		return 1
	}
	version := sdk.BigEndianToUint64(value) + 1
	store.Set(key, sdk.Uint64ToBigEndian(version))
	return version
}

func (k Keeper) SetStakerIndex(ctx sdk.Context, chainID uint64, index uint32, stakerAddr string) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTStakerAddrKey(chainID, index)
	store.Set(key, []byte(stakerAddr))
}

func (k Keeper) GetLatestStakerIndex(ctx sdk.Context, chainID uint64) (uint32, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTLatestStakerIndexKey(chainID)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	return types.BytesToUint32(bz), true
}

func (k Keeper) IncreaseLatestStakerIndex(ctx sdk.Context, chainID uint64) uint32 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTLatestStakerIndexKey(chainID)
	bz := store.Get(key)
	if bz == nil {
		store.Set(key, types.Uint32Bytes(0))
		return 0
	}
	latestStakerIndex := types.BytesToUint32(bz)
	latestStakerIndex++
	store.Set(key, types.Uint32Bytes(latestStakerIndex))
	return latestStakerIndex
}

func (k Keeper) SetStaker(ctx sdk.Context, chainID uint64, stakerAddr string, staker *types.Staker) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(staker)
	keyStaker := types.NSTStakerKey(chainID, stakerAddr)
	store.Set(keyStaker, bz)
}

// if fromAssetsMoule=false, it means the opposite way: oracleModule->assetModule
func (k Keeper) convertDecimal(ctx sdk.Context, assetID string, amount sdkmath.Int, feederID uint64, fromAssetsModule bool) (sdkmath.Int, error) {
	decimalMap, err := k.assetsKeeper.GetAssetsDecimal(ctx, map[string]any{assetID: nil})
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	// #nosec G115
	fromDecimal := int32(decimalMap[assetID])
	toDecimal, err := k.GetDecimalFromFeederID(feederID)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	if fromDecimal == toDecimal {
		return amount, nil
	}
	if !fromAssetsModule {
		fromDecimal, toDecimal = toDecimal, fromDecimal
	}
	amountlegacy := sdkmath.LegacyNewDecFromInt(amount)
	if toDecimal > fromDecimal {
		delta := int64(toDecimal - fromDecimal)
		retDec := amountlegacy.Quo(sdkmath.LegacyNewDecWithPrec(1, delta))
		if !retDec.IsInteger() {
			return sdkmath.ZeroInt(), errors.New("conversion lost precision")
		}
		return retDec.RoundInt(), nil
	}
	// toDecimal < fromDecimal
	delta := int64(fromDecimal - toDecimal)
	retDec := amountlegacy.Mul(sdkmath.LegacyNewDecWithPrec(1, delta))
	if retDec.LT(sdkmath.LegacyOneDec()) {
		return sdkmath.ZeroInt(), errors.New("convert amount to 0 for converting to too many decimals")
	}
	if !retDec.IsInteger() {
		return sdkmath.ZeroInt(), errors.New("conversion lost precision")
	}
	return retDec.RoundInt(), nil
}

// this is called in EndBlock not as a part of transaction, so the 'error' will not revert process
// UpdateNSTBalanceChange serves the post handling for nst balance change
func UpdateNSTBalanceChange(ctx sdk.Context, rootHash []byte, rawData []byte, feederID, roundID uint64, kInf common.KeeperOracle) error {
	balanceChanges := &types.RawDataNST{}
	kInf.MustUnmarshal(rawData, balanceChanges)
	k, ok := kInf.(*Keeper)
	if !ok {
		return errors.New("input keeper interface type error")
	}
	assetID, chainIDStr := k.GetParamsFromCache().GetAssetIDForNSTFromFeederID(feederID)
	chainID, _ := hexutil.DecodeUint64(chainIDStr)
	// TODO(leonz): use uint64 for version state
	// #nosec G115
	v := k.GetNSTVersion(ctx, chainID)
	if balanceChanges.Version != v {
		return fmt.Errorf("version not match, expected %d, got %d, assetID:%s", v, balanceChanges.Version, assetID)
	}
	sl := k.GetStakerList(ctx, assetID)
	if len(sl.StakerAddrs) == 0 {
		return errors.New("staker list is empty")
	}

	cc, writeCache := ctx.CacheContext()

	for _, changeKV := range balanceChanges.NstBalanceChanges {
		stakerAddr := sl.StakerAddrs[changeKV.StakerIndex]
		_, _, balanceDelta, err := k.updateStaker(cc, chainID, roundID, changeKV.Balance, stakerAddr, "", types.Action_ACTION_SLASH_REFUND)
		if err != nil {
			return err
		}
		if err := k.delegationKeeper.UpdateNSTBalance(cc, getStakerID(stakerAddr, chainID), assetID, balanceDelta); err != nil {
			return err
		}
	}
	writeCache()
	version := k.IncreaseVersion(ctx, chainID)
	base64.StdEncoding.EncodeToString(rootHash)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNSTBalanceUpdate, types.AttributeValueTrue),
		sdk.NewAttribute(types.AttributeKeyNSTBalanceChange, fmt.Sprintf("%s|%d|%d", base64.StdEncoding.EncodeToString(rootHash), version, feederID)),
	))
	return nil
}

// TODO use []byte and assetstypes.GetStakerIDAndAssetID for stakerAddr representation
func getStakerID(stakerAddr string, chainID uint64) string {
	return strings.Join([]string{strings.ToLower(stakerAddr), hexutil.EncodeUint64(chainID)}, utils.DelimiterForID)
}
