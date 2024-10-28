package keeper

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"

	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

	"github.com/ExocoreNetwork/exocore/utils"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
)

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeKey       storetypes.StoreKey
		operatorKeeper types.OperatorKeeper
		// other keepers
		assetsKeeper types.AssetsKeeper
		epochsKeeper types.EpochsKeeper
		evmKeeper    types.EVMKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	operatorKeeper types.OperatorKeeper,
	assetKeeper types.AssetsKeeper,
	epochsKeeper types.EpochsKeeper,
	evmKeeper types.EVMKeeper,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		operatorKeeper: operatorKeeper,
		assetsKeeper:   assetKeeper,
		epochsKeeper:   epochsKeeper,
		evmKeeper:      evmKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetOperatorKeeper returns the operatorKeeper from the Keeper struct.
func (k Keeper) GetOperatorKeeper() types.OperatorKeeper {
	return k.operatorKeeper
}

func (k Keeper) ValidateAssetIDs(ctx sdk.Context, assetIDs []string) error {
	for _, assetID := range assetIDs {
		if !k.assetsKeeper.IsStakingAsset(ctx, assetID) {
			return errorsmod.Wrap(types.ErrInvalidAssetID, fmt.Sprintf("Invalid assetID: %s", assetID))
		}
	}
	return nil
}

func (k Keeper) UpdateAVSInfo(ctx sdk.Context, params *types.AVSRegisterOrDeregisterParams) error {
	avsInfo, _ := k.GetAVSInfo(ctx, params.AvsAddress)
	action := params.Action
	epochIdentifier := params.EpochIdentifier
	if avsInfo != nil && avsInfo.Info.EpochIdentifier != "" {
		epochIdentifier = avsInfo.Info.EpochIdentifier
	}
	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !found {
		return errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s", epochIdentifier))
	}
	switch action {
	case RegisterAction:
		if avsInfo != nil {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		if k.GetAVSInfoByTaskAddress(ctx, params.TaskAddr).AvsAddress != "" {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("this TaskAddr has already been used by other AVS,the TaskAddr is :%s", params.TaskAddr))
		}
		startingEpoch := uint64(epoch.CurrentEpoch + 1)
		if params.ChainID == utils.ChainIDWithoutRevision(ctx.ChainID()) {
			// TODO: handle this better
			startingEpoch = uint64(epoch.CurrentEpoch)
		}

		if err := k.ValidateAssetIDs(ctx, params.AssetID); err != nil {
			return err
		}

		avs := &types.AVSInfo{
			Name:                params.AvsName,
			AvsAddress:          params.AvsAddress,
			RewardAddr:          params.RewardContractAddr,
			SlashAddr:           params.SlashContractAddr,
			AvsOwnerAddress:     params.AvsOwnerAddress,
			AssetIDs:            params.AssetID,
			MinSelfDelegation:   params.MinSelfDelegation,
			AvsUnbondingPeriod:  params.UnbondingPeriod,
			EpochIdentifier:     epochIdentifier,
			StartingEpoch:       startingEpoch,
			MinOptInOperators:   params.MinOptInOperators,
			TaskAddr:            params.TaskAddr,
			MinStakeAmount:      params.MinStakeAmount, // Effective at CurrentEpoch+1, avoid immediate effects and ensure that the first epoch time of avs is equal to a normal identifier
			MinTotalStakeAmount: params.MinTotalStakeAmount,
			// #nosec G115
			AvsSlash: sdk.NewDecWithPrec(int64(params.AvsSlash), 2),
			// #nosec G115
			AvsReward: sdk.NewDecWithPrec(int64(params.AvsReward), 2),
		}

		return k.SetAVSInfo(ctx, avs)
	case DeRegisterAction:
		if avsInfo == nil {
			return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		// If avs DeRegisterAction check CallerAddress
		if !slices.Contains(avsInfo.Info.AvsOwnerAddress, params.CallerAddress) {
			return errorsmod.Wrap(types.ErrCallerAddressUnauthorized, fmt.Sprintf("this caller not qualified to deregister %s", params.CallerAddress))
		}

		// If avs DeRegisterAction check UnbondingPeriod
		// #nosec G115
		if epoch.CurrentEpoch-int64(avsInfo.GetInfo().StartingEpoch) > int64(avsInfo.Info.AvsUnbondingPeriod) {
			return errorsmod.Wrap(types.ErrUnbondingPeriod, fmt.Sprintf("not qualified to deregister %s", avsInfo))
		}

		// If avs DeRegisterAction check avsname
		if avsInfo.Info.Name != params.AvsName {
			return errorsmod.Wrap(types.ErrAvsNameMismatch, fmt.Sprintf("Unregistered AVS name is incorrect %s", params.AvsName))
		}
		return k.DeleteAVSInfo(ctx, params.AvsAddress)
	case UpdateAction:
		if avsInfo == nil {
			return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		// Check here to ensure that the task address is only used  by one avs
		avsAddress := k.GetAVSInfoByTaskAddress(ctx, params.TaskAddr).AvsAddress
		if avsAddress != "" && avsAddress != avsInfo.Info.AvsAddress {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("this TaskAddr has already been used by other AVS,the TaskAddr is :%s", params.TaskAddr))
		}
		// TODO: The AvsUnbondingPeriod is used for undelegation, but this check currently blocks updates to AVS information. Remove this check to allow AVS updates, while detailed control mechanisms for updates should be considered and implemented in the future.
		// If avs UpdateAction check UnbondingPeriod

		// #nosec G115
		//	if int64(avsInfo.Info.AvsUnbondingPeriod) < (epoch.CurrentEpoch - int64(avsInfo.GetInfo().StartingEpoch)) {
		//	return errorsmod.Wrap(types.ErrUnbondingPeriod, fmt.Sprintf("not qualified to deregister %s", avsInfo))
		// }
		// If avs UpdateAction check CallerAddress

		avs := avsInfo.Info

		if params.AvsName != "" {
			avs.Name = params.AvsName
		}
		if params.MinStakeAmount > 0 {
			avs.MinStakeAmount = params.MinStakeAmount
		}
		if params.TaskAddr != "" {
			avs.TaskAddr = params.TaskAddr
		}
		if params.SlashContractAddr != "" {
			avs.SlashAddr = params.SlashContractAddr
		}
		if params.RewardContractAddr != "" {
			avs.RewardAddr = params.RewardContractAddr
		}
		if params.AvsOwnerAddress != nil {
			avs.AvsOwnerAddress = params.AvsOwnerAddress
		}
		if params.AssetID != nil {
			avs.AssetIDs = params.AssetID
			if err := k.ValidateAssetIDs(ctx, params.AssetID); err != nil {
				return err
			}
		}

		if params.UnbondingPeriod > 0 {
			avs.AvsUnbondingPeriod = params.UnbondingPeriod
		}

		avs.MinSelfDelegation = params.MinSelfDelegation

		if params.EpochIdentifier != "" {
			avs.EpochIdentifier = params.EpochIdentifier
		}

		if params.MinOptInOperators > 0 {
			avs.MinOptInOperators = params.MinOptInOperators
		}
		if params.MinTotalStakeAmount > 0 {
			avs.MinTotalStakeAmount = params.MinTotalStakeAmount
		}
		if params.AvsSlash > 0 {
			// #nosec G115
			avs.AvsSlash = sdk.NewDecWithPrec(int64(params.AvsSlash), 2)
		}
		if params.AvsReward > 0 {
			// #nosec G115
			avs.AvsReward = sdk.NewDecWithPrec(int64(params.AvsReward), 2)
		}
		avs.AvsAddress = params.AvsAddress
		avs.StartingEpoch = uint64(epoch.CurrentEpoch + 1)

		return k.SetAVSInfo(ctx, avs)
	default:
		return errorsmod.Wrap(types.ErrInvalidAction, fmt.Sprintf("Invalid action: %d", action))
	}
}

func (k Keeper) CreateAVSTask(ctx sdk.Context, params *TaskInfoParams) error {
	avsInfo := k.GetAVSInfoByTaskAddress(ctx, params.TaskContractAddress)
	if avsInfo.AvsAddress == "" {
		return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the taskaddr is :%s", params.TaskContractAddress))
	}
	// If avs CreateAVSTask check CallerAddress
	if !slices.Contains(avsInfo.AvsOwnerAddress, params.CallerAddress) {
		return errorsmod.Wrap(types.ErrCallerAddressUnauthorized, fmt.Sprintf("this caller not qualified to CreateAVSTask %s", params.CallerAddress))
	}
	taskPowerTotal, err := k.operatorKeeper.GetAVSUSDValue(ctx, avsInfo.AvsAddress)

	if err != nil || taskPowerTotal.IsZero() || taskPowerTotal.IsNegative() {
		return errorsmod.Wrap(types.ErrVotingPowerIncorrect, fmt.Sprintf("the votingpower of avs is <<=0,avs addr is：%s", avsInfo.AvsAddress))
	}

	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	if !found {
		return errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s", avsInfo.EpochIdentifier))
	}

	if k.IsExistTask(ctx, strconv.FormatUint(params.TaskID, 10), params.TaskContractAddress) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the task is :%s", strconv.FormatUint(params.TaskID, 10)))
	}
	operatorList, err := k.GetOptInOperators(ctx, avsInfo.AvsAddress)
	if err != nil {
		return errorsmod.Wrap(err, "CreateAVSTask: failed to get opt-in operators")
	}
	params.TaskID = k.GetTaskID(ctx, common.HexToAddress(params.TaskContractAddress))
	task := &types.TaskInfo{
		Name:                  params.TaskName,
		Hash:                  params.Hash,
		TaskContractAddress:   params.TaskContractAddress,
		TaskId:                params.TaskID,
		TaskChallengePeriod:   params.TaskChallengePeriod,
		ThresholdPercentage:   params.ThresholdPercentage,
		TaskResponsePeriod:    params.TaskResponsePeriod,
		TaskStatisticalPeriod: params.TaskStatisticalPeriod,
		StartingEpoch:         uint64(epoch.CurrentEpoch + 1),
		ActualThreshold:       0,
		OptInOperators:        operatorList,
	}
	return k.SetTaskInfo(ctx, task)
}

func (k Keeper) RegisterBLSPublicKey(ctx sdk.Context, params *BlsParams) error {
	// check bls signature to prevent rogue key attacks
	sig := params.PubkeyRegistrationSignature
	msgHash := params.PubkeyRegistrationMessageHash
	pubKey, _ := bls.PublicKeyFromBytes(params.PubKey)
	valid, err := blst.VerifySignature(sig, [32]byte(msgHash), pubKey)
	if err != nil || !valid {
		return errorsmod.Wrap(types.ErrSigNotMatchPubKey, fmt.Sprintf("the operator is :%s", params.Operator))
	}

	if k.IsExistPubKey(ctx, params.Operator) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the operator is :%s", params.Operator))
	}
	bls := &types.BlsPubKeyInfo{
		Name:     params.Name,
		Operator: params.Operator,
		PubKey:   params.PubKey,
	}
	return k.SetOperatorPubKey(ctx, bls)
}

func (k Keeper) GetOptInOperators(ctx sdk.Context, avsAddr string) ([]string, error) {
	return k.operatorKeeper.GetOptedInOperatorListByAVS(ctx, avsAddr)
}

func (k Keeper) OperatorOptAction(ctx sdk.Context, params *OperatorOptParams) error {
	operatorAddress := params.OperatorAddress
	opAccAddr, err := sdk.AccAddressFromBech32(operatorAddress)
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error occurred when parse acc address from Bech32,the addr is:%s", operatorAddress))
	}

	if !k.operatorKeeper.IsOperator(ctx, opAccAddr) {
		return errorsmod.Wrap(delegationtypes.ErrOperatorNotExist, fmt.Sprintf("UpdateAVSInfo: invalid operator address:%s", operatorAddress))
	}

	f, err := k.IsAVS(ctx, params.AvsAddress)
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error occurred when get avs info,this avs address: %s", params.AvsAddress))
	}
	if !f {
		return fmt.Errorf("avs does not exist,this avs address: %s", params.AvsAddress)
	}

	switch params.Action {
	case RegisterAction:
		return k.operatorKeeper.OptIn(ctx, opAccAddr, params.AvsAddress)
	case DeRegisterAction:
		return k.operatorKeeper.OptOut(ctx, opAccAddr, params.AvsAddress)
	default:
		return errorsmod.Wrap(types.ErrInvalidAction, fmt.Sprintf("Invalid action: %d", params.Action))
	}
}

// SetAVSInfo sets the avs info. The caller must ensure that avs.AvsAddress is hex.
func (k Keeper) SetAVSInfo(ctx sdk.Context, avs *types.AVSInfo) (err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)
	bz := k.cdc.MustMarshal(avs)
	store.Set(common.HexToAddress(avs.AvsAddress).Bytes(), bz)
	return nil
}

func (k Keeper) GetAVSInfo(ctx sdk.Context, addr string) (*types.QueryAVSInfoResponse, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)
	value := store.Get(common.HexToAddress(addr).Bytes())
	if value == nil {
		return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("GetAVSInfo: key is %s", addr))
	}
	ret := types.AVSInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	res := &types.QueryAVSInfoResponse{
		Info: &ret,
	}
	return res, nil
}

func (k *Keeper) IsAVS(ctx sdk.Context, addr string) (bool, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)
	return store.Has(common.HexToAddress(addr).Bytes()), nil
}

// IsAVSByChainID queries whether an AVS exists by chainID.
// It returns the AVS address if it exists.
func (k Keeper) IsAVSByChainID(ctx sdk.Context, chainID string) (bool, string) {
	avsAddrStr := types.GenerateAVSAddr(chainID)
	avsAddr := common.HexToAddress(avsAddrStr)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)
	return store.Has(avsAddr.Bytes()), avsAddrStr
}

func (k Keeper) DeleteAVSInfo(ctx sdk.Context, addr string) error {
	hexAddr := common.HexToAddress(addr)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)
	if !store.Has(hexAddr.Bytes()) {
		return errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("AVSInfo didn't exist: key is %s", addr))
	}
	store.Delete(hexAddr[:])
	return nil
}

// IterateAVSInfo iterate through avs
func (k Keeper) IterateAVSInfo(ctx sdk.Context, fn func(index int64, avsInfo types.AVSInfo) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSInfo)

	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		avs := types.AVSInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &avs)

		stop := fn(i, avs)

		if stop {
			break
		}
		i++
	}
}

func (k Keeper) RaiseAndResolveChallenge(ctx sdk.Context, params *ChallengeParams) error {
	taskInfo, err := k.GetTaskInfo(ctx, strconv.FormatUint(params.TaskID, 10), params.TaskContractAddress.String())
	if err != nil {
		return fmt.Errorf("task does not exist,this task address: %s", params.TaskContractAddress)
	}
	// check Task
	if hex.EncodeToString(taskInfo.Hash) != hex.EncodeToString(params.TaskHash) {
		return errorsmod.Wrap(err, fmt.Sprintf("error Task hasn't been responded to yet: %s", params.TaskContractAddress))
	}
	// check Task result
	res, err := k.GetTaskResultInfo(ctx, params.OperatorAddress.String(), params.TaskContractAddress.String(),
		params.TaskID)
	if err != nil {
		return fmt.Errorf("task result does not exist, this task address: %s", params.TaskContractAddress)
	}
	taskRes, err := types.UnmarshalTaskResponse(res.TaskResponse)
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error occurred when unmarshal task response, this task address: %s", params.TaskContractAddress))
	}
	hash, err := types.GetTaskResponseDigestEncodeByAbi(taskRes)

	if err != nil || res.TaskId != params.TaskID || hex.EncodeToString(hash[:]) != hex.EncodeToString(params.TaskResponseHash) {
		return errorsmod.Wrap(
			types.ErrInconsistentParams,
			fmt.Sprintf("Task response does not match the one recorded,task addr: %s ,(TaskContractAddress: %s)"+
				",(TaskId: %d),(TaskResponseHash: %s)",
				params.OperatorAddress, params.TaskContractAddress, params.TaskID, params.TaskResponseHash),
		)
	}
	// check challenge record
	if k.IsExistTaskChallengedInfo(ctx, params.OperatorAddress.String(),
		params.TaskContractAddress.String(), params.TaskID) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the challenge has been raised: %s", params.TaskContractAddress))
	}

	// check challenge period
	//  check epoch，The challenge must be within the challenge window period
	avsInfo := k.GetAVSInfoByTaskAddress(ctx, taskInfo.TaskContractAddress)
	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	if !found {
		return errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s",
			avsInfo.EpochIdentifier))
	}
	if epoch.CurrentEpoch <= int64(taskInfo.StartingEpoch)+int64(taskInfo.TaskResponsePeriod)+int64(taskInfo.TaskStatisticalPeriod) {
		return errorsmod.Wrap(
			types.ErrSubmitTooSoonError,
			fmt.Sprintf("SetTaskResultInfo:the challenge period has not started , CurrentEpoch:%d", epoch.CurrentEpoch),
		)
	}
	if epoch.CurrentEpoch > int64(taskInfo.StartingEpoch)+int64(taskInfo.TaskResponsePeriod)+int64(taskInfo.TaskStatisticalPeriod)+int64(taskInfo.TaskChallengePeriod) {
		return errorsmod.Wrap(
			types.ErrSubmitTooLateError,
			fmt.Sprintf("SetTaskResultInfo:submit  too late, CurrentEpoch:%d", epoch.CurrentEpoch),
		)
	}
	return k.SetTaskChallengedInfo(ctx, params.TaskID, params.OperatorAddress.String(), params.CallerAddress,
		params.TaskContractAddress)
}
