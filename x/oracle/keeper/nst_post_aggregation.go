package keeper

import (
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

// SetStakerInfos set stakerInfos for the specific assetID
func (k Keeper) SetStakerInfos(ctx sdk.Context, assetID string, stakerInfos []*types.StakerInfo) {
	store := ctx.KVStore(k.storeKey)
	for _, stakerInfo := range stakerInfos {
		bz := k.cdc.MustMarshal(stakerInfo)
		store.Set(types.NativeTokenStakerKey(assetID, stakerInfo.StakerAddr), bz)
	}
}

// GetStakerInfo returns details about staker for native-restaking under asset of assetID
func (k Keeper) GetStakerInfo(ctx sdk.Context, assetID, stakerAddr string) types.StakerInfo {
	stakerAddr = strings.ToLower(stakerAddr)
	store := ctx.KVStore(k.storeKey)
	stakerInfo := types.StakerInfo{}
	value := store.Get(types.NativeTokenStakerKey(assetID, stakerAddr))
	if value == nil {
		return stakerInfo
	}
	k.cdc.MustUnmarshal(value, &stakerInfo)
	return stakerInfo
}

// GetStakerInfos returns all stakers information
func (k Keeper) GetStakerInfos(ctx sdk.Context, req *types.QueryStakerInfosRequest) (*types.QueryStakerInfosResponse, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.NativeTokenStakerKeyPrefix(req.AssetId))
	retStakerInfos := make([]*types.StakerInfo, 0)
	if req.Pagination != nil && req.Pagination.Limit > types.MaxPageLimit {
		return nil, status.Errorf(codes.InvalidArgument, "pagination limit %d exceeds maximum allowed %d", req.Pagination.Limit, types.MaxPageLimit)
	}
	resPage, err := query.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		sInfo := types.StakerInfo{}
		k.cdc.MustUnmarshal(value, &sInfo)
		// keep only the latest effective-balance
		if len(sInfo.BalanceList) > 0 {
			sInfo.BalanceList = sInfo.BalanceList[len(sInfo.BalanceList)-1:]
		}
		retStakerInfos = append(retStakerInfos, &sInfo)
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}
	return &types.QueryStakerInfosResponse{
		StakerInfos: retStakerInfos,
		Pagination:  resPage,
	}, nil
}

// GetAllStakerInfosAssets returns all stakerInfos combined with assetIDs they belong to, used for genesisstate exporting
func (k Keeper) GetAllStakerInfosAssets(ctx sdk.Context) (ret []types.StakerInfosAssets) {
	store := ctx.KVStore(k.storeKey)
	store = prefix.NewStore(store, types.NativeTokenStakerKeyPrefix(""))
	// set assetID as "" to iterate all value with different assetIDs
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	ret = make([]types.StakerInfosAssets, 0)
	l := 0
	for ; iterator.Valid(); iterator.Next() {
		assetID, _ := types.ParseNativeTokenStakerKey(iterator.Key())
		if l == 0 || ret[l-1].AssetId != assetID {
			version := k.GetNSTVersion(ctx, assetID)
			ret = append(ret, types.StakerInfosAssets{
				NstVersion:  version,
				AssetId:     assetID,
				StakerInfos: make([]*types.StakerInfo, 0),
			})
			l++
		}
		v := &types.StakerInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), v)
		ret[l-1].StakerInfos = append(ret[l-1].StakerInfos, v)
	}
	return ret
}

// SetStakerList set staker list for assetID, this is mainly used for genesis init
func (k Keeper) SetStakerList(ctx sdk.Context, assetID string, sl *types.StakerList) {
	if sl == nil {
		return
	}
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(sl)
	store.Set(types.NativeTokenStakerListKey(assetID), bz)
}

// GetStakerList return stakerList for native-restaking asset of assetID
func (k Keeper) GetStakerList(ctx sdk.Context, assetID string) types.StakerList {
	store := ctx.KVStore(k.storeKey)
	value := store.Get(types.NativeTokenStakerListKey(assetID))
	if value == nil {
		return types.StakerList{}
	}
	stakerList := &types.StakerList{}
	k.cdc.MustUnmarshal(value, stakerList)
	return *stakerList
}

// GetAllStakerListAssets return stakerList combined with assetIDs they belong to, used for genesisstate exporting
func (k Keeper) GetAllStakerListAssets(ctx sdk.Context) (ret []types.StakerListAssets) {
	// set assetID with "" to iterate all stakerList with every assetIDs
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.NativeTokenStakerListKey(""))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	ret = make([]types.StakerListAssets, 0)
	for ; iterator.Valid(); iterator.Next() {
		v := &types.StakerList{}
		k.cdc.MustUnmarshal(iterator.Value(), v)
		version := k.GetNSTVersion(ctx, string(iterator.Key()))
		ret = append(ret, types.StakerListAssets{
			AssetId:    string(iterator.Key()),
			StakerList: v,
			NstVersion: version,
		})
	}
	return ret
}

func (k Keeper) UpdateNSTValidatorListForStaker(ctx sdk.Context, assetID, stakerAddr, validatorPubkey string, amount sdkmath.Int) error {
	stakerAddr = strings.ToLower(stakerAddr)
	_, decimalInt, err := k.getDecimal(ctx, assetID)
	if err != nil {
		return err
	}

	_, chainID, _ := assetstypes.ParseID(assetID)
	feederID, ok := k.GetNSTFeederIDFromClientChainID(chainID)
	if !ok {
		return errors.New("failed to get corresponding feederID from clientChainID")
	}

	amountInt64 := amount.Quo(decimalInt).Int64()
	var withdraw bool
	if amountInt64 < 0 {
		withdraw = true
		amountInt64 = -amountInt64
	}
	// emit an event to tell that a staker's validator list has changed
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNativeTokenUpdate, types.AttributeValueNativeTokenUpdate),
	))
	store := ctx.KVStore(k.storeKey)
	key := types.NativeTokenStakerKey(assetID, stakerAddr)
	stakerInfo := &types.StakerInfo{}
	if value := store.Get(key); value == nil {
		// create a new item for this staker
		stakerInfo = types.NewStakerInfo(stakerAddr, validatorPubkey)
	} else {
		k.cdc.MustUnmarshal(value, stakerInfo)
		if !withdraw {
			// deposit add a new validator into staker's validatorList
			// one validator can only deposit once before it completed withdraw which remove its pubkey form this list. So there's no need to check duplication
			stakerInfo.ValidatorPubkeyList = append(stakerInfo.ValidatorPubkeyList, validatorPubkey)
		}
	}

	newBalance := types.BalanceInfo{}

	if latestIndex := len(stakerInfo.BalanceList) - 1; latestIndex >= 0 {
		newBalance = *(stakerInfo.BalanceList[latestIndex])
		newBalance.Index++
	}
	// #nosec G115
	newBalance.Block = uint64(ctx.BlockHeight())

	if !withdraw {
		newBalance.Change = types.Action_ACTION_DEPOSIT
	} else {
		// TODO: check if this validator has withdraw all its asset and then we can move it out from the staker's validatorList
		// currently when withdraw happened we assume this validator has left the staker's validatorList (deposit/withdraw all of that validator's staking ETH)
		newBalance.Change = types.Action_ACTION_WITHDRAW
		for i, vPubkey := range stakerInfo.ValidatorPubkeyList {
			if vPubkey == validatorPubkey {
				// TODO: len(stkaerInfo.ValidatorPubkeyList)==0 should equal to newBalance.Balance<=0
				stakerInfo.ValidatorPubkeyList = append(stakerInfo.ValidatorPubkeyList[:i], stakerInfo.ValidatorPubkeyList[i+1:]...)
				break
			}
		}
	}

	if withdraw {
		if newBalance.Balance < uint64(amountInt64) {
			return errors.New("withdraw more than deposit")
		}
		newBalance.Balance -= uint64(amountInt64)
	} else {
		newBalance.Balance += uint64(amountInt64)
	}

	keyStakerList := types.NativeTokenStakerListKey(assetID)
	valueStakerList := store.Get(keyStakerList)
	var stakerList types.StakerList
	stakerList.StakerAddrs = make([]string, 0, 1)
	if valueStakerList != nil {
		k.cdc.MustUnmarshal(valueStakerList, &stakerList)
	}
	exists := false
	for idx, stakerExists := range stakerList.StakerAddrs {
		// this should noly happen when do withdraw
		if stakerExists == stakerAddr {
			if newBalance.Balance <= 0 {
				stakerList.StakerAddrs = append(stakerList.StakerAddrs[:idx], stakerList.StakerAddrs[idx+1:]...)
				valueStakerList = k.cdc.MustMarshal(&stakerList)
				store.Set(keyStakerList, valueStakerList)
			}
			exists = true
			// #nosec G115
			stakerInfo.StakerIndex = uint32(idx)
			break
		}
	}

	if !exists {
		if withdraw {
			return errors.New("remove unexist validator")
		}
		stakerList.StakerAddrs = append(stakerList.StakerAddrs, stakerAddr)
		valueStakerList = k.cdc.MustMarshal(&stakerList)
		store.Set(keyStakerList, valueStakerList)
		// #nosec G115
		stakerInfo.StakerIndex = uint32(len(stakerList.StakerAddrs) - 1)
	}

	if newBalance.Balance <= 0 {
		store.Delete(key)
	} else {
		stakerInfo.Append(&newBalance)
		bz := k.cdc.MustMarshal(stakerInfo)
		store.Set(key, bz)
	}

	// valid veriosn start from 1
	version := k.IncreaseNSTVersion(ctx, assetID)
	// we use index to sync with client about status of stakerInfo.ValidatorPubkeyList
	eventValue := fmt.Sprintf("%d_%s_%d_%d_%d", stakerInfo.StakerIndex, validatorPubkey, version, amountInt64, feederID)
	if newBalance.Change == types.Action_ACTION_DEPOSIT {
		eventValue = fmt.Sprintf("%s_%s", types.AttributeValueNativeTokenDeposit, eventValue)
	} else {
		eventValue = fmt.Sprintf("%s_%s", types.AttributeValueNativeTokenWithdraw, eventValue)
	}
	// emit an event to tell the details that a new valdiator added/or a validator is removed for the staker
	// deposit_stakerID_validatorKey
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNativeTokenChange, eventValue),
	))

	return nil
}

// IncreaseNSTVersion increases the version of native token for assetID
func (k Keeper) IncreaseNSTVersion(ctx sdk.Context, assetID string) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NativeTokenVersionKey(assetID)
	value := store.Get(key)
	if value == nil {
		// set the first index of version to 1
		store.Set(key, sdk.Uint64ToBigEndian(1))
		return 1
	}
	version := sdk.BigEndianToUint64(value) + 1
	store.Set(key, sdk.Uint64ToBigEndian(version))
	// #nosec G115
	// TODO: use uint64 for version, the price-feeder may need corresponding change
	return int64(version)
}

// IncreaseNSTVersion increases the version of native token for assetID
func (k Keeper) SetNSTVersion(ctx sdk.Context, assetID string, version int64) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NativeTokenVersionKey(assetID)
	// #nosec version is not negative
	store.Set(key, sdk.Uint64ToBigEndian(uint64(version)))
	return version
}

func (k Keeper) GetNSTVersion(ctx sdk.Context, assetID string) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.NativeTokenVersionKey(assetID)
	value := store.Get(key)
	if value == nil {
		return 0
	}
	// #nosec G115
	return int64(sdk.BigEndianToUint64(value))
}

func (k Keeper) getDecimal(ctx sdk.Context, assetID string) (int, sdkmath.Int, error) {
	decimalMap, err := k.assetsKeeper.GetAssetsDecimal(ctx, map[string]any{assetID: nil})
	if err != nil {
		return 0, sdkmath.ZeroInt(), err
	}
	decimal := decimalMap[assetID]
	return int(decimal), sdkmath.NewIntWithDecimal(1, int(decimal)), nil
}

// TODO use []byte and assetstypes.GetStakerIDAndAssetID for stakerAddr representation
func getStakerID(stakerAddr string, chainID uint64) string {
	return strings.Join([]string{strings.ToLower(stakerAddr), hexutil.EncodeUint64(chainID)}, utils.DelimiterForID)
}

// UpdateNSTBalanceChange serves the post handling for nst balance change
func UpdateNSTBalanceChange(ctx sdk.Context, rawData []byte, feederID, roundID uint64, kInf common.KeeperOracle) error {
	balanceChanges := &types.RawDataNST{}
	kInf.MustUnmarshal(rawData, balanceChanges)
	k, ok := kInf.(*Keeper)
	if !ok {
		return errors.New("input keeper interface type error")
	}
	assetID := k.GetParamsFromCache().GetAssetIDForNSTFromFeederID(feederID)
	// TODO(leonz): use uint64 for version state
	// #nosec G115
	v := uint64(k.GetNSTVersion(ctx, assetID))
	if balanceChanges.Version != v {
		return fmt.Errorf("version not match, expected %d, got %d", v, balanceChanges.Version)
	}
	_, chainID, _ := assetstypes.ParseID(assetID)
	sl := k.GetStakerList(ctx, assetID)
	if len(sl.StakerAddrs) == 0 {
		return errors.New("staker list is empty")
	}

	store := ctx.KVStore(k.storeKey)

	for _, changeKV := range balanceChanges.NstBalanceChanges {
		stakerAddr := sl.StakerAddrs[changeKV.StakerIndex]
		key := types.NativeTokenStakerKey(assetID, stakerAddr)
		value := store.Get(key)
		if value == nil {
			return errors.New("stakerInfo does not exist")
		}
		stakerInfo := &types.StakerInfo{}
		k.cdc.MustUnmarshal(value, stakerInfo)
		newBalance := types.BalanceInfo{}
		if length := len(stakerInfo.BalanceList); length > 0 {
			newBalance = *(stakerInfo.BalanceList[length-1])
		}
		// #nosec G115 - block height will never be negative
		newBalance.Block = uint64(ctx.BlockHeight())
		// we set index as a global reference used through all rounds
		newBalance.Index++
		newBalance.Change = types.Action_ACTION_SLASH_REFUND
		newBalance.RoundID = roundID
		balance := changeKV.Balance

		if delta := balance - newBalance.Balance; delta != 0 {
			decimal, _, err := k.getDecimal(ctx, assetID)
			if err != nil {
				return err
			}
			amountChange := sdkmath.NewIntFromUint64(delta)
			amountChange = amountChange.Mul(sdkmath.NewIntWithDecimal(1, decimal))

			// if err := k.delegationKeeper.UpdateNSTBalance(ctx, getStakerID(stakerAddr, chainID), assetID, sdkmath.NewIntWithDecimal(delta, decimal)); err != nil {
			if err := k.delegationKeeper.UpdateNSTBalance(ctx, getStakerID(stakerAddr, chainID), assetID, amountChange); err != nil {
				return err
			}
			newBalance.Balance = balance
		}
		stakerInfo.Append(&newBalance)
		bz := k.cdc.MustMarshal(stakerInfo)
		store.Set(key, bz)
	}
	return nil
}
