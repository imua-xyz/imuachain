package keeper

import (
	"sort"
	"time"

	"cosmossdk.io/math"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ExocoreNetwork/exocore/x/dogfood/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// UnbondingTime returns the time duration of the unbonding period. It is part of the
// implementation of the staking keeper expected by IBC.
// It is calculated as the number of epochs until unbonded multiplied by the duration of an
// epoch. This function is used by IBC's client keeper to validate the self client, and
// nowhere else. As long as it reports a consistent value, it's fine.
func (k Keeper) UnbondingTime(ctx sdk.Context) time.Duration {
	params := k.GetDogfoodParams(ctx)
	// no need to check for found, as the epoch info is validated at genesis.
	epoch, _ := k.epochsKeeper.GetEpochInfo(ctx, params.EpochIdentifier)
	durationPerEpoch := epoch.Duration
	// the extra 1 is added to account for the current epoch. this is,
	// therefore, the maximum time it takes for unbonding. if the tx
	// is sent towards the end of the current epoch, the actual time
	// will be closer to 7 days; if it is sent towards the beginning,
	// the actual time will be closer to 8 days.
	return time.Duration(params.EpochsUntilUnbonded+1) * durationPerEpoch
}

// ApplyValidatorChanges returns the validator set as is. However, it also
// stores the validators that are added or those that are removed, and updates
// the stored power for the existing validators. It also allows any hooks registered
// on the keeper to be executed. Lastly, it stores the validator set against the
// provided validator set id. The caller must ensure that `change.Key` is not nil.
func (k Keeper) ApplyValidatorChanges(
	ctx sdk.Context, changes []keytypes.WrappedConsKeyWithPower,
) []abci.ValidatorUpdate {
	ret := []abci.ValidatorUpdate{}
	logger := k.Logger(ctx)
	for _, change := range changes {
		addr := change.Key.ToConsAddr()
		val, found := k.GetExocoreValidator(ctx, addr)
		switch found {
		case true:
			// update or delete an existing validator.
			// assumption: power can not be negative.
			if change.Power < 1 {
				// guard for errors within the hooks.
				cc, writeFunc := ctx.CacheContext()
				k.DeleteExocoreValidator(cc, addr)
				// sdk slashing.AfterValidatorRemoved deletes the lookup from cons address to
				// cons pub key
				if err := k.Hooks().AfterValidatorRemoved(cc, addr, nil); err != nil {
					logger.Error("error in AfterValidatorRemoved", "error", err)
					continue
				}
				writeFunc()
			} else {
				val.Power = change.Power
				// guard for errors within the hooks.
				cc, writeFunc := ctx.CacheContext()
				k.SetExocoreValidator(ctx, val)
				// sdk slashing.AfterValidatorCreated stores the lookup from cons address to
				// cons pub key. it loads the validator from `valAddr` (operator address)
				// via stakingkeeeper.Validator(ctx, valAddr)
				// then it fetches the cons pub key from said validator to generate the lookup
				found, accAddress := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
					ctx, avstypes.ChainIDWithoutRevision(ctx.ChainID()), addr,
				)
				if !found {
					// should never happen
					logger.Error("operator address not found for validator", "cons address", addr)
					continue
				}
				if err := k.Hooks().AfterValidatorCreated(
					cc, sdk.ValAddress(accAddress),
				); err != nil {
					logger.Error("error in AfterValidatorCreated", "error", err)
					continue
				}
				writeFunc()
			}
		case false:
			if change.Power > 0 {
				// create a new validator.
				ocVal, err := types.NewExocoreValidator(addr, change.Power, change.Key.ToSdkKey())
				if err != nil {
					logger.Error("could not create new exocore validator", "error", err)
					continue
				}
				// guard for errors within the hooks.
				cc, writeFunc := ctx.CacheContext()
				k.SetExocoreValidator(cc, ocVal)
				err = k.Hooks().AfterValidatorBonded(cc, addr, nil)
				if err != nil {
					logger.Error("error in AfterValidatorBonded", "error", err)
					// If an error is returned, the validator is not added to the `ret` slice.
					continue
				}
				writeFunc()
			} else {
				// edge case: we received an update for 0 power
				// but the validator is already deleted. Do not forward
				// to tendermint.
				logger.Info("received update for non-existent validator", "cons address", addr)
				continue
			}
		}
		ret = append(ret, abci.ValidatorUpdate{
			PubKey: *change.Key.ToTmProtoKey(),
			Power:  change.Power,
		})
	}

	// sort for determinism
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Power != ret[j].Power {
			return ret[i].Power > ret[j].Power
		}
		return ret[i].PubKey.String() > ret[j].PubKey.String()
	})

	// set the list of validator updates
	k.SetValidatorUpdates(ctx, ret)

	return ret
}

// SetExocoreValidator stores a validator based on the pub key derived address. This
// is accessible in the genesis state via `val_set`.
func (k Keeper) SetExocoreValidator(ctx sdk.Context, validator types.ExocoreValidator) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&validator)

	store.Set(types.ExocoreValidatorKey(validator.Address), bz)
}

// GetExocoreValidator gets a validator based on the pub key derived (consensus) address.
func (k Keeper) GetExocoreValidator(
	ctx sdk.Context, addr sdk.ConsAddress,
) (validator types.ExocoreValidator, found bool) {
	store := ctx.KVStore(k.storeKey)
	v := store.Get(types.ExocoreValidatorKey(addr.Bytes()))
	if v == nil {
		return
	}
	k.cdc.MustUnmarshal(v, &validator)
	found = true

	return
}

// IsExocoreValidator gets a validator based on the pub key derived (consensus) address.
func (k Keeper) IsExocoreValidator(
	ctx sdk.Context, addr sdk.ConsAddress,
) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.ExocoreValidatorKey(addr.Bytes()))
}

// DeleteExocoreValidator deletes a validator based on the pub key derived address.
func (k Keeper) DeleteExocoreValidator(ctx sdk.Context, addr sdk.ConsAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ExocoreValidatorKey(addr.Bytes()))
}

// GetAllExocoreValidators returns all validators in the store.
func (k Keeper) GetAllExocoreValidators(
	ctx sdk.Context,
) (validators []types.ExocoreValidator) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{types.ExocoreValidatorBytePrefix})

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		val := types.ExocoreValidator{}
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		validators = append(validators, val)
	}

	return validators
}

// GetHistoricalInfo gets the historical info at a given height
func (k Keeper) GetHistoricalInfo(
	ctx sdk.Context, height int64,
) (stakingtypes.HistoricalInfo, bool) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.HistoricalInfoKey(height)

	value := store.Get(key)
	if value == nil {
		return stakingtypes.HistoricalInfo{}, false
	}

	return stakingtypes.MustUnmarshalHistoricalInfo(k.cdc, value), true
}

// SetHistoricalInfo sets the historical info at a given height
func (k Keeper) SetHistoricalInfo(
	ctx sdk.Context, height int64, hi *stakingtypes.HistoricalInfo,
) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.HistoricalInfoKey(height)
	value := k.cdc.MustMarshal(hi)

	store.Set(key, value)
}

// DeleteHistoricalInfo deletes the historical info at a given height
func (k Keeper) DeleteHistoricalInfo(ctx sdk.Context, height int64) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.HistoricalInfoKey(height)

	store.Delete(key)
}

// TrackHistoricalInfo saves the latest historical info and deletes the ones eligible to be
// pruned. The function is called within the BeginBlock of the module, so it is kept public.
// It is mostly a copy of the function used by interchain-security.
// If the historical info were only used by IBC, this function would store a subset of the
// header for each block, since only those parts were used.
// However, the historical info is used by the EVM keeper as well, which hashes the full header
// to report it via Solidity to the caller. Therefore, the full header must be stored.
func (k Keeper) TrackHistoricalInfo(ctx sdk.Context) {
	// Get the number of historical entries to persist, as the number of block heights.
	// #nosec G701 // uint32 fits into int64 always.
	numHistoricalEntries := int64(
		k.GetHistoricalEntries(ctx),
	)

	// we are deleting headers, say, from, 0 to 999 at block 1999
	for i := ctx.BlockHeight() - numHistoricalEntries; i >= 0; i-- {
		_, found := k.GetHistoricalInfo(ctx, i)
		if found {
			k.DeleteHistoricalInfo(ctx, i)
		} else {
			break
		}
	}

	// if there is no need to persist historicalInfo, return.
	if numHistoricalEntries == 0 {
		return
	}

	// Create HistoricalInfo struct
	lastVals := []stakingtypes.Validator{}
	for _, v := range k.GetAllExocoreValidators(ctx) {
		pk, err := v.ConsPubKey()
		if err != nil {
			// since we stored the validator in the first place, something like this
			// should never happen, but if it does it is an extremely grave error
			// that will result in a block mismatch and hence that node will halt.
			continue
		}
		val, err := stakingtypes.NewValidator(nil, pk, stakingtypes.Description{})
		if err != nil {
			// same as above.
			continue
		}

		// Set validator to bonded status
		val.Status = stakingtypes.Bonded
		// Compute tokens from voting power
		val.Tokens = sdk.TokensFromConsensusPower(v.Power, sdk.DefaultPowerReduction)
		lastVals = append(lastVals, val)
	}

	// Create historical info entry which sorts the validator set by voting power
	historicalEntry := stakingtypes.NewHistoricalInfo(
		ctx.BlockHeader(), lastVals, sdk.DefaultPowerReduction,
	)

	// Set latest HistoricalInfo at current height
	k.SetHistoricalInfo(ctx, ctx.BlockHeight(), &historicalEntry)
}

// MustGetCurrentValidatorsAsABCIUpdates gets all validators converted
// to the ABCI validator update type. It panics in case of failure.
func (k Keeper) MustGetCurrentValidatorsAsABCIUpdates(ctx sdk.Context) []abci.ValidatorUpdate {
	vals := k.GetAllExocoreValidators(ctx)
	valUpdates := make([]abci.ValidatorUpdate, 0, len(vals))
	for _, v := range vals {
		pk, err := v.ConsPubKey()
		if err != nil {
			// This should never happen as the pubkey is assumed
			// to be stored correctly earlier.
			panic(err)
		}
		tmPK, err := cryptocodec.ToTmProtoPublicKey(pk)
		if err != nil {
			// This should never happen as the pubkey is assumed
			// to be stored correctly earlier.
			panic(err)
		}
		valUpdates = append(valUpdates, abci.ValidatorUpdate{PubKey: tmPK, Power: v.Power})
	}

	return valUpdates
}

// GetLastTotalPower gets the last total validator power.
func (k Keeper) GetLastTotalPower(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastTotalPowerKey())
	if bz == nil {
		return math.ZeroInt()
	}
	ip := sdk.IntProto{}
	k.cdc.MustUnmarshal(bz, &ip)
	return ip.Int
}

// SetLastTotalPower sets the last total validator power.
func (k Keeper) SetLastTotalPower(ctx sdk.Context, power math.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: power})
	store.Set(types.LastTotalPowerKey(), bz)
}

// SetValidatorUpdates sets the ABCI validator power updates for the current block.
func (k Keeper) SetValidatorUpdates(ctx sdk.Context, valUpdates []abci.ValidatorUpdate) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&stakingtypes.ValidatorUpdates{Updates: valUpdates})
	store.Set(types.ValidatorUpdatesKey(), bz)
}

// GetValidatorUpdates returns the ABCI validator power updates within the current block.
func (k Keeper) GetValidatorUpdates(ctx sdk.Context) []abci.ValidatorUpdate {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ValidatorUpdatesKey())

	var valUpdates stakingtypes.ValidatorUpdates
	k.cdc.MustUnmarshal(bz, &valUpdates)

	return valUpdates.Updates
}

// GetValidator fetchs a stakingtypes.Validator given the validator's address.
// This is just the account address, but sent as sdk.ValAddress(accAddr).
func (k Keeper) GetValidator(
	ctx sdk.Context, valAddr sdk.ValAddress,
) (stakingtypes.Validator, bool) {
	accAddr := sdk.AccAddress(valAddr)
	found, wrappedKey, err := k.operatorKeeper.GetOperatorConsKeyForChainID(
		ctx, accAddr, avstypes.ChainIDWithoutRevision(ctx.ChainID()),
	)
	if !found || err != nil || wrappedKey == nil {
		return stakingtypes.Validator{}, false
	}
	val, found := k.operatorKeeper.ValidatorByConsAddrForChainID(
		ctx, wrappedKey.ToConsAddr(), avstypes.ChainIDWithoutRevision(ctx.ChainID()),
	)
	if !found {
		return stakingtypes.Validator{}, false
	}
	// the bonded status of the validator is unspecified, since we don't know if it is
	// actually in this module or not. we are not checking it either, since there is no
	// particular reason to do so.
	return val, true
}
