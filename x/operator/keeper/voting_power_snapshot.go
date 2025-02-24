package keeper

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	epochtypes "github.com/imua-xyz/imuachain/x/epochs/types"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

const InitialEpochNumber int64 = 1

func (k *Keeper) SetVotingPowerSnapshot(ctx sdk.Context, key []byte, snapshot *types.VotingPowerSnapshot) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	bz := k.cdc.MustMarshal(snapshot)
	store.Set(key, bz)
	return nil
}

func (k *Keeper) GetVotingPowerSnapshot(ctx sdk.Context, key []byte) (*types.VotingPowerSnapshot, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	var ret types.VotingPowerSnapshot
	value := store.Get(key)
	if value == nil {
		avsAddr, height, err := types.ParseVotingPowerSnapshotKey(key)
		if err != nil {
			return nil, err
		}
		return nil, types.ErrNoKeyInTheStore.Wrapf("GetVotingPowerSnapshot: invalid key, avs:%s height:%v", avsAddr, height)
	}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k *Keeper) IterateVotingPowerSnapshot(ctx sdk.Context, avsAddr string, isUpdate bool, opFunc func(height int64, snapshot *types.VotingPowerSnapshot) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	iterator := sdk.KVStorePrefixIterator(store, common.HexToAddress(avsAddr).Bytes())
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.VotingPowerSnapshot
		k.cdc.MustUnmarshal(iterator.Value(), &snapshot)
		_, height, err := types.ParseVotingPowerSnapshotKey(iterator.Key())
		if err != nil {
			return err
		}
		err = opFunc(height, &snapshot)
		if err != nil {
			return err
		}
		if isUpdate {
			bz := k.cdc.MustMarshal(&snapshot)
			store.Set(iterator.Key(), bz)
		}
	}
	return nil
}

func (k *Keeper) GetSnapshotHeightAndKey(ctx sdk.Context, avsAddr string, height int64) (int64, []byte, error) {
	if !common.IsHexAddress(avsAddr) {
		return 0, nil, types.ErrParameterInvalid.Wrapf("invalid AVS address format: %s", avsAddr)
	}
	if height < 0 {
		return 0, nil, types.ErrParameterInvalid.Wrapf("the input height is negative, height:%v", height)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	// If there is no snapshot for the input height, we need to find the correct key.
	// The snapshot closest to the input height is the one used for its voting
	// power information. The correct snapshot key can be found by taking advantage
	// of the ascending order of data returned when using an iterator range.
	avsEthAddr := common.HexToAddress(avsAddr)
	findKey := types.KeyForVotingPowerSnapshot(avsEthAddr, height)
	findHeight := height
	if !store.Has(findKey) {
		iterator := sdk.KVStorePrefixIterator(store, avsEthAddr.Bytes())
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			_, keyHeight, err := types.ParseVotingPowerSnapshotKey(iterator.Key())
			if err != nil {
				return 0, nil, err
			}
			if keyHeight <= height {
				findKey = iterator.Key()
				findHeight = keyHeight
			} else {
				break
			}
		}
	}
	return findHeight, findKey, nil
}

func (k *Keeper) GetEpochNumberByOptOutHeight(ctx sdk.Context, avsAddr string, optOutHeight int64) (int64, error) {
	if optOutHeight < 0 {
		return 0, types.ErrParameterInvalid.Wrapf("the opt out height is negative, optOutHeight:%v", optOutHeight)
	}
	_, findKey, err := k.GetSnapshotHeightAndKey(ctx, avsAddr, optOutHeight)
	if err != nil {
		return 0, err
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	value := store.Get(findKey)
	if value == nil {
		ctx.Logger().Info("GetEpochNumberByOptOutHeight: can't find the epoch number of optOutHeight", "avs", avsAddr, "optOutHeight", optOutHeight)
		// We don't save the voting power snapshots for the AVSs that don't have any opted-in operators.
		// Therefore, for these AVSs, the `findKey` might not exist in the store if the operator opts in and out
		// within the same epoch. In this case, we return `NullEpochNumber` as the virtual epoch number, so that
		// the caller `GetImpactfulAVSForOperator` can skip this AVS. This is acceptable because opt-in and opt-out
		// actions submitted within the same epoch won't influence the AVS, and the AVS doesn't have any operators
		// opted in.
		// Additionally, since expired voting power snapshots are deleted, it will also be impossible to retrieve
		// the corresponding epochNumber when the opt-out height is too old. In such cases, the same handling method
		// is applied.
		return epochtypes.NullEpochNumber, nil
	}
	var ret types.VotingPowerSnapshot
	k.cdc.MustUnmarshal(value, &ret)
	return ret.EpochNumber, nil
}

// LoadVotingPowerSnapshot loads the voting power snapshot information for the provided height,
// returning the height of the first block in the epoch the snapshot serves, along with the specific
// voting power data. The start height will be used to filter pending undelegations during slashing.
func (k *Keeper) LoadVotingPowerSnapshot(ctx sdk.Context, avsAddr string, height int64) (int64, *types.VotingPowerSnapshot, error) {
	findHeight, findKey, err := k.GetSnapshotHeightAndKey(ctx, avsAddr, height)
	if err != nil {
		return 0, nil, err
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	value := store.Get(findKey)
	if value == nil {
		avs, keyHeight, _ := types.ParseVotingPowerSnapshotKey(findKey)
		return 0, nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: findHeight is %v, avs:%s ,keyHeight:%v", findHeight, avs, keyHeight)
	}
	var ret types.VotingPowerSnapshot
	k.cdc.MustUnmarshal(value, &ret)

	// fall back to get the snapshot if the key height doesn't equal to the `LastChangedHeight`
	if findHeight != ret.LastChangedHeight {
		value = store.Get(types.KeyForVotingPowerSnapshot(common.HexToAddress(avsAddr), ret.LastChangedHeight))
		if value == nil {
			return 0, nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: fall back to the height %v", ret.LastChangedHeight)
		}
		k.cdc.MustUnmarshal(value, &ret)
	}
	return findHeight, &ret, nil
}

// RemoveVotingPowerSnapshot remove all snapshots older than the input epoch number.
func (k *Keeper) RemoveVotingPowerSnapshot(ctx sdk.Context, avsAddr string, epochNumber int64) error {
	if epochNumber < 0 {
		return types.ErrParameterInvalid.Wrapf("the input epoch number is negative, epochNumber:%v", epochNumber)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	iterator := sdk.KVStorePrefixIterator(store, common.HexToAddress(avsAddr).Bytes())
	defer iterator.Close()
	// the retained key is used to record the snapshot that will be fallen back to
	// by snapshots earlier than the input time.
	var retainedKey []byte
	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.VotingPowerSnapshot
		k.cdc.MustUnmarshal(iterator.Value(), &snapshot)
		_, height, _ := types.ParseVotingPowerSnapshotKey(iterator.Key())
		if snapshot.EpochNumber > epochNumber {
			// delete the retained key, because the snapshots that is earlier than the input time
			// don't need to retain any old snapshot key.
			if height == snapshot.LastChangedHeight && retainedKey != nil {
				store.Delete(retainedKey)
			}
			break
		}
		// When height == snapshot.LastChangedHeight, it indicates that the current snapshot
		// contains the current voting power set, so there is no need to fall back to other keys.
		if height == snapshot.LastChangedHeight {
			// delete the old retained key, because the key currently holding the voting power set
			// will become the latest retained key.
			if retainedKey != nil {
				store.Delete(retainedKey)
			}
			retainedKey = iterator.Key()
		} else {
			store.Delete(iterator.Key())
		}
	}
	return nil
}

func (k *Keeper) UpdateSnapshotHelper(ctx sdk.Context, avsAddr string, opFunc func(helper *types.SnapshotHelper) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	var snapshotHelper types.SnapshotHelper
	value := store.Get([]byte(avsAddr))
	if value != nil {
		k.cdc.MustUnmarshal(value, &snapshotHelper)
	}
	err := opFunc(&snapshotHelper)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&snapshotHelper)
	store.Set([]byte(avsAddr), bz)
	return nil
}

func (k *Keeper) SetOptOutFlag(ctx sdk.Context, avsAddr string, hasOptOut bool) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.HasOptOut = hasOptOut
		return nil // Reserve for future error handling
	}
	return k.UpdateSnapshotHelper(ctx, avsAddr, opFunc)
}

func (k *Keeper) SetLastChangedHeight(ctx sdk.Context, avsAddr string, lastChangeHeight int64) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.LastChangedHeight = lastChangeHeight
		return nil // Reserve for future error handling
	}
	return k.UpdateSnapshotHelper(ctx, avsAddr, opFunc)
}

func (k *Keeper) SetSnapshotHelper(ctx sdk.Context, avsAddr string, snapshotHelper types.SnapshotHelper) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	bz := k.cdc.MustMarshal(&snapshotHelper)
	store.Set([]byte(avsAddr), bz)
	return nil
}

func (k *Keeper) GetSnapshotHelper(ctx sdk.Context, avsAddr string) (types.SnapshotHelper, error) {
	var ret types.SnapshotHelper
	if !common.IsHexAddress(avsAddr) {
		return ret, types.ErrParameterInvalid.Wrapf("invalid AVS address format: %s", avsAddr)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	value := store.Get([]byte(strings.ToLower(avsAddr)))
	if value == nil {
		return ret, types.ErrNoKeyInTheStore.Wrapf("GetSnapshotHelper: the key is %s", avsAddr)
	}
	k.cdc.MustUnmarshal(value, &ret)
	return ret, nil
}

func (k *Keeper) HasSnapshotHelper(ctx sdk.Context, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	return store.Has([]byte(avsAddr))
}

func (k *Keeper) InitGenesisVPSnapshot(ctx sdk.Context) error {
	snapshotStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	snapshotHelperStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	opFunc := func(avsAddr string, avsUSDValue *types.DecValueField) error {
		votingPowerSet := make([]*types.OperatorVotingPower, 0)
		opFunc := func(operator string, optedUSDValues *types.OperatorOptedUSDValue) error {
			if optedUSDValues.ActiveUSDValue.IsPositive() {
				votingPowerSet = append(votingPowerSet, &types.OperatorVotingPower{
					OperatorAddr: operator,
					VotingPower:  optedUSDValues.ActiveUSDValue,
				})
			}
			return nil
		}
		err := k.IterateOperatorsForAVS(ctx, avsAddr, false, opFunc)
		if err != nil {
			return err
		}
		epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
		if err != nil {
			return err
		}
		genesisSnapshotHeight := ctx.BlockHeight() + 1
		epochNumber := epochInfo.CurrentEpoch
		// set the epoch number to 1 when epoch start for the first time.
		if !epochInfo.EpochCountingStarted {
			epochNumber = InitialEpochNumber
		}
		bz := k.cdc.MustMarshal(&types.VotingPowerSnapshot{
			TotalVotingPower:     avsUSDValue.Amount,
			OperatorVotingPowers: votingPowerSet,
			LastChangedHeight:    genesisSnapshotHeight,
			EpochIdentifier:      epochInfo.Identifier,
			EpochNumber:          epochNumber,
		})
		snapshotKey := types.KeyForVotingPowerSnapshot(common.HexToAddress(avsAddr), genesisSnapshotHeight)
		snapshotStore.Set(snapshotKey, bz)

		bz = k.cdc.MustMarshal(&types.SnapshotHelper{
			LastChangedHeight: genesisSnapshotHeight,
			HasOptOut:         false,
		})
		snapshotHelperStore.Set([]byte(avsAddr), bz)
		return nil
	}
	err := k.IterateAVSUSDValues(ctx, false, opFunc)
	if err != nil {
		return err
	}
	return nil
}
