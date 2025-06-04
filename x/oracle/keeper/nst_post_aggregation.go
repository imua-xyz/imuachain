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
// SetStakerInfosForAsset sets the staker information and balances for a given asset (chainID),
// and updates the latest staker index and version. Used during aggregation or state sync.
func (k Keeper) SetStakerInfosForAsset(ctx sdk.Context, chainID uint64, stakerInfos []*types.StakerInfo, version uint64) {
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
			ValidatorList: stakerInfo.ValidatorPubkeyList,
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
	store.Set(keyVersion, types.Uint64Bytes(version))
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
		StakerAddr:          stakerAddr,
		StakerIndex:         staker.StakerIndex,
		ValidatorPubkeyList: staker.ValidatorList,
		BalanceList:         balances.BalanceList,
	}
}

// GetStakerInfos returns all stakers' information for a given asset, with optional pagination and balance history.
// Used for queries and state export.
func (k Keeper) GetStakerInfos(ctx sdk.Context, req *types.QueryStakerInfosRequest) (*types.QueryStakerInfosResponse, error) {
	if req.Pagination != nil && req.Pagination.Limit > types.MaxPageLimit {
		return nil, status.Errorf(codes.InvalidArgument, "pagination limit %d exceeds maximum allowed %d", req.Pagination.Limit, types.MaxPageLimit)
	}

	_, chainID, _ := assetstypes.ParseID(strings.ToLower(req.AssetId))
	store := ctx.KVStore(k.storeKey)
	// retrieve version
	bz := store.Get(types.NSTVersionKey(chainID))
	version := uint64(0)
	var err error
	if bz != nil {
		version, err = types.BytesToUint64(bz)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse version: %v", err)
		}
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
		Version:     version,
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
		StakerAddr:          string(key),
		StakerIndex:         staker.StakerIndex,
		ValidatorPubkeyList: staker.ValidatorList,
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
		iteratorStakers := sdk.KVStorePrefixIterator(store, types.NSTStakerKeyChainIDPrefix(chainID))
		stakerInfos := make([]*types.StakerInfo, 0)
		for ; iteratorStakers.Valid(); iteratorStakers.Next() {
			stakerInfo, err := k.getStakerInfos(store, types.NSTBalancesKeyChainIDPrefix(chainID), iteratorStakers.Key(), iteratorStakers.Value(), true)
			if err != nil {
				return nil, err
			}
			stakerInfos = append(stakerInfos, stakerInfo)
		}
		version, err := types.BytesToUint64(iterator.Value())
		if err != nil {
			return nil, fmt.Errorf("failed to parse version from value: %w", err)
		}
		ret = append(ret, types.StakerInfosAssets{
			// #nosec G115
			NstVersion:  version,
			ChainId:     chainID,
			StakerInfos: stakerInfos,
		})
	}

	return ret, nil
}

// getStakerListNoCache retrieves the list of staker addresses for an asset (chainID) directly from the store (no cache).
func (k Keeper) getStakerListNoCache(ctx sdk.Context, assetID string) types.StakerList {
	_, chainID, _ := assetstypes.ParseID(assetID)
	store := ctx.KVStore(k.storeKey)
	keyStakerAddrPrefix := types.NSTStakerAddrKeyChainIDPrefix(chainID)
	store = prefix.NewStore(store, keyStakerAddrPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
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

// GetStakerList returns the staker list for a native-restaking asset, using cache if available.
// If not cached, fetches from store and updates the cache.
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

// UpdateNSTValidatorListForStaker handles deposits from the assets module, updating the staker's validator list and balance.
// Emits an event for the deposit and updates the version.
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

	index, _, _, err := k.updateStaker(ctx, chainID, 0, amountConverted.Uint64(), stakerAddr, validatorPubkey, types.Action_ACTION_DEPOSIT)
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

// GetNSTVersionFromAssetID retrieves the NST version for a given assetID (parsing chainID from assetID).
func (k Keeper) GetNSTVersionFromAssetID(ctx sdk.Context, assetID string) uint64 {
	_, chainID, _ := assetstypes.ParseID(strings.ToLower(assetID))
	return k.GetNSTVersion(ctx, chainID)
}

// GetNSTVersion gets the NST version for a given chainID from the store.
func (k Keeper) GetNSTVersion(ctx sdk.Context, chainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		return 0
	}
	v, err := types.BytesToUint64(value)
	if err != nil {
		return 0
	}
	return v
}

// removeStaker removes a staker and their balances from the store when their balance becomes zero.
// Returns the removed staker's index and a bool indicating if removal occurred.
func (k Keeper) removeStaker(ctx sdk.Context, chainID uint64, stakerAddr string) (uint32, bool) {
	_, found := k.GetLatestStakerIndex(ctx, chainID)
	if !found {
		return 0, false
	}

	store := ctx.KVStore(k.storeKey)
	keyStaker := types.NSTStakerKey(chainID, stakerAddr)
	staker := types.Staker{}
	bz := store.Get(keyStaker)
	if bz == nil {
		return 0, false
	}

	k.cdc.MustUnmarshal(bz, &staker)
	removedIndex := staker.StakerIndex
	// remove staker basic info
	store.Delete(keyStaker)

	// remove balanceList
	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	store.Delete(keyBalances)

	k.IncreaseVersion(ctx, chainID)
	return removedIndex, true
}

// updateStaker updates a staker's info (validator list, balances, index, version) based on the action (deposit, slash, refund, etc).
// Handles new stakers, balance changes, and removal if balance is zero.
// Returns updated index, removal status, balance delta, and error if any.
func (k Keeper) updateStaker(ctx sdk.Context, chainID, roundID, balance uint64, stakerAddr string, validator string, action types.Action) (updatedIndex uint32, removed bool, balanceDelta sdkmath.Int, err error) {
	store := ctx.KVStore(k.storeKey)
	if action == types.Action_ACTION_SLASH_REFUND && balance == 0 {
		// this is a special case, we need to remove the staker from the list
		// and update the index of the last staker
		keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
		bz := store.Get(keyBalances)
		if bz == nil {
			balanceDelta = sdkmath.ZeroInt()
			// nothing to remove, apply no balance change
			return updatedIndex, removed, balanceDelta, err
		}

		balances := &types.Balances{}
		k.cdc.MustUnmarshal(bz, balances)
		if len(balances.BalanceList) > 0 {
			balanceDelta = sdkmath.NewIntFromUint64(balances.BalanceList[len(balances.BalanceList)-1].Balance)
		}
		updatedIndex, removed = k.removeStaker(ctx, chainID, stakerAddr)
		return updatedIndex, removed, balanceDelta, err
	}

	if action == types.Action_ACTION_DEPOSIT && validator == "" {
		err = fmt.Errorf("deposit should have one validator, but got empty string")
		return updatedIndex, removed, balanceDelta, err
	}

	stakerInfo := k.GetStakerInfo(ctx, chainID, stakerAddr)
	if action != types.Action_ACTION_DEPOSIT && (stakerInfo.StakerAddr == "" || len(stakerInfo.BalanceList) == 0) {
		err = fmt.Errorf("staker or balanceList is not found, stakerAddr is empty: %t, balanceList is empty: %t, action: %s",
			stakerInfo.StakerAddr == "", len(stakerInfo.BalanceList) == 0, action)
		return updatedIndex, removed, balanceDelta, err
	}

	newBalance := &types.BalanceInfo{
		RoundID: roundID,
		// #nosec G115
		Block:  uint64(ctx.BlockHeight()),
		Change: action,
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

	updatedIndex = staker.StakerIndex

	// set staker
	if action == types.Action_ACTION_DEPOSIT {
		staker.ValidatorList = append(staker.ValidatorList, validator)
	}
	k.SetStaker(ctx, chainID, stakerAddr, staker)

	// set balanceList
	newBalance.Index++
	newBalance.Balance += balance
	balances := &types.Balances{
		BalanceList: stakerInfo.BalanceList,
	}
	balances.Append(newBalance)
	keyBalances := types.NSTBalancesKey(chainID, stakerAddr)
	bz := k.cdc.MustMarshal(balances)
	store.Set(keyBalances, bz)

	// increase version
	k.IncreaseVersion(ctx, chainID)
	return updatedIndex, removed, balanceDelta, err
}

// IncreaseVersion increments the NST version for a chainID, used to track state changes.
func (k Keeper) IncreaseVersion(ctx sdk.Context, chainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NSTVersionKey(chainID)
	value := store.Get(key)
	if value == nil {
		store.Set(key, types.Uint64Bytes(1))
		return 1
	}
	version, err := types.BytesToUint64(value)
	if err != nil {
		return 0
	}
	version++
	store.Set(key, types.Uint64Bytes(version))
	return version
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

// removeStakerIndexes rotates the staker list after removals, updating indexes and cache as needed.
func (k Keeper) removeStakerIndexes(ctx sdk.Context, chainID uint64, removedIndexes []uint32) error {
	updatedStakers, err := k.c.RotateStakerList(chainID, removedIndexes)
	if err != nil {
		// TODO: do we refresh the cache here ?
		return fmt.Errorf("failed to rotate stakerList")
	}
	l := len(updatedStakers)
	if l > 0 {
		store := ctx.KVStore(k.storeKey)
		keyLatestStakerIndex := types.NSTLatestStakerIndexKey(chainID)
		latestBz := store.Get(keyLatestStakerIndex)
		if latestBz == nil {
			return fmt.Errorf("latest staker index not found for chainID %d", chainID)
		}
		// latestStakerIndex, err := types.BytesToUint32(store.Get(keyLatestStakerIndex))
		latestStakerIndex, err := types.BytesToUint32(latestBz)
		if err != nil {
			return fmt.Errorf("failed to parse latest staker index: %w", err)
		}
		if l > int(latestStakerIndex) {
			store.Delete(keyLatestStakerIndex)
		} else {
			// #nosec G115
			latestStakerIndex -= uint32(l)
			store.Set(keyLatestStakerIndex, types.Uint32Bytes(latestStakerIndex))
		}
		for index, stakerAddr := range updatedStakers {
			keyStaker := types.NSTStakerKey(chainID, stakerAddr)
			staker := types.Staker{}
			bz := store.Get(keyStaker)
			if bz == nil {
				return fmt.Errorf("staker %s not found when rotate index for removed stakers", stakerAddr)
			}
			k.cdc.MustUnmarshal(bz, &staker)
			staker.StakerIndex = index
			store.Set(keyStaker, k.cdc.MustMarshal(&staker))
			keyStakerAddr := types.NSTStakerAddrKey(chainID, index)
			store.Set(keyStakerAddr, []byte(stakerAddr))
		}
	}
	return nil
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
	v := k.GetNSTVersion(ctx, chainID)
	if balanceChanges.Version != v {
		return fmt.Errorf("version not match, expected %d, got %d, assetID:%s", v, balanceChanges.Version, assetID)
	}

	sl := k.GetStakerList(ctx, assetID)
	if len(sl.StakerAddrs) == 0 {
		return errors.New("staker list is empty")
	}

	// fill staker list cache
	if len(k.c.GetNSTStakerList(chainID)) == 0 {
		//		_ = k.GetStakerList(ctx, assetID)
		sl := k.getStakerListNoCache(ctx, assetID)
		if len(sl.StakerAddrs) > 0 {
			k.c.SetNSTStakerList(chainID, sl.StakerAddrs)
		}
	}
	cc, writeCache := ctx.CacheContext()
	removedIndexes := make([]uint32, 0)
	for _, changeKV := range balanceChanges.NstBalanceChanges {
		stakerAddr := sl.StakerAddrs[changeKV.StakerIndex]
		index, removed, balanceDelta, err := k.updateStaker(cc, chainID, roundID, changeKV.Balance, stakerAddr, "", types.Action_ACTION_SLASH_REFUND)
		if err != nil {
			return err
		}

		if removed {
			removedIndexes = append(removedIndexes, index)
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

	if err := k.removeStakerIndexes(cc, chainID, removedIndexes); err != nil {
		return err
	}

	// update all removed stakers' index
	writeCache()
	version := k.IncreaseVersion(ctx, chainID)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNSTBalanceUpdate, types.AttributeValueTrue),
		sdk.NewAttribute(types.AttributeKeyNSTBalanceChange, fmt.Sprintf("%s|%d|%d", base64.StdEncoding.EncodeToString(rootHash), version, feederID)),
	))
	return nil
}

// getStakerID returns a unique string identifier for a staker on a given chainID, used for cross-module referencing.
func getStakerID(stakerAddr string, chainID uint64) string {
	return strings.Join([]string{strings.ToLower(stakerAddr), hexutil.EncodeUint64(chainID)}, utils.DelimiterForID)
}
