package keeper

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"

	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"

	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

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

// GetEpochKeeper returns the operatorKeeper from the Keeper struct.
func (k Keeper) GetEpochKeeper() types.EpochsKeeper {
	return k.epochsKeeper
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
	avsInfo, _ := k.GetAVSInfo(ctx, params.AvsAddress.String())
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
	case types.RegisterAction:
		if avsInfo != nil {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		if k.GetAVSInfoByTaskAddress(ctx, params.TaskAddress.String()).AvsAddress != "" {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("this TaskAddr has already been used by other AVS,the TaskAddr is :%s", params.TaskAddress))
		}
		startingEpoch := uint64(epoch.CurrentEpoch + 1)
		if params.ChainID == types.ChainIDWithoutRevision(ctx.ChainID()) {
			// TODO: handle this better
			startingEpoch = uint64(epoch.CurrentEpoch)
		}

		if err := k.ValidateAssetIDs(ctx, params.AssetIDs); err != nil {
			return err
		}

		avs := &types.AVSInfo{
			Name:                params.AvsName,
			AvsAddress:          strings.ToLower(params.AvsAddress.String()),
			RewardAddress:       strings.ToLower(params.RewardContractAddress.String()),
			SlashAddress:        strings.ToLower(params.SlashContractAddress.String()),
			AvsOwnerAddresses:   params.AvsOwnerAddresses,
			AssetIDs:            params.AssetIDs,
			MinSelfDelegation:   params.MinSelfDelegation,
			AvsUnbondingPeriod:  params.UnbondingPeriod,
			EpochIdentifier:     epochIdentifier,
			StartingEpoch:       startingEpoch,
			MinOptInOperators:   params.MinOptInOperators,
			TaskAddress:         strings.ToLower(params.TaskAddress.String()),
			MinStakeAmount:      params.MinStakeAmount, // Effective at CurrentEpoch+1, avoid immediate effects and ensure that the first epoch time of avs is equal to a normal identifier
			MinTotalStakeAmount: params.MinTotalStakeAmount,
			// #nosec G115
			AvsSlash: sdk.NewDecWithPrec(int64(params.AvsSlash), 2),
			// #nosec G115
			AvsReward: sdk.NewDecWithPrec(int64(params.AvsReward), 2),
			// whitelist addresses are already validated
			WhitelistAddresses: params.WhitelistAddresses,
		}

		if err := k.SetAVSInfo(ctx, avs); err != nil {
			return err
		}
		// emit the event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeAvsCreated,
				sdk.NewAttribute(types.AttributeKeyAvsAddress, avs.AvsAddress),
			),
		)

		return nil

	case types.DeRegisterAction:
		if avsInfo == nil {
			return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		// If avs DeRegisterAction check CallerAddress
		if !slices.Contains(avsInfo.Info.AvsOwnerAddresses, params.CallerAddress.String()) {
			return errorsmod.Wrap(types.ErrCallerAddressUnauthorized, fmt.Sprintf("this caller not qualified to deregister %s", params.CallerAddress))
		}

		// If avs DeRegisterAction check UnbondingPeriod
		// #nosec G115
		if epoch.CurrentEpoch-int64(avsInfo.GetInfo().StartingEpoch) <= int64(avsInfo.Info.AvsUnbondingPeriod) {
			return errorsmod.Wrap(types.ErrUnbondingPeriod, fmt.Sprintf("not qualified to deregister %s", avsInfo))
		}

		// If avs DeRegisterAction check avsname
		if avsInfo.Info.Name != params.AvsName {
			return errorsmod.Wrap(types.ErrAvsNameMismatch, fmt.Sprintf("Unregistered AVS name is incorrect %s", params.AvsName))
		}
		return k.DeleteAVSInfo(ctx, params.AvsAddress.String())
	case types.UpdateAction:
		if avsInfo == nil {
			return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the avsaddress is :%s", params.AvsAddress))
		}
		// Check here to ensure that the task address is only used  by one avs
		avsAddress := k.GetAVSInfoByTaskAddress(ctx, params.TaskAddress.String()).AvsAddress
		if avsAddress != "" && avsAddress != avsInfo.Info.AvsAddress {
			return errorsmod.Wrap(types.ErrAlreadyRegistered, fmt.Sprintf("this TaskAddr has already been used by other AVS,the TaskAddr is :%s", params.TaskAddress))
		}
		// TODO: The AvsUnbondingPeriod is used for undelegation, but this check currently blocks updates to AVS information. Remove this check to allow AVS updates, while detailed control mechanisms for updates should be considered and implemented in the future.
		// If avs UpdateAction check UnbondingPeriod

		avs := avsInfo.Info

		if params.AvsName != "" {
			avs.Name = params.AvsName
		}
		if params.MinStakeAmount > 0 {
			avs.MinStakeAmount = params.MinStakeAmount
		}
		if params.TaskAddress.String() != "" {
			avs.TaskAddress = strings.ToLower(params.TaskAddress.String())
		}
		if params.SlashContractAddress.String() != "" {
			avs.SlashAddress = strings.ToLower(params.SlashContractAddress.String())
		}
		if params.RewardContractAddress.String() != "" {
			avs.RewardAddress = strings.ToLower(params.RewardContractAddress.String())
		}
		if params.AvsOwnerAddresses != nil {
			avs.AvsOwnerAddresses = params.AvsOwnerAddresses
		}
		if params.WhitelistAddresses != nil {
			avs.WhitelistAddresses = params.WhitelistAddresses
		}
		if params.AssetIDs != nil {
			avs.AssetIDs = params.AssetIDs
			if err := k.ValidateAssetIDs(ctx, params.AssetIDs); err != nil {
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
		avs.AvsAddress = params.AvsAddress.String()
		avs.StartingEpoch = uint64(epoch.CurrentEpoch + 1)

		return k.SetAVSInfo(ctx, avs)
	default:
		return errorsmod.Wrap(types.ErrInvalidAction, fmt.Sprintf("Invalid action: %d", action))
	}
}

func (k Keeper) CreateAVSTask(ctx sdk.Context, params *types.TaskInfoParams) (uint64, error) {
	avsInfo := k.GetAVSInfoByTaskAddress(ctx, params.TaskContractAddress.String())
	if avsInfo.AvsAddress == "" {
		return types.InvalidTaskID, errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the taskaddr is :%s", params.TaskContractAddress))
	}
	// If avs CreateAVSTask check CallerAddress
	if !slices.Contains(avsInfo.AvsOwnerAddresses, params.CallerAddress.String()) {
		return types.InvalidTaskID, errorsmod.Wrap(types.ErrCallerAddressUnauthorized, fmt.Sprintf("this caller not qualified to CreateAVSTask %s", params.CallerAddress))
	}
	taskPowerTotal, err := k.operatorKeeper.GetAVSUSDValue(ctx, avsInfo.AvsAddress)
	if err != nil {
		return types.InvalidTaskID, errorsmod.Wrap(err, "failed to get AVS USD value")
	}
	if taskPowerTotal.IsZero() || taskPowerTotal.IsNegative() {
		return types.InvalidTaskID, errorsmod.Wrap(types.ErrVotingPowerIncorrect, fmt.Sprintf("the voting power of AVS is zero or negative, AVS address: %s", avsInfo.AvsAddress))
	}

	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	if !found {
		return types.InvalidTaskID, errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s", avsInfo.EpochIdentifier))
	}

	if k.IsExistTask(ctx, strconv.FormatUint(params.TaskID, 10), params.TaskContractAddress.String()) {
		return types.InvalidTaskID, errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the task is :%s", strconv.FormatUint(params.TaskID, 10)))
	}
	operatorList, err := k.operatorKeeper.GetOptedInOperatorListByAVS(ctx, avsInfo.AvsAddress)
	if err != nil {
		return types.InvalidTaskID, errorsmod.Wrap(err, "CreateAVSTask: failed to get opt-in operators")
	}
	params.TaskID = k.GetTaskID(ctx, common.HexToAddress(params.TaskContractAddress.String()))
	task := &types.TaskInfo{
		Name:                  params.TaskName,
		Hash:                  params.Hash,
		TaskContractAddress:   strings.ToLower(params.TaskContractAddress.String()),
		TaskId:                params.TaskID,
		TaskChallengePeriod:   params.TaskChallengePeriod,
		ThresholdPercentage:   uint64(params.ThresholdPercentage),
		TaskResponsePeriod:    params.TaskResponsePeriod,
		TaskStatisticalPeriod: params.TaskStatisticalPeriod,
		StartingEpoch:         uint64(epoch.CurrentEpoch + 1),
		OptInOperators:        operatorList,
	}
	return task.TaskId, k.SetTaskInfo(ctx, task)
}

func (k Keeper) RegisterBLSPublicKey(ctx sdk.Context, params *types.BlsParams) error {
	// check bls signature to prevent rogue key attacks
	sig := params.PubkeyRegistrationSignature
	msgHash := params.PubkeyRegistrationMessageHash
	pubKey, _ := bls.PublicKeyFromBytes(params.PubKey)
	valid, err := blst.VerifySignature(sig, [32]byte(msgHash), pubKey)
	if err != nil || !valid {
		return errorsmod.Wrap(types.ErrSigNotMatchPubKey, fmt.Sprintf("the operator is :%s", params.OperatorAddress))
	}
	if k.IsExistPubKeyForAVS(ctx, params.OperatorAddress.String(), params.AvsAddress.String()) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the operator is :%s", params.OperatorAddress))
	}
	bls := &types.BlsPubKeyInfo{
		AvsAddress:      strings.ToLower(params.AvsAddress.String()),
		OperatorAddress: strings.ToLower(params.OperatorAddress.String()),
		PubKey:          params.PubKey,
	}
	// check a bls key can only be used once.
	// if operator are using multiple servers for different AVSs .
	// In case one server is compromised, signing can continue as expected on the AVSs for which there has been no compromise.
	if k.IsExistPubKey(ctx, bls) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the bls key is already exists:%s", bls.PubKey))
	}
	return k.SetOperatorPubKey(ctx, bls)
}

func (k Keeper) OperatorOptAction(ctx sdk.Context, params *types.OperatorOptParams) error {
	opAccAddr := params.OperatorAddress
	if !k.operatorKeeper.IsOperator(ctx, opAccAddr) {
		return errorsmod.Wrap(delegationtypes.ErrOperatorNotExist, fmt.Sprintf("UpdateAVSInfo: invalid operator address:%s", opAccAddr.String()))
	}

	f, err := k.IsAVS(ctx, params.AvsAddress.String())
	if err != nil {
		return errorsmod.Wrap(err, fmt.Sprintf("error occurred when get avs info,this avs address: %s", params.AvsAddress))
	}
	if !f {
		return fmt.Errorf("avs does not exist,this avs address: %s", params.AvsAddress)
	}

	switch params.Action {
	case types.RegisterAction:
		return k.operatorKeeper.OptIn(ctx, opAccAddr, strings.ToLower(params.AvsAddress.String()))
	case types.DeRegisterAction:
		return k.operatorKeeper.OptOut(ctx, opAccAddr, strings.ToLower(params.AvsAddress.String()))
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
	avsAddrStr := types.GenerateAVSAddress(chainID)
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

func (k Keeper) GetAVSEpochInfo(ctx sdk.Context, addr string) (*epochstypes.EpochInfo, error) {
	avsInfoResp, err := k.GetAVSInfo(ctx, addr)
	if err != nil {
		return nil, err
	}
	avsInfo := avsInfoResp.Info
	// Epoch information must be available because it is checked when setting AVS information.
	// Therefore, we don’t need to check it here.
	epochInfo, _ := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	return &epochInfo, nil
}

func (k Keeper) RaiseAndResolveChallenge(ctx sdk.Context, params *types.ChallengeParams) error {
	taskInfo, err := k.GetTaskInfo(ctx, strconv.FormatUint(params.TaskID, 10), params.TaskContractAddress.String())
	if err != nil || taskInfo == nil {
		return fmt.Errorf("task does not exist,this task address: %s", params.TaskContractAddress)
	}
	// check task isExpected
	if taskInfo.IsExpected {
		return fmt.Errorf("the task has been finished: %s", params.TaskContractAddress)
	}
	// check task OptInOperators count
	if len(taskInfo.OptInOperators) < 1 {
		return fmt.Errorf("this task has no opted-in operators: %s", params.TaskContractAddress)
	}
	// check challenge record
	if k.IsExistTaskChallengedInfo(ctx, params.TaskContractAddress.String(), params.TaskID) {
		return errorsmod.Wrap(types.ErrAlreadyExists, fmt.Sprintf("the challenge has been raised: %s", params.TaskContractAddress))
	}
	// check challenge period
	//  check epoch，The challenge must be within the challenge window period
	avsInfo := k.GetAVSInfoByTaskAddress(ctx, taskInfo.TaskContractAddress)
	if avsInfo.AvsAddress == "" {
		return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the taskaddr is :%s", taskInfo.TaskContractAddress))
	}
	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	if !found {
		return errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s",
			avsInfo.EpochIdentifier))
	}
	// #nosec G115
	if epoch.CurrentEpoch <= int64(taskInfo.StartingEpoch)+int64(taskInfo.TaskResponsePeriod)+int64(taskInfo.TaskStatisticalPeriod) {
		return errorsmod.Wrap(
			types.ErrSubmitTooSoonError,
			fmt.Sprintf("SetTaskResultInfo:the challenge period has not started , CurrentEpoch:%d", epoch.CurrentEpoch),
		)
	}

	err = k.SetTaskChallengedInfo(ctx, params.TaskID, params.CallerAddress.String(), params.TaskContractAddress)
	if err != nil {
		return err
	}

	taskInfo.ActualThreshold = strconv.Itoa(int(params.ActualThreshold))
	taskInfo.IsExpected = uint64(params.ActualThreshold) >= taskInfo.ThresholdPercentage
	taskInfo.EligibleRewardOperators = types.AddressToString(params.EligibleRewardOperators)
	taskInfo.EligibleSlashOperators = types.AddressToString(params.EligibleSlashOperators)
	return k.SetTaskInfo(ctx, taskInfo)
}

func (k *Keeper) GetAllAVSInfos(ctx sdk.Context) ([]types.AVSInfo, error) {
	var ret []types.AVSInfo
	k.IterateAVSInfo(ctx, func(_ int64, avsInfo types.AVSInfo) bool {
		ret = append(ret, avsInfo)
		return false
	})
	return ret, nil
}

func (k Keeper) SubmitTaskResult(ctx sdk.Context, addr string, info *types.TaskResultInfo) error {
	// the operator's `addr` must match the from address.
	if addr != info.OperatorAddress {
		return errorsmod.Wrap(
			types.ErrInvalidAddr,
			"SetTaskResultInfo:from address is not equal to the operator address",
		)
	}
	opAccAddr, _ := sdk.AccAddressFromBech32(info.OperatorAddress)
	// check operator
	if !k.operatorKeeper.IsOperator(ctx, opAccAddr) {
		return errorsmod.Wrap(
			delegationtypes.ErrOperatorNotExist,
			fmt.Sprintf("SetTaskResultInfo:invalid operator address:%s", opAccAddr),
		)
	}
	// check operator bls pubkey
	avsInfo := k.GetAVSInfoByTaskAddress(ctx, info.TaskContractAddress)
	if avsInfo.AvsAddress == "" {
		return errorsmod.Wrap(types.ErrUnregisterNonExistent, fmt.Sprintf("the taskaddr is :%s", info.TaskContractAddress))
	}
	keyInfo, err := k.GetOperatorPubKey(ctx, info.OperatorAddress, avsInfo.AvsAddress)
	if err != nil || keyInfo.PubKey == nil {
		return errorsmod.Wrap(
			types.ErrPubKeyIsNotExists,
			fmt.Sprintf("SetTaskResultInfo:get operator address:%s", opAccAddr),
		)
	}
	pubKey, err := blst.PublicKeyFromBytes(keyInfo.PubKey)
	if err != nil || pubKey == nil {
		return errorsmod.Wrap(
			types.ErrParsePubKey,
			fmt.Sprintf("SetTaskResultInfo:get operator address:%s", opAccAddr),
		)
	}
	//	check task contract
	task, err := k.GetTaskInfo(ctx, strconv.FormatUint(info.TaskId, 10), info.TaskContractAddress)
	if err != nil || task.TaskContractAddress == "" {
		return errorsmod.Wrap(
			types.ErrTaskIsNotExists,
			fmt.Sprintf("SetTaskResultInfo: task info not found: %s (Task ID: %d)",
				info.TaskContractAddress, info.TaskId),
		)
	}

	//  check prescribed period
	//  If submitted in the first phase, in order  to avoid plagiarism by other operators,
	//	TaskResponse and TaskResponseHash must be null values
	//	At the same time, it must be submitted within the response deadline in the first phase
	epoch, found := k.epochsKeeper.GetEpochInfo(ctx, avsInfo.EpochIdentifier)
	if !found {
		return errorsmod.Wrap(types.ErrEpochNotFound, fmt.Sprintf("epoch info not found %s",
			avsInfo.EpochIdentifier))
	}

	switch info.Phase {
	case types.PhasePrepare:
		if k.IsExistTaskResultInfo(ctx, info.OperatorAddress, info.TaskContractAddress, info.TaskId) {
			return errorsmod.Wrap(
				types.ErrResAlreadyExists,
				fmt.Sprintf("SetTaskResultInfo: task result is already exists, "+
					"OperatorAddress: %s (TaskContractAddress: %s),(Task ID: %d)",
					info.OperatorAddress, info.TaskContractAddress, info.TaskId),
			)
		}
		// check parameters
		if info.BlsSignature == nil {
			return errorsmod.Wrap(
				types.ErrParamNotEmptyError,
				fmt.Sprintf("SetTaskResultInfo: invalid param BlsSignature is not be null (BlsSignature: %s)", info.BlsSignature),
			)
		}
		if info.TaskResponseHash != "" || info.TaskResponse != nil {
			return errorsmod.Wrap(
				types.ErrParamNotEmptyError,
				fmt.Sprintf("SetTaskResultInfo: invalid param TaskResponseHash: %s (TaskResponse: %d)",
					info.TaskResponseHash, info.TaskResponse),
			)
		}
		// check epoch，The first phase submission must be within the response window period
		// #nosec G115
		if epoch.CurrentEpoch > int64(task.StartingEpoch)+int64(task.TaskResponsePeriod) {
			return errorsmod.Wrap(
				types.ErrSubmitTooLateError,
				fmt.Sprintf("SetTaskResultInfo:submit  too late, CurrentEpoch:%d", epoch.CurrentEpoch),
			)
		}
		k.SetTaskResultInfo(ctx, info)
		return nil

	case types.PhaseDoCommit:
		// check task response
		if info.TaskResponse == nil {
			return errorsmod.Wrap(
				types.ErrNotNull,
				fmt.Sprintf("SetTaskResultInfo: invalid param  (TaskResponse: %d)",
					info.TaskResponse),
			)
		}
		// check parameters
		res, err := k.GetTaskResultInfo(ctx, info.OperatorAddress, info.TaskContractAddress, info.TaskId)
		if err != nil || !bytes.Equal(res.BlsSignature, info.BlsSignature) {
			return errorsmod.Wrap(
				types.ErrInconsistentParams,
				fmt.Sprintf("SetTaskResultInfo: invalid param OperatorAddress: %s ,(TaskContractAddress: %s)"+
					",(TaskID: %d),(BlsSignature: %s)",
					info.OperatorAddress, info.TaskContractAddress, info.TaskId, info.BlsSignature),
			)
		}
		//  check epoch，The second phase submission must be within the statistical window period
		// #nosec G115
		if epoch.CurrentEpoch <= int64(task.StartingEpoch)+int64(task.TaskResponsePeriod) {
			return errorsmod.Wrap(
				types.ErrSubmitTooSoonError,
				fmt.Sprintf("SetTaskResultInfo:the TaskResponse period has not started , CurrentEpoch:%d", epoch.CurrentEpoch),
			)
		}
		// #nosec G115
		if epoch.CurrentEpoch > int64(task.StartingEpoch)+int64(task.TaskResponsePeriod)+int64(task.TaskStatisticalPeriod) {
			return errorsmod.Wrap(
				types.ErrSubmitTooLateError,
				fmt.Sprintf("SetTaskResultInfo:submit  too late, CurrentEpoch:%d", epoch.CurrentEpoch),
			)
		}

		// calculate hash by original task
		taskResponseDigest := crypto.Keccak256Hash(info.TaskResponse)
		info.TaskResponseHash = taskResponseDigest.String()
		// check bls sig
		flag, err := blst.VerifySignature(info.BlsSignature, taskResponseDigest, pubKey)
		if !flag || err != nil {
			return errorsmod.Wrap(
				types.ErrSigVerifyError,
				fmt.Sprintf("SetTaskResultInfo: invalid task address: %s (Task ID: %d)", info.TaskContractAddress, info.TaskId),
			)
		}

		k.SetTaskResultInfo(ctx, info)
		return nil
	default:
		return errorsmod.Wrap(
			types.ErrParamError,
			fmt.Sprintf("SetTaskResultInfo: invalid param value:%d", info.Phase),
		)
	}
}
