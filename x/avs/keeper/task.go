package keeper

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k Keeper) SetTaskInfo(ctx sdk.Context, task *types.TaskInfo) (err error) {
	if !common.IsHexAddress(task.TaskContractAddress) {
		return types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSTaskInfo)
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(task.TaskContractAddress), strconv.FormatUint(task.TaskId, 10))
	bz := k.cdc.MustMarshal(task)
	store.Set(infoKey, bz)
	return nil
}

func (k *Keeper) GetTaskInfo(ctx sdk.Context, taskID, taskContractAddress string) (info *types.TaskInfo, err error) {
	if !common.IsHexAddress(taskContractAddress) {
		return nil, types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSTaskInfo)
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(taskContractAddress), taskID)
	value := store.Get(infoKey)
	if value == nil {
		return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore,
			fmt.Sprintf("GetTaskInfo: key not found for task ID %s at contract address %s", taskID, taskContractAddress))
	}

	ret := types.TaskInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k Keeper) GetAllTaskInfos(ctx sdk.Context) ([]types.TaskInfo, error) {
	var taskInfos []types.TaskInfo
	k.IterateTaskAVSInfo(ctx, func(_ int64, taskInfo types.TaskInfo) bool {
		taskInfos = append(taskInfos, taskInfo)
		return false
	})
	return taskInfos, nil
}

func (k *Keeper) IsExistTask(ctx sdk.Context, taskID, taskContractAddress string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSTaskInfo)
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(taskContractAddress), taskID)

	return store.Has(infoKey)
}

func (k *Keeper) SetOperatorPubKey(ctx sdk.Context, pub *types.BlsPubKeyInfo) (err error) {
	operatorAddress, err := sdk.AccAddressFromBech32(pub.Operator)
	if err != nil {
		return types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatePub)
	bz := k.cdc.MustMarshal(pub)
	store.Set(operatorAddress, bz)
	return nil
}

func (k *Keeper) GetOperatorPubKey(ctx sdk.Context, addr string) (pub *types.BlsPubKeyInfo, err error) {
	opAccAddr, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "GetOperatorPubKey: error occurred when parsing account address from Bech32: "+addr)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatePub)
	// key := common.HexToAddress(incentive.Contract)
	isExist := store.Has(opAccAddr)
	if !isExist {
		return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore,
			fmt.Sprintf("GetOperatorPubKey: public key not found for address %s", opAccAddr))
	}
	value := store.Get(opAccAddr)
	ret := types.BlsPubKeyInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k *Keeper) GetAllBlsPubKeys(ctx sdk.Context) ([]types.BlsPubKeyInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatePub)
	// Count items first for optimal slice allocation
	iterator := sdk.KVStorePrefixIterator(store, nil)
	// Pre-allocate the slice for better performance by counting items first.
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	iterator.Close()

	// Reset iterator
	iterator = sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	pubKeys := make([]types.BlsPubKeyInfo, 0, count)
	for ; iterator.Valid(); iterator.Next() {
		var pubKey types.BlsPubKeyInfo
		err := k.cdc.Unmarshal(iterator.Value(), &pubKey)
		if err != nil {
			return nil, errorsmod.Wrap(err, "GetAllBlsPubKeys: failed to unmarshal pubkey")
		}
		pubKeys = append(pubKeys, pubKey)
	}

	return pubKeys, nil
}

func (k *Keeper) IsExistPubKey(ctx sdk.Context, addr string) bool {
	opAccAddr, _ := sdk.AccAddressFromBech32(addr)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatePub)
	return store.Has(opAccAddr)
}

// IterateTaskAVSInfo iterate through task
func (k Keeper) IterateTaskAVSInfo(ctx sdk.Context, fn func(index int64, taskInfo types.TaskInfo) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSTaskInfo)

	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		task := types.TaskInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &task)

		stop := fn(i, task)

		if stop {
			break
		}
		i++
	}
}

func (k Keeper) SetTaskID(ctx sdk.Context, taskAddr common.Address, id uint64) {
	if taskAddr == (common.Address{}) {
		panic("invalid task address")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixLatestTaskNum)
	store.Set(taskAddr.Bytes(), sdk.Uint64ToBigEndian(id))
}

// GetTaskID Increase the task ID by 1 each time.
func (k Keeper) GetTaskID(ctx sdk.Context, taskAddr common.Address) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixLatestTaskNum)
	var id uint64
	if store.Has(taskAddr.Bytes()) {
		bz := store.Get(taskAddr.Bytes())
		id = sdk.BigEndianToUint64(bz)
		id++
	} else {
		id = 1
	}
	store.Set(taskAddr.Bytes(), sdk.Uint64ToBigEndian(id))
	return id
}

// GetAllTaskNums returns a map containing all task addresses and their corresponding task IDs.
func (k *Keeper) GetAllTaskNums(ctx sdk.Context) ([]types.TaskID, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixLatestTaskNum)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()
	ret := make([]types.TaskID, 0)
	for ; iterator.Valid(); iterator.Next() {
		taskAddr := strings.ToLower(common.BytesToAddress(iterator.Key()).Hex())
		id := sdk.BigEndianToUint64(iterator.Value())
		ret = append(ret, types.TaskID{
			TaskAddr: taskAddr,
			TaskId:   id,
		})
	}
	return ret, nil
}

// SetTaskResultInfo is used to store the operator's sign task information.
func (k *Keeper) SetTaskResultInfo(
	ctx sdk.Context, info *types.TaskResultInfo,
) {
	if _, err := sdk.AccAddressFromBech32(info.OperatorAddress); err != nil {
		panic(fmt.Sprintf("invalid operator address: %s", info.OperatorAddress))
	}
	if !common.IsHexAddress(info.TaskContractAddress) {
		panic(fmt.Sprintf("invalid task contract address: %s", info.TaskContractAddress))
	}
	infoKey := assetstype.GetJoinedStoreKey(info.OperatorAddress, strings.ToLower(info.TaskContractAddress),
		strconv.FormatUint(info.TaskId, 10))
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskResult)
	bz := k.cdc.MustMarshal(info)
	store.Set(infoKey, bz)
}

func (k *Keeper) IsExistTaskResultInfo(ctx sdk.Context, operatorAddress, taskContractAddress string, taskID uint64) bool {
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(operatorAddress), strings.ToLower(taskContractAddress),
		strconv.FormatUint(taskID, 10))
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskResult)
	return store.Has(infoKey)
}

func (k *Keeper) GetTaskResultInfo(ctx sdk.Context, operatorAddress, taskContractAddress string, taskID uint64) (info *types.TaskResultInfo, err error) {
	if !common.IsHexAddress(taskContractAddress) {
		return nil, types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskResult)
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(operatorAddress), strings.ToLower(taskContractAddress),
		strconv.FormatUint(taskID, 10))
	value := store.Get(infoKey)
	if value == nil {
		return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore,
			fmt.Sprintf("GetTaskResultInfo: key is %s", infoKey))
	}

	ret := types.TaskResultInfo{}
	if err := k.cdc.Unmarshal(value, &ret); err != nil {
		return nil, errorsmod.Wrap(err, "GetTaskResultInfo: failed to unmarshal task result info")
	}
	return &ret, nil
}

// IterateResultInfo iterate through task result info
func (k Keeper) IterateResultInfo(ctx sdk.Context, fn func(index int64, info types.TaskResultInfo) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskResult)

	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		task := types.TaskResultInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &task)

		stop := fn(i, task)

		if stop {
			break
		}
		i++
	}
}

func (k Keeper) GroupTasksByIDAndAddress(tasks []types.TaskResultInfo) map[string][]types.TaskResultInfo {
	taskMap := make(map[string][]types.TaskResultInfo)
	for _, task := range tasks {
		key := task.TaskContractAddress + "_" + strconv.FormatUint(task.TaskId, 10)
		taskMap[key] = append(taskMap[key], task)
	}

	// Sort tasks in each group by OperatorAddress
	for key, taskGroup := range taskMap {
		sort.Slice(taskGroup, func(i, j int) bool {
			return taskGroup[i].OperatorAddress < taskGroup[j].OperatorAddress
		})
		taskMap[key] = taskGroup
	}
	return taskMap
}

// SetTaskChallengedInfo is used to store the challenger's challenge information.
func (k *Keeper) SetTaskChallengedInfo(
	ctx sdk.Context, taskID uint64, operatorAddress, challengeAddr string,
	taskAddr common.Address,
) (err error) {
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(operatorAddress), strings.ToLower(taskAddr.String()),
		strconv.FormatUint(taskID, 10))

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskChallengeResult)
	key, err := sdk.AccAddressFromBech32(challengeAddr)
	if err != nil {
		return err
	}
	store.Set(infoKey, key)

	return nil
}

func (k *Keeper) IsExistTaskChallengedInfo(ctx sdk.Context, operatorAddress, taskContractAddress string, taskID uint64) bool {
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(operatorAddress), strings.ToLower(taskContractAddress),
		strconv.FormatUint(taskID, 10))
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskChallengeResult)
	return store.Has(infoKey)
}

func (k *Keeper) GetTaskChallengedInfo(ctx sdk.Context, operatorAddress, taskContractAddress string, taskID uint64) (addr string, err error) {
	if !common.IsHexAddress(taskContractAddress) {
		return "", types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskChallengeResult)
	infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(operatorAddress), strings.ToLower(taskContractAddress),
		strconv.FormatUint(taskID, 10))
	value := store.Get(infoKey)
	if value == nil {
		return "", errorsmod.Wrap(types.ErrNoKeyInTheStore,
			fmt.Sprintf("GetTaskChallengedInfo: key is %s", infoKey))
	}

	return common.Bytes2Hex(value), nil
}

// GetAllTaskResultInfos returns a slice containing all task result information.
func (k *Keeper) GetAllTaskResultInfos(ctx sdk.Context) ([]types.TaskResultInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskResult)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]types.TaskResultInfo, 0)
	for ; iterator.Valid(); iterator.Next() {
		task := types.TaskResultInfo{}
		err := k.cdc.Unmarshal(iterator.Value(), &task)
		if err != nil {
			return nil, errorsmod.Wrap(err, "GetAllTaskResultInfos: failed to unmarshal task result info")
		}
		ret = append(ret, task)
	}
	return ret, nil
}

func (k *Keeper) SetAllTaskChallengedInfo(ctx sdk.Context, states []types.ChallengeInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskChallengeResult)
	for i := range states {
		state := states[i]
		bz, err := sdk.AccAddressFromBech32(state.ChallengeAddr)
		if err != nil {
			return err
		}
		store.Set([]byte(state.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllChallengeInfos(ctx sdk.Context) ([]types.ChallengeInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTaskChallengeResult)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	// Count items first for optimal slice allocation
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	iterator.Close()

	// Reset iterator
	iterator = sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]types.ChallengeInfo, 0, count)
	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())
		challengeAddr := sdk.AccAddress(iterator.Value())
		if len(challengeAddr) == 0 {
			return nil, errorsmod.Wrap(types.ErrInvalidAddr, "invalid challenge address")
		}

		ret = append(ret, types.ChallengeInfo{
			Key:           key,
			ChallengeAddr: challengeAddr.String(),
		})
	}
	return ret, nil
}
