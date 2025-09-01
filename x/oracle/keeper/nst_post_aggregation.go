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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	utils "github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// deposit: update staker's totalDeposit
// withdoraw: update staker's totalDeposit
// delegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// undelegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// msg(refund or slash on beaconChain): update staker's price, operator's price
// SetStakerInfosForAsset sets the staker information and balances for a given asset (chainID),
// and updates the latest staker index and version. Used during aggregation or state sync.
func (k Keeper) SetStakerInfosForAsset(ctx sdk.Context, chainID uint64, stakerInfos []*types.StakerInfo, version types.NSTVersion) {
	store := ctx.KVStore(k.storeKey)

	lastIndex := uint32(0)
	for _, stakerInfo := range stakerInfos {
		// set staker balances
		stakerAddr := strings.ToLower(stakerInfo.StakerAddr)
		keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
		balances := types.Balances{
			BalanceList: stakerInfo.BalanceList,
		}
		store.Set(keyBalances, k.cdc.MustMarshal(&balances))

		// set staker basic info
		keyStaker := types.NSTStakerKey(chainID, stakerAddr)
		staker := types.Staker{
			StakerIndex:   stakerInfo.StakerIndex,
			ValidatorList: stakerInfo.ValidatorList,
		}
		store.Set(keyStaker, k.cdc.MustMarshal(&staker))
		if stakerInfo.StakerIndex > lastIndex {
			lastIndex = stakerInfo.StakerIndex
		}
		keyStakerIndex := types.NSTStakerAddrKey(chainID, staker.StakerIndex)
		store.Set(keyStakerIndex, []byte(stakerAddr))
	}
	// set indexes for staker
	keyStakerIndex := types.NSTLatestStakerIndexKey(chainID)
	store.Set(keyStakerIndex, types.Uint32Bytes(lastIndex))

	// set version for assetID
	keyVersion := types.NSTVersionKey(chainID)
	bz := k.cdc.MustMarshal(&version)
	store.Set(keyVersion, bz)
}

// GetStakerInfo returns details about a staker for native-restaking under a specific asset (chainID).
// Returns an empty StakerInfo if not found or if balances are missing.
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
		// this should not happen, if balanceList is nil, the corresponding staker should not exist
		return types.StakerInfo{}
	}

	balances := &types.Balances{}
	k.cdc.MustUnmarshal(value, balances)

	return types.StakerInfo{
		StakerAddr:    stakerAddr,
		StakerIndex:   staker.StakerIndex,
		ValidatorList: staker.ValidatorList,
		BalanceList:   balances.BalanceList,
	}
}

// GetStakerInfos returns all stakers' information for a given asset, with optional pagination and balance history.
// Used for queries and state export.
func (k Keeper) GetStakerInfos(ctx sdk.Context, req *types.QueryStakerInfosRequest) (*types.QueryStakerInfosResponse, error) {
	if req.Pagination != nil && req.Pagination.Limit > types.MaxPageLimit {
		return nil, status.Errorf(codes.InvalidArgument, "pagination limit %d exceeds maximum allowed %d", req.Pagination.Limit, types.MaxPageLimit)
	}

	_, chainID, _ := assetstypes.ParseID(strings.ToLower(req.AssetId))
	versions, found := k.GetNSTVersionsFromAssetID(ctx, req.AssetId)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("version for assetID:%s not found", req.AssetId))
	}

	store := ctx.KVStore(k.storeKey)
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
		Version:     &versions,
		StakerInfos: retStakerInfos,
		Pagination:  resPage,
	}, nil
}

// getStakerInfos is a helper to retrieve a single staker's info and balances from the store.
// If 'all' is true, returns full balance history; otherwise, only the latest.
func (k Keeper) getStakerInfos(store sdk.KVStore, balancesKeyPrefix, key, value []byte, all bool) (*types.StakerInfo, error) {
	if value == nil {
		return nil, status.Errorf(codes.NotFound, "staker %s not found", string(key))
	}
	staker := &types.Staker{}
	k.cdc.MustUnmarshal(value, staker)

	var keyBalances []byte
	keyBalances = types.AppendMultiple(keyBalances, balancesKeyPrefix, key)
	value = store.Get(keyBalances)

	if value == nil {
		return nil, status.Errorf(codes.NotFound, "balanceList of staker %s not found", string(key))
	}

	stakerInfo := types.StakerInfo{
		StakerAddr:      string(key),
		StakerIndex:     staker.StakerIndex,
		ValidatorList:   staker.ValidatorList,
		WithdrawVersion: staker.WithdrawVersion,
	}

	balances := &types.Balances{}
	k.cdc.MustUnmarshal(value, balances)
	// this should always be true
	if len(balances.BalanceList) > 0 {
		if all {
			stakerInfo.BalanceList = balances.BalanceList
		} else {
			stakerInfo.BalanceList = balances.BalanceList[len(balances.BalanceList)-1:]
		}
	}
	return &stakerInfo, nil
}

// GetAllStakerInfosAssets returns all stakerInfos grouped by asset (chainID), used for genesis state export.
func (k Keeper) GetAllStakerInfosAssets(ctx sdk.Context) ([]types.StakerInfosAssets, error) {
	store := ctx.KVStore(k.storeKey)
	storePrefix := prefix.NewStore(store, []byte(types.NSTVersionKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(storePrefix, []byte{})
	ret := make([]types.StakerInfosAssets, 0)
	for ; iterator.Valid(); iterator.Next() {
		chainID, err := types.BytesToUint64(iterator.Key())
		if err != nil {
			return nil, fmt.Errorf("failed to parse chainID from key: %w", err)
		}
		storePrefixStaker := prefix.NewStore(store, types.NSTStakerKeyChainIDPrefix(chainID))
		iteratorStakers := sdk.KVStorePrefixIterator(storePrefixStaker, []byte{})
		stakerInfos := make([]*types.StakerInfo, 0)
		for ; iteratorStakers.Valid(); iteratorStakers.Next() {
			stakerInfo, err := k.getStakerInfos(store, types.NSTBalancesKeyChainIDPrefix(chainID), iteratorStakers.Key(), iteratorStakers.Value(), true)
			if err != nil {
				return nil, err
			}
			stakerInfos = append(stakerInfos, stakerInfo)
		}
		version := types.NSTVersion{}
		k.cdc.MustUnmarshal(iterator.Value(), &version)
		ret = append(ret, types.StakerInfosAssets{
			// #nosec G115
			NstVersionInfo: version,
			ChainId:        chainID,
			StakerInfos:    stakerInfos,
		})
	}

	return ret, nil
}

// getStakerListNoCache retrieves the list of staker addresses for an asset (chainID) directly from the store (no cache).
func (k Keeper) getStakerListNoCache(ctx sdk.Context, assetID string, chainID uint64) types.StakerList {
	if chainID == 0 {
		if len(assetID) == 0 {
			return types.StakerList{}
		}
		_, chainID, _ = assetstypes.ParseID(assetID)
		if chainID == 0 {
			return types.StakerList{}
		}
	}
	store := ctx.KVStore(k.storeKey)
	keyStakerAddrPrefix := types.NSTStakerAddrKeyChainIDPrefix(chainID)
	storePrefix := prefix.NewStore(store, keyStakerAddrPrefix)
	iterator := sdk.KVStorePrefixIterator(storePrefix, []byte{})
	defer iterator.Close()
	stakerList := types.StakerList{
		Stakers: make([]*types.StakerListEntry, 0),
	}
	for ; iterator.Valid(); iterator.Next() {
		stakerAddr := string(iterator.Value())
		if stakerAddr == "" {
			continue
		}
		stakerKey := types.NSTStakerKey(chainID, stakerAddr)
		bz := store.Get(stakerKey)
		if bz == nil {
			return types.StakerList{}
		}
		var staker types.Staker
		k.cdc.MustUnmarshal(bz, &staker)
		if int(staker.StakerIndex) != len(stakerList.Stakers) {
			return types.StakerList{}
		}

		stakerList.Stakers = append(stakerList.Stakers, &types.StakerListEntry{StakerAddr: stakerAddr, WithdrawVersion: staker.WithdrawVersion})
	}
	return stakerList
}

// GetStakerList returns the staker list for a native-restaking asset, using cache if available.
// If not cached, fetches from store and updates the cache.
func (k Keeper) GetStakerList(ctx sdk.Context, assetID string, chainID uint64) types.StakerList {
	if chainID == 0 {
		if len(assetID) == 0 {
			return types.StakerList{}
		}
		_, chainID, _ = assetstypes.ParseID(assetID)
		if chainID == 0 {
			return types.StakerList{}
		}
	}
	if sl := k.c.GetNSTStakerList(chainID); len(sl) > 0 {
		return types.StakerList{
			Stakers: sl,
		}
	}
	stakerList := k.getStakerListNoCache(ctx, assetID, chainID)
	// update cache
	if !ctx.IsCheckTx() {
		k.c.SetNSTStakerList(chainID, stakerList.Stakers)
	}
	return stakerList
}

// UpdateNSTValidatorListForStaker handles deposits from the assets module, updating the staker's validator list and balance.
// Emits an event for the deposit and updates the version.
func (k Keeper) UpdateNSTValidatorListForStaker(ctx sdk.Context, assetID, stakerID, validatorPubkey string, amount sdkmath.Int) error {
	// zero value will cause no change, so we do not allow it
	if amount.IsZero() {
		return errors.New("amount should not be zero")
	}
	action := types.Action_ACTION_DEPOSIT
	// if amount is negative, it means withdraw
	if amount.LT(sdkmath.ZeroInt()) {
		amount = amount.Neg()
		action = types.Action_ACTION_WITHDRAW
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

	index, _, err := k.updateStaker(ctx, chainID, 0, amountConverted.Uint64(), 0, stakerAddr, validatorPubkey, action)
	if err != nil {
		return err
	}

	if !ctx.IsCheckTx() {
		versions, _ := k.GetNSTVersions(ctx, chainID)
		// we use index to sync with client about status of stakerInfo.ValidatorPubkeyList
		eValidator := validatorPubkey
		eVersion := versions.Version.Version
		if action == types.Action_ACTION_WITHDRAW {
			eValidator = types.WithdrawValidator
			eVersion = versions.WithdrawVersion
		}
		eventValue := fmt.Sprintf("%s_%d_%s_%s_%d_%d_%d", types.AttributeValueNativeTokenDeposit, index, stakerAddr, eValidator, eVersion, amountConverted.Uint64(), feederID)
		if len(*k.cachedNSTStakersEventValue) > 0 {
			*k.cachedNSTStakersEventValue += types.DelimiterForBase64
		}
		*k.cachedNSTStakersEventValue += eventValue
	}
	return nil
}

// updateStaker updates a staker's info (validator list, balances, index, version) based on the action (deposit, slash, refund, etc).
// Handles new stakers, balance changes, and removal if balance is zero.
// Returns updated index, removal status, balance delta, and error if any.
func (k Keeper) updateStaker(ctx sdk.Context, chainID, roundID, balance, feedVersion uint64, stakerAddr string, validator string, action types.Action) (updatedIndex uint32, balanceDelta sdkmath.Int, err error) {
	if action == types.Action_ACTION_DEPOSIT && validator == "" {
		err = fmt.Errorf("deposit should have one validator, but got empty string")
		return updatedIndex, balanceDelta, err
	}

	stakerInfo := k.GetStakerInfo(ctx, chainID, stakerAddr)
	// make sure stakerInfo is not empty when the action is not DEPOSIT, stakerAddr != "" means both len(stakerInfo.BalanceList) > 0 and len(stakerInfo.ValidatorList) > 0
	if action != types.Action_ACTION_DEPOSIT && (stakerInfo.StakerAddr == "" || len(stakerInfo.BalanceList) == 0 || len(stakerInfo.ValidatorList) == 0) {
		return 0, sdkmath.ZeroInt(), fmt.Errorf("staker or balanceList is not found, stakerAddr is empty: %t, balanceList is empty: %t, action: %s",
			stakerInfo.StakerAddr == "", len(stakerInfo.BalanceList) == 0, action)
	}
	if action == types.Action_ACTION_SLASH_REFUND && feedVersion == 0 {
		return 0, sdkmath.ZeroInt(), fmt.Errorf("feedVersion should not be 0 for slash refund action")
	}

	balanceAtFeedVersion, latestBalance, _ := stakerInfo.GetBalanceAtVersion(feedVersion)
	store := ctx.KVStore(k.storeKey)

	newBalance := &types.BalanceInfo{
		RoundID: roundID,
		// #nosec G115
		Block:   uint64(ctx.BlockHeight()),
		Change:  action,
		Balance: latestBalance,
	}
	staker := &types.Staker{
		ValidatorList: stakerInfo.ValidatorList,
	}
	balanceDelta = sdkmath.NewIntFromUint64(balance)
	if action == types.Action_ACTION_WITHDRAW {
		balanceDelta = balanceDelta.Neg()
	}
	if stakerInfo.StakerAddr == "" {
		// update latest staker index
		latestIndex := k.IncreaseLatestStakerIndex(ctx, chainID)
		staker.StakerIndex = latestIndex
		// set index for stakerAddress index->stakerAddr
		k.SetStakerIndex(ctx, chainID, latestIndex, stakerAddr)
	} else {
		// update staker with new validator
		staker.StakerIndex = stakerInfo.StakerIndex
		// this should always be true
		newBalance.Index = stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].Index
		if action == types.Action_ACTION_SLASH_REFUND {
			balanceDelta = balanceDelta.Sub(sdkmath.NewIntFromUint64(balanceAtFeedVersion))
			newBalance.Balance -= balanceAtFeedVersion
		} else {
			newBalance.RoundID = stakerInfo.BalanceList[len(stakerInfo.BalanceList)-1].RoundID
		}
	}

	updatedIndex = staker.StakerIndex
	// set staker
	if action == types.Action_ACTION_DEPOSIT {
		updatedVersion := k.IncreaseVersionByDeposit(ctx, chainID, balance)
		staker.ValidatorList = append(staker.ValidatorList, &types.ValidatorDeposit{
			ValidatorPubkey: validator,
			Version:         updatedVersion,
			DepositAmount:   balance,
		})
		k.SetStaker(ctx, chainID, stakerAddr, staker)
		if !ctx.IsCheckTx() && stakerInfo.StakerAddr == "" {
			if success := k.c.AddNSTStaker(chainID, types.StakerListEntry{StakerAddr: stakerAddr, WithdrawVersion: staker.WithdrawVersion}, updatedIndex); !success {
				k.refreshCachedStakerList(ctx, chainID)
			}
		}
	} else if action == types.Action_ACTION_WITHDRAW {
		withdrawVersion := k.IncreaseVersionByWithdraw(ctx, chainID)
		staker.WithdrawVersion = withdrawVersion
		k.SetStaker(ctx, chainID, stakerAddr, staker)
		// TODO: use update instead of refresh all
		if !ctx.IsCheckTx() {
			if success := k.c.UpdateWithdrawVersion(chainID, stakerAddr, staker.StakerIndex, withdrawVersion); !success {
				k.refreshCachedStakerList(ctx, chainID)
			}
		}
	}

	// set balanceList
	newBalance.Index++
	if action == types.Action_ACTION_WITHDRAW {
		if balance > newBalance.Balance {
			newBalance.Balance = 0
		} else {
			newBalance.Balance -= balance
		}
	} else {
		newBalance.Balance += balance
	}
	balances := &types.Balances{
		BalanceList: stakerInfo.BalanceList,
	}
	balances.Append(newBalance)
	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	bz := k.cdc.MustMarshal(balances)
	store.Set(keyBalances, bz)
	return updatedIndex, balanceDelta, err
}

func (k Keeper) GetNSTVersionsFromAssetID(ctx sdk.Context, assetID string) (types.NSTVersion, bool) {
	_, chainID, _ := assetstypes.ParseID(strings.ToLower(assetID))
	return k.GetNSTVersions(ctx, chainID)
}

func (k Keeper) GetNSTVersions(ctx sdk.Context, chainID uint64) (types.NSTVersion, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		return types.NSTVersion{}, false
	}
	var v types.NSTVersion
	k.cdc.MustUnmarshal(value, &v)
	return v, true
}

func (k Keeper) UpdateNSTFeedVersion(ctx sdk.Context, chainID uint64) (uint64, uint64, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	var v types.NSTVersion
	if value == nil {
		return 0, 0, false
	}
	k.cdc.MustUnmarshal(value, &v)
	if v.FeedVersion.Version >= v.Version.Version && v.FeedWithdrawVersion >= v.WithdrawVersion {
		return v.FeedVersion.Version, v.FeedWithdrawVersion, false
	}
	v.FeedVersion.Version = v.Version.Version
	v.FeedVersion.DepositAmount = v.Version.DepositAmount
	v.FeedWithdrawVersion = v.WithdrawVersion
	bz := k.cdc.MustMarshal(&v)
	store.Set(key, bz)
	return v.FeedVersion.Version, v.FeedWithdrawVersion, true
}

func (k Keeper) IncreaseVersionByWithdraw(ctx sdk.Context, chainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		return 0
	}
	var v types.NSTVersion
	k.cdc.MustUnmarshal(value, &v)
	v.WithdrawVersion++
	store.Set(key, k.cdc.MustMarshal(&v))
	return v.WithdrawVersion
}

func (k Keeper) IncreaseVersionByDeposit(ctx sdk.Context, chainID, amountAdd uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	var v types.NSTVersion
	if value == nil {
		v = types.NSTVersion{
			Version: &types.VersionDepositAmount{
				Version:       1,
				DepositAmount: amountAdd,
			},
			FeedVersion: &types.VersionDepositAmount{
				Version:       1,
				DepositAmount: amountAdd,
			},
		}
	} else {
		k.cdc.MustUnmarshal(value, &v)
		v.Version.DepositAmount += amountAdd
		v.Version.Version++
	}
	bz := k.cdc.MustMarshal(&v)
	store.Set(key, bz)
	return v.Version.Version
}

// IncreaseVersionByFeed increments the NST version for a given chainID by 1, if the current version is bigger than the feed version.
// returns the new version and the old version {prevVersion, newVersion, err}
func (k Keeper) IncreaseVersionByFeed(ctx sdk.Context, chainID uint64) (uint64, uint64, uint64, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	var v types.NSTVersion
	if value == nil {
		// this should not happen when the workflow is correct (feeding price can only happen after deposit)
		return 0, 0, 0, errors.New("version not found")
	}
	k.cdc.MustUnmarshal(value, &v)
	prevVersion := v.FeedVersion.Version
	if v.Version.Version > v.FeedVersion.Version || v.WithdrawVersion > v.FeedWithdrawVersion {
		v.FeedVersion.Version = v.Version.Version
		v.FeedVersion.DepositAmount = v.Version.DepositAmount
		v.FeedWithdrawVersion = v.WithdrawVersion
		bz := k.cdc.MustMarshal(&v)
		store.Set(key, bz)
	}
	return prevVersion, v.FeedVersion.Version, v.WithdrawVersion, nil
}

// SetStakerIndex sets the mapping from staker index to address for a chainID.
func (k Keeper) SetStakerIndex(ctx sdk.Context, chainID uint64, index uint32, stakerAddr string) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTStakerAddrKey(chainID, index)
	store.Set(key, []byte(stakerAddr))
}

// GetLatestStakerIndex retrieves the latest staker index for a chainID.
func (k Keeper) GetLatestStakerIndex(ctx sdk.Context, chainID uint64) (uint32, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTLatestStakerIndexKey(chainID)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	idx, err := types.BytesToUint32(bz)
	if err != nil {
		return 0, false
	}
	return idx, true
}

// IncreaseLatestStakerIndex increments and returns the latest staker index for a chainID.
func (k Keeper) IncreaseLatestStakerIndex(ctx sdk.Context, chainID uint64) uint32 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTLatestStakerIndexKey(chainID)
	bz := store.Get(key)
	if bz == nil {
		store.Set(key, types.Uint32Bytes(0))
		return 0
	}
	latestStakerIndex, err := types.BytesToUint32(bz)
	if err != nil {
		return 0
	}
	latestStakerIndex++
	store.Set(key, types.Uint32Bytes(latestStakerIndex))
	return latestStakerIndex
}

// SetStaker stores the staker struct for a given chainID and address.
func (k Keeper) SetStaker(ctx sdk.Context, chainID uint64, stakerAddr string, staker *types.Staker) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(staker)
	keyStaker := types.NSTStakerKey(chainID, stakerAddr)
	store.Set(keyStaker, bz)
}

// convertDecimal converts an amount between asset and oracle module decimals, depending on direction.
// Handles precision and rounding errors.
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

func (k Keeper) refreshCachedStakerList(ctx sdk.Context, chainID uint64) {
	sl := k.getStakerListNoCache(ctx, "", chainID)
	if len(sl.Stakers) > 0 {
		k.c.SetNSTStakerList(chainID, sl.Stakers)
	}
}

// UpdateNSTBalanceChange processes post-aggregation NST (Native Staking Token) balance changes at the end of a block.
//
// This function is called in EndBlock (not as a transaction), so errors do not revert the block but are returned for logging/monitoring.
// It is responsible for synchronizing the on-chain state with the results of off-chain aggregation/settlement.
//
// Steps performed:
// 1. Unmarshal the provided rawData into a RawDataNST struct containing all balance changes for this round.
// 2. Validate that the version in the balance changes matches the current on-chain version for the asset/chain.
// 3. Retrieve the current staker list for the asset. If the cache is empty, fill it from the store.
// 4. For each balance change:
//   - Update the staker's balance and state (using updateStaker).
//   - If the staker is removed (balance zero), record the index for later removal.
//   - If the balance delta is nonzero, update the delegation module accordingly.
//
// 5. Remove all stakers that were marked for removal, updating the staker list and indexes.
// 6. Commit all changes to the context cache.
// 7. Increment the NST version for the asset/chain.
// 8. Emit an event with the new root hash, version, and feederID for downstream consumers.
//
// This function ensures that the staker list, balances, and delegation state are all kept in sync after aggregation,
// and that the system is ready for the next round of staking/aggregation.
//
// Errors are returned for monitoring but do not revert the block.
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
	versions, found := k.GetNSTVersions(ctx, chainID)
	if !found || balanceChanges.Version != versions.FeedVersion.Version || balanceChanges.WithdrawVersion != versions.FeedWithdrawVersion {
		return fmt.Errorf("version not match, expected feedVersion:%d, feedWithdrawVersion:%d,  got f:%d, fw:%d, assetID:%s", versions.FeedVersion.Version, versions.FeedWithdrawVersion, balanceChanges.Version, balanceChanges.WithdrawVersion, assetID)
	}

	sl := k.GetStakerList(ctx, "", chainID)
	if len(sl.Stakers) == 0 {
		return errors.New("staker list is empty")
	}

	cc, writeCache := ctx.CacheContext()
	for _, changeKV := range balanceChanges.NstBalanceChanges {
		if int(changeKV.StakerIndex) >= len(sl.Stakers) {
			return fmt.Errorf("staker index %d out of range for staker list length %d", changeKV.StakerIndex, len(sl.Stakers))
		}

		staker := sl.Stakers[changeKV.StakerIndex]
		if balanceChanges.WithdrawVersion < staker.WithdrawVersion {
			// skip balance change update for stakers who had executed withdraw during price feeding
			continue
		}
		stakerAddr := staker.StakerAddr
		_, balanceDelta, err := k.updateStaker(cc, chainID, roundID, changeKV.Balance, balanceChanges.Version, stakerAddr, "", types.Action_ACTION_SLASH_REFUND)
		if err != nil {
			return err
		}

		if balanceDelta.IsZero() {
			continue
		}

		balanceDeltaConverted, err := k.convertDecimal(cc, assetID, balanceDelta, feederID, false)
		if err != nil {
			return fmt.Errorf("failed to convert balance delta: %w", err)
		}
		if err := k.delegationKeeper.UpdateNSTBalance(cc, getStakerID(stakerAddr, chainID), assetID, balanceDeltaConverted); err != nil {
			return err
		}
	}

	_, newVersion, withdrawVersion, err := k.IncreaseVersionByFeed(cc, chainID)
	if err != nil {
		return err
	}

	writeCache()
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNSTBalanceUpdate, types.AttributeValueTrue),
		sdk.NewAttribute(types.AttributeKeyNSTBalanceChange, fmt.Sprintf("%s|%d|%d|%d", base64.StdEncoding.EncodeToString(rootHash), newVersion, feederID, withdrawVersion)),
	))
	return nil
}

// getStakerID returns a unique string identifier for a staker on a given chainID, used for cross-module referencing.
func getStakerID(stakerAddr string, chainID uint64) string {
	return strings.Join([]string{strings.ToLower(stakerAddr), hexutil.EncodeUint64(chainID)}, utils.DelimiterForID)
}
