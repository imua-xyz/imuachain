package keeper

import (
	"fmt"
	"math/big"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v16/x/evm/statedb"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetAVSSupportedAssets returns a map of assets supported by the AVS. The avsAddr supplied must be hex.
func (k *Keeper) GetAVSSupportedAssets(ctx sdk.Context, avsAddr string) (map[string]interface{}, error) {
	if !common.IsHexAddress(avsAddr) {
		return nil, errorsmod.Wrap(types.ErrInvalidAddr, fmt.Sprintf("GetAVSSupportedAssets: key is %s", avsAddr))
	}
	avsInfo, err := k.GetAVSInfo(ctx, avsAddr)
	if err != nil {
		return nil, errorsmod.Wrap(err, fmt.Sprintf("GetAVSSupportedAssets: key is %s", avsAddr))
	}
	assetIDList := avsInfo.Info.AssetIDs
	ret := make(map[string]interface{})

	for _, assetID := range assetIDList {
		asset, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
		if err != nil {
			return nil, errorsmod.Wrap(err, fmt.Sprintf("[GetAVSSupportedAssets] GetStakingAssetInfo: key is %s", assetID))
		}
		ret[assetID] = asset
	}

	return ret, nil
}

// GetAVSSlashContract returns the address of the contract that will be used to slash the AVS.
// The avsAddr supplied must be hex.
func (k *Keeper) GetAVSSlashContract(ctx sdk.Context, avsAddr string) (string, error) {
	avsInfo, err := k.GetAVSInfo(ctx, avsAddr)
	if err != nil {
		return "", errorsmod.Wrap(err, fmt.Sprintf("GetAVSSlashContract: key is %s", avsAddr))
	}

	return avsInfo.Info.SlashAddr, nil
}

// GetAVSMinimumSelfDelegation returns the minimum self-delegation required for the AVS, on a per-operator basis.
// The avsAddr supplied must be hex.
func (k *Keeper) GetAVSMinimumSelfDelegation(ctx sdk.Context, avsAddr string) (sdkmath.LegacyDec, error) {
	avsInfo, err := k.GetAVSInfo(ctx, avsAddr)
	if err != nil {
		return sdkmath.LegacyNewDec(0), errorsmod.Wrap(err, fmt.Sprintf("GetAVSMinimumSelfDelegation: key is %s", avsAddr))
	}
	// #nosec G115
	return sdkmath.LegacyNewDec(int64(avsInfo.Info.MinSelfDelegation)), nil
}

// GetEpochEndAVSs returns a list of hex AVS addresses for AVSs which are scheduled to start at the end of the
// current epoch, or the beginning of the next one. The address format returned is hex.
func (k *Keeper) GetEpochEndAVSs(ctx sdk.Context, epochIdentifier string, endingEpochNumber int64) []string {
	var avsList []string
	k.IterateAVSInfo(ctx, func(_ int64, avsInfo types.AVSInfo) (stop bool) {
		// consider the dogfood AVS as an example. it is scheduled to start at epoch 0.
		// the currentEpoch is 1, so we will return it.
		// consider another AVS which will start at epoch 5. the current epoch is 4.
		// it should be returned here, since the operator module should start tracking this.
		// #nosec G115
		if epochIdentifier == avsInfo.EpochIdentifier && endingEpochNumber >= int64(avsInfo.StartingEpoch)-1 {
			avsList = append(avsList, avsInfo.AvsAddress)
		}
		return false
	})

	return avsList
}

// GetEpochEndChainIDs returns a list of chainIDs for AVSs which are scheduled to start at the end of the
// current epoch, or the beginning of the next one. The chainIDs used are without the revision number.
func (k Keeper) GetEpochEndChainIDs(ctx sdk.Context, epochIdentifier string, endingEpochNumber int64) []string {
	avsList := k.GetEpochEndAVSs(ctx, epochIdentifier, endingEpochNumber)
	var chainIDs []string
	for _, avsAddr := range avsList {
		chainID, found := k.GetChainIDByAVSAddr(ctx, avsAddr)
		if found {
			chainIDs = append(chainIDs, chainID)
		}
	}
	return chainIDs
}

// GetAVSInfoByTaskAddress returns the AVS  which containing this task address
// A task contract address can only be used by one avs
// TODO:this function is frequently used while its implementation iterates over existing avs to find the target avs by task contract address,  we should use a reverse mapping to avoid iteration
func (k *Keeper) GetAVSInfoByTaskAddress(ctx sdk.Context, taskAddr string) types.AVSInfo {
	var avs types.AVSInfo
	if taskAddr == "" {
		return avs
	}
	k.IterateAVSInfo(ctx, func(_ int64, avsInfo types.AVSInfo) (stop bool) {
		if taskAddr == avsInfo.GetTaskAddr() {
			avs = avsInfo
			return true // stop because we found the AVS
		}
		return false
	})
	return avs
}

// GetTaskStatisticalEpochEndAVSs returns the task list where the current block marks the end of their statistical period.
func (k *Keeper) GetTaskStatisticalEpochEndAVSs(ctx sdk.Context, epochIdentifier string, epochNumber int64) []types.TaskResultInfo {
	var taskResList []types.TaskResultInfo
	k.IterateResultInfo(ctx, func(_ int64, info types.TaskResultInfo) (stop bool) {
		avsInfo := k.GetAVSInfoByTaskAddress(ctx, info.TaskContractAddress)
		taskInfo, err := k.GetTaskInfo(ctx, strconv.FormatUint(info.TaskId, 10), info.TaskContractAddress)
		if err != nil {
			return false
		}
		// Determine if the statistical period has passed, the range of the statistical period is the num marked (StartingEpoch) add TaskStatisticalPeriod
		// #nosec G115
		if epochIdentifier == avsInfo.EpochIdentifier && epochNumber ==
			// #nosec G115
			int64(taskInfo.StartingEpoch)+int64(taskInfo.TaskResponsePeriod)+int64(taskInfo.TaskStatisticalPeriod) {
			taskResList = append(taskResList, info)
		}
		return false
	})
	return taskResList
}

// RegisterAVSWithChainID registers an AVS by its chainID.
// It is responsible for generating an AVS address based on the chainID.
// The following bare minimum parameters must be supplied:
// AssetIDs, EpochsUntilUnbonded, EpochIdentifier, MinSelfDelegation and StartingEpoch.
// This will ensure compatibility with all of the related AVS functions, like
// GetEpochEndAVSs, GetAVSSupportedAssets, and GetAVSMinimumSelfDelegation.
// The caller must use a chainID without the revision number if the AVS is intended
// to outlive the upgrades, for example, the x/dogfood AVS. The same cannot be said
// for x/appchain AVSs wherein the post-upgrade operators may not be the same.
func (k Keeper) RegisterAVSWithChainID(
	oCtx sdk.Context, params *types.AVSRegisterOrDeregisterParams,
) (avsAddr common.Address, err error) {
	// guard against errors
	ctx, writeFunc := oCtx.CacheContext()
	// remove the version number and validate
	if len(params.ChainID) == 0 {
		return common.Address{}, errorsmod.Wrap(types.ErrNotNull, "RegisterAVSWithChainID: chainID is null")
	}
	avsAddrStr := types.GenerateAVSAddr(params.ChainID)
	avsAddr = common.HexToAddress(avsAddrStr)
	defer func() {
		if err == nil {
			// store the reverse lookup from AVSAddress to ChainID
			// (the forward can be generated on the fly by hashing).
			k.SetAVSAddrToChainID(ctx, avsAddr, params.ChainID)
			// write the cache
			writeFunc()
			// TODO: do events need to be handled separately? currently no events emitted so not urgent.
		}
	}()
	// Mark the account as occupied by a contract, so that any transactions that originate
	// from it are rejected in `state_transition.go`. This protects us against the very
	// rare case of address collision.
	if err := k.evmKeeper.SetAccount(
		ctx, avsAddr,
		statedb.Account{
			Balance:  big.NewInt(0),
			CodeHash: types.ChainIDCodeHash[:],
			Nonce:    1,
		},
	); err != nil {
		return common.Address{}, err
	}
	// SetAVSInfo expects HexAddress for the AvsAddress
	params.AvsAddress = avsAddrStr
	params.Action = RegisterAction

	if err := k.UpdateAVSInfo(ctx, params); err != nil {
		return common.Address{}, err
	}
	return avsAddr, nil
}

// SetAVSAddressToChainID stores a lookup from the generated AVS address to the chainID.
func (k Keeper) SetAVSAddrToChainID(ctx sdk.Context, avsAddr common.Address, chainID string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSAddressToChainID)
	store.Set(avsAddr[:], []byte(chainID))
}

// GetChainIDByAVSAddr returns the chainID for a given AVS address. It is a stateful
// function since it only returns the chainID if an AVS with the chainID was previously
// registered.
func (k Keeper) GetChainIDByAVSAddr(ctx sdk.Context, avsAddr string) (string, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSAddressToChainID)
	bz := store.Get(common.HexToAddress(avsAddr).Bytes())
	if bz == nil {
		return "", false
	}
	return string(bz), true
}
