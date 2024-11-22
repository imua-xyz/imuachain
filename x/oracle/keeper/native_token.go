package keeper

import (
	"errors"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	utils "github.com/ExocoreNetwork/exocore/utils"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// deposit: update staker's totalDeposit
// withdoraw: update staker's totalDeposit
// delegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// undelegate: update operator's price, operator's totalAmount, operator's totalShare, staker's share
// msg(refund or slash on beaconChain): update staker's price, operator's price

type NSTETHAssetID string

const (
	// TODO: we currently support NSTETH only which has capped effective balance for one validator
	// TODO: this is a bad practice, and for Lz, they have different version of endpoint with different chainID
	// Do the validation before invoke oracle related functions instead of check these hard code ids here.
	NSTETHAssetIDMainnet  NSTETHAssetID = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x7595"
	NSTETHAssetIDLocalnet NSTETHAssetID = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"
	NSTETHAssetIDHolesky  NSTETHAssetID = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x9d19"
	NSTETHAssetIDSepolia  NSTETHAssetID = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x9ce1"
)

var (
	limitedChangeNST = map[NSTETHAssetID]bool{
		NSTETHAssetIDMainnet:  true,
		NSTETHAssetIDLocalnet: true,
		NSTETHAssetIDHolesky:  true,
		NSTETHAssetIDSepolia:  true,
	}

	maxEffectiveBalances = map[NSTETHAssetID]int{
		NSTETHAssetIDMainnet:  32,
		NSTETHAssetIDLocalnet: 32,
		NSTETHAssetIDHolesky:  32,
		NSTETHAssetIDSepolia:  32,
	}
)

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
	store := ctx.KVStore(k.storeKey)
	stakerInfo := types.StakerInfo{}
	value := store.Get(types.NativeTokenStakerKey(assetID, stakerAddr))
	if value == nil {
		return stakerInfo
	}
	k.cdc.MustUnmarshal(value, &stakerInfo)
	return stakerInfo
}

// TODO: pagination
// GetStakerInfos returns all stakers information
func (k Keeper) GetStakerInfos(ctx sdk.Context, assetID string) (ret []*types.StakerInfo) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.NativeTokenStakerKeyPrefix(assetID))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		sInfo := types.StakerInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &sInfo)
		// keep only the latest effective-balance
		sInfo.BalanceList = sInfo.BalanceList[:len(sInfo.BalanceList)-1]
		// this is mainly used by price feeder, so we remove the stakerAddr to reduce the size of return value
		sInfo.StakerAddr = ""
		ret = append(ret, &sInfo)
	}
	return ret
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
			ret = append(ret, types.StakerInfosAssets{
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
		ret = append(ret, types.StakerListAssets{
			AssetId:    string(iterator.Key()),
			StakerList: v,
		})
	}
	return ret
}

func (k Keeper) UpdateNSTValidatorListForStaker(ctx sdk.Context, assetID, stakerAddr, validatorPubkey string, amount sdkmath.Int) error {
	if !IsLimitedChangeNST(assetID) {
		return types.ErrNSTAssetNotSupported
	}
	_, decimalInt, err := k.getDecimal(ctx, assetID)
	if err != nil {
		return err
	}
	amountInt64 := amount.Quo(decimalInt).Int64()
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
		if amountInt64 > 0 {
			// deopsit add a new validator into staker's validatorList
			// one validator can only deposit once before it completed withdraw which remove its pubkey form this list. So there's no need to check duplication
			stakerInfo.ValidatorPubkeyList = append(stakerInfo.ValidatorPubkeyList, validatorPubkey)
		}
	}

	newBalance := types.BalanceInfo{}

	if latestIndex := len(stakerInfo.BalanceList) - 1; latestIndex >= 0 {
		newBalance = *(stakerInfo.BalanceList[latestIndex])
		newBalance.Index++
	}
	newBalance.Block = uint64(ctx.BlockHeight())
	if amountInt64 > 0 {
		newBalance.Change = types.Action_ACTION_DEPOSIT
	} else {
		// TODO: check if this validator has withdraw all its asset and then we can move it out from the staker's validatorList
		// currently when withdraw happened we assume this validator has left the staker's validatorList (deposit/withdraw all of that validator's staking ETH(<=32))
		newBalance.Change = types.Action_ACTION_WITHDRAW
		for i, vPubkey := range stakerInfo.ValidatorPubkeyList {
			if vPubkey == validatorPubkey {
				// TODO: len(stkaerInfo.ValidatorPubkeyList)==0 shoule equal to newBalance.Balance<=0
				stakerInfo.ValidatorPubkeyList = append(stakerInfo.ValidatorPubkeyList[:i], stakerInfo.ValidatorPubkeyList[i+1:]...)
				break
			}
		}
	}

	newBalance.Balance += amountInt64

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
			stakerInfo.StakerIndex = int64(idx)
			break
		}
	}
	if !exists {
		if amountInt64 <= 0 {
			return errors.New("remove unexist validator")
		}
		stakerList.StakerAddrs = append(stakerList.StakerAddrs, stakerAddr)
		valueStakerList = k.cdc.MustMarshal(&stakerList)
		store.Set(keyStakerList, valueStakerList)
		stakerInfo.StakerIndex = int64(len(stakerList.StakerAddrs) - 1)
	}

	if newBalance.Balance <= 0 {
		store.Delete(key)
	} else {
		stakerInfo.BalanceList = append(stakerInfo.BalanceList, &newBalance)
		bz := k.cdc.MustMarshal(stakerInfo)
		store.Set(key, bz)
	}

	// we use index to sync with client about status of stakerInfo.ValidatorPubkeyList
	eventValue := fmt.Sprintf("%d_%s_%d", stakerInfo.StakerIndex, validatorPubkey, newBalance.Index)
	if newBalance.Change == types.Action_ACTION_DEPOSIT {
		eventValue = fmt.Sprintf("%s_%s", types.AttributeValueNativeTokenDeposit, eventValue)
	} else {
		eventValue = fmt.Sprintf("%s_%s", types.AttributeValueNativeTokenWithdraw, eventValue)
	}
	// emit an event to tell a new valdiator added/or a validator is removed for the staker
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyNativeTokenChange, eventValue),
	))
	return nil
}

// UpdateNSTByBalanceChange updates balance info for staker under native-restaking asset of assetID when its balance changed by slash/refund on the source chain (beacon chain for eth)
func (k Keeper) UpdateNSTByBalanceChange(ctx sdk.Context, assetID string, rawData []byte, roundID uint64) error {
	if !IsLimitedChangeNST(assetID) {
		return types.ErrNSTAssetNotSupported
	}
	_, chainID, _ := assetstypes.ParseID(assetID)
	if len(rawData) < 32 {
		return errors.New("length of indicate maps for stakers shoule be exactly 32 bytes")
	}
	sl := k.GetStakerList(ctx, assetID)
	if len(sl.StakerAddrs) == 0 {
		return errors.New("staker list is empty")
	}
	stakerChanges, err := parseBalanceChangeCapped(rawData, sl)
	if err != nil {
		return fmt.Errorf("failed to parse balance changes: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	for _, stakerAddr := range sl.StakerAddrs {
		// if stakerAddr is not in stakerChanges, then the change would be set to 0 which is expected
		change := stakerChanges[stakerAddr]
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
		newBalance.Block = uint64(ctx.BlockHeight())
		if newBalance.RoundID == roundID {
			newBalance.Index++
		} else {
			newBalance.RoundID = roundID
			newBalance.Index = 0
		}
		newBalance.Change = types.Action_ACTION_SLASH_REFUND
		// balance update are based on initial/max effective balance: 32
		maxBalance := maxEffectiveBalance(assetID) * (len(stakerInfo.ValidatorPubkeyList))
		balance := maxBalance + change
		// there's one case that this delta might be more than previous Balance
		// staker's validatorlist: {v1, v2, v3, v5}
		// in one same block: withdraw v2, v3, v5, balance of v2, v3, v5 all be slashed by -16
		// => amount: 32*4->32(by withdraw), the validatorList of feeder will be updated on next block, so it will report the balance change of v5: -16 as in the staker's balance change, result to: 32*4->32-> 32-16*3 = -16
		// we will just ingore this misbehavior introduced by synchronize-issue, and this will be correct in next block/round
		if balance > maxBalance || balance <= 0 {
			// balance should not be able to be reduced to 0 by balance change
			return errors.New("effective balance should never exceeds 32 for one validator and should be positive")
		}

		if delta := int64(balance) - newBalance.Balance; delta != 0 {
			decimal, _, err := k.getDecimal(ctx, assetID)
			if err != nil {
				return err
			}
			if err := k.delegationKeeper.UpdateNSTBalance(ctx, getStakerID(stakerAddr, chainID), assetID, sdkmath.NewIntWithDecimal(delta, decimal)); err != nil {
				return err
			}
			newBalance.Balance = int64(balance)
		}
		//	newBalance.Balance += int64(change)
		stakerInfo.Append(&newBalance)
		bz := k.cdc.MustMarshal(stakerInfo)
		store.Set(key, bz)
	}
	return nil
}

func (k Keeper) getDecimal(ctx sdk.Context, assetID string) (int, sdkmath.Int, error) {
	decimalMap, err := k.assetsKeeper.GetAssetsDecimal(ctx, map[string]interface{}{assetID: nil})
	if err != nil {
		return 0, sdkmath.ZeroInt(), err
	}
	decimal := decimalMap[assetID]
	return int(decimal), sdkmath.NewIntWithDecimal(1, int(decimal)), nil
}

// TODO: This conversion has limited length for balance change, it suites for beaconchain currently, If we extend to other changes, this method need to be upgrade
// for value that might be too big leading too long length of the change value, many related changes need to be done since the message size might be too big then
// parseBalanceChange parses rawData to details of amount change for all stakers relative to native restaking
func parseBalanceChangeCapped(rawData []byte, sl types.StakerList) (map[string]int, error) {
	// eg. 0100-000011
	// first part 0100 tells that the effective-balance of staker corresponding to index 2 in StakerList
	// the lenft part 000011. we use the first 4 bits to tell the length of this number, and it shows as 1 here, the 5th bit is used to tell symbol of the number, 1 means negative, then we can get the abs number indicate by the length. It's -1 here, means effective-balane is 32-1 on beacon chain for now
	// the first 32 bytes are information to indicates effective-balance of which staker has changed, 1 means changed, 0 means not. 32 bytes can represents changes for at most 256 stakers
	indexes := rawData[:32]
	// bytes after first 32 are details of effective-balance change for each staker which has been marked with 1 in the first 32 bytes, for those who are marked with 0 will just be ignored
	// For each staker we support at most 256 validators to join, so the biggest effective-balance change we would have is 256*16, then we need 12 bits to represents the number for each staker. And for compression we use 4 bits to tell then length of bits without leading 0 this number has.
	// Then with the symbol we need at most 17 bits for each staker's effective-balance change: 0000.0.0000-0000-0000 (the leading 0 will be ignored for the last 12 bits)
	changes := rawData[32:]
	index := -1
	byteIndex := 0
	bitOffset := 0
	lengthBits := 5
	stakerChanges := make(map[string]int)
	for _, b := range indexes {
		for i := 7; i >= 0; i-- {
			index++
			if (b>>i)&1 == 1 {
				lenValue := changes[byteIndex] << bitOffset
				bitsLeft := 8 - bitOffset
				lenValue >>= (8 - lengthBits)
				if bitsLeft < lengthBits {
					byteIndex++
					lenValue |= changes[byteIndex] >> (8 - lengthBits + bitsLeft)
					bitOffset = lengthBits - bitsLeft
				} else {
					if bitOffset += lengthBits; bitOffset == 8 {
						bitOffset = 0
					}
					if bitsLeft == lengthBits {
						byteIndex++
					}
				}

				symbol := lenValue & 1
				lenValue >>= 1
				if lenValue <= 0 {
					return stakerChanges, errors.New("length of change value must be at least 1 bit")
				}

				bitsExtracted := 0
				stakerChange := 0
				for bitsExtracted < int(lenValue) {
					bitsLeft := 8 - bitOffset
					byteValue := changes[byteIndex] << bitOffset
					if (int(lenValue) - bitsExtracted) < bitsLeft {
						bitsLeft = int(lenValue) - bitsExtracted
						bitOffset += bitsLeft
					} else {
						byteIndex++
						bitOffset = 0
					}
					byteValue >>= (8 - bitsLeft)
					stakerChange = (stakerChange << bitsLeft) | int(byteValue)
					bitsExtracted += bitsLeft
				}
				stakerChange++
				if symbol == 1 {
					stakerChange *= -1
				}
				stakerChanges[sl.StakerAddrs[index]] = stakerChange
			}
		}
	}
	return stakerChanges, nil
}

// TODO use []byte and assetstypes.GetStakerIDAndAssetID for stakerAddr representation
func getStakerID(stakerAddr string, chainID uint64) string {
	return strings.Join([]string{strings.ToLower(stakerAddr), hexutil.EncodeUint64(chainID)}, utils.DelimiterForID)
}

// IsLimitChangesNST returns that is input assetID corresponding to asset which balance change has a cap limit
func IsLimitedChangeNST(assetID string) bool {
	return limitedChangeNST[NSTETHAssetID(assetID)]
}

func maxEffectiveBalance(assetID string) int {
	return maxEffectiveBalances[NSTETHAssetID(assetID)]
}
