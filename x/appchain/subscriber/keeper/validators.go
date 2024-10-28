package keeper

import (
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	types "github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SetValsetUpdateIDForHeight sets the valset update ID for a given height
func (k Keeper) SetValsetUpdateIDForHeight(
	ctx sdk.Context, height int64, valsetUpdateID uint64,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ValsetUpdateIDKey(height), sdk.Uint64ToBigEndian(valsetUpdateID))
}

// GetValsetUpdateIDForHeight gets the valset update ID for a given height
func (k Keeper) GetValsetUpdateIDForHeight(
	ctx sdk.Context, height int64,
) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.ValsetUpdateIDKey(height)
	if !store.Has(key) {
		return 0
	}
	bz := store.Get(key)
	return sdk.BigEndianToUint64(bz)
}

// ApplyValidatorChanges is a wrapper function that returns the provided validator set
// update. The wrapping allows to save the validator set information in the store.
// The caller should (but _not_ must) provide `changes` that are different from the
// ones already with Tendermint.
func (k Keeper) ApplyValidatorChanges(
	ctx sdk.Context,
	// in dogfood, we use the wrappedkeywithpower because the operator module provides
	// keys in that format. since the subscriber chain doesn't need the operator module
	// we can use the tm validator update type.
	changes []abci.ValidatorUpdate,
) []abci.ValidatorUpdate {
	ret := make([]abci.ValidatorUpdate, 0, len(changes))
	logger := k.Logger(ctx)
	for i := range changes {
		change := changes[i] // avoid implicit memory aliasing
		wrappedKey := keytypes.NewWrappedConsKeyFromTmProtoKey(&change.PubKey)
		if wrappedKey == nil {
			// an error in deserializing the key would indicate that the coordinator
			// has provided invalid data. this is a critical error and should be
			// investigated.
			logger.Error(
				"failed to deserialize validator key",
				"i", i, "validator", change.PubKey,
			)
			continue
		}
		consAddress := wrappedKey.ToConsAddr()
		val, found := k.GetSubscriberValidator(ctx, consAddress)
		switch found {
		case true:
			if change.Power < 1 {
				logger.Info("deleting validator", "consAddress", consAddress)
				k.DeleteSubscriberValidator(ctx, consAddress)
			} else {
				logger.Info("updating validator", "consAddress", consAddress)
				val.Power = change.Power
				k.SetSubscriberValidator(ctx, val)
			}
		case false:
			if change.Power > 0 {
				cc, writeFunc := ctx.CacheContext()
				ocVal, err := commontypes.NewSubscriberValidator(
					consAddress, change.Power, wrappedKey.ToSdkKey(),
				)
				if err != nil {
					// cannot happen, but just in case add this check.
					// simply skip the validator if it does.
					logger.Error(
						"failed to instantiate validator",
						"i", i, "validator", change.PubKey,
					)
					continue
				}
				logger.Info("adding validator", "consAddress", consAddress)
				k.SetSubscriberValidator(cc, ocVal)
				if err := k.Hooks().AfterValidatorBonded(cc, consAddress, nil); err != nil {
					logger.Error(
						"failed to execute AfterValidatorBonded hook",
						"error", err,
					)
					continue
				}
				writeFunc()
			} else {
				// edge case: we received an update for 0 power
				// but the validator is already deleted. Do not forward
				// to tendermint.
				logger.Info(
					"received update for non-existent validator",
					"cons address", consAddress,
				)
				continue
			}
		}
		ret = append(ret, change)
	}
	return ret
}

// SetSubscriberValidator stores a validator based on the pub key derived address.
func (k Keeper) SetSubscriberValidator(
	ctx sdk.Context, validator commontypes.SubscriberValidator,
) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&validator)

	store.Set(types.SubscriberValidatorKey(validator.ConsAddress), bz)
}

// GetSubscriberValidator gets a validator based on the pub key derived (consensus) address.
func (k Keeper) GetSubscriberValidator(
	ctx sdk.Context, addr sdk.ConsAddress,
) (validator commontypes.SubscriberValidator, found bool) {
	store := ctx.KVStore(k.storeKey)
	v := store.Get(types.SubscriberValidatorKey(addr))
	if v == nil {
		return
	}
	k.cdc.MustUnmarshal(v, &validator)
	found = true

	return
}

// DeleteSubscriberValidator deletes a validator based on the pub key derived address.
func (k Keeper) DeleteSubscriberValidator(ctx sdk.Context, addr sdk.ConsAddress) {
	store := ctx.KVStore(k.storeKey)
	key := types.SubscriberValidatorKey(addr)
	if store.Has(key) {
		store.Delete(key)
	} else {
		k.Logger(ctx).Info("validator not found", "address", addr)
	}
}

// GetAllSubscriberValidators returns all validators in the store.
func (k Keeper) GetAllSubscriberValidators(
	ctx sdk.Context,
) (validators []commontypes.SubscriberValidator) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{types.SubscriberValidatorBytePrefix})

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		val := commontypes.SubscriberValidator{}
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		validators = append(validators, val)
	}

	return validators
}

// GetHistoricalInfo gets the historical info at a given height
func (k Keeper) GetHistoricalInfo(
	ctx sdk.Context,
	height int64,
) (stakingtypes.HistoricalInfo, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.HistoricalInfoKey(height)

	value := store.Get(key)
	if value == nil {
		return stakingtypes.HistoricalInfo{}, false
	}

	return stakingtypes.MustUnmarshalHistoricalInfo(k.cdc, value), true
}

// SetHistoricalInfo sets the historical info at a given height
func (k Keeper) SetHistoricalInfo(
	ctx sdk.Context,
	height int64,
	hi *stakingtypes.HistoricalInfo,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.HistoricalInfoKey(height)
	value := k.cdc.MustMarshal(hi)

	store.Set(key, value)
}

// DeleteHistoricalInfo deletes the historical info at a given height
func (k Keeper) DeleteHistoricalInfo(ctx sdk.Context, height int64) {
	store := ctx.KVStore(k.storeKey)
	key := types.HistoricalInfoKey(height)

	store.Delete(key)
}

// TrackHistoricalInfo saves the latest historical-info and deletes the oldest
// heights that are below pruning height
func (k Keeper) TrackHistoricalInfo(ctx sdk.Context) {
	numHistoricalEntries := int64(k.GetParams(ctx).HistoricalEntries)

	// Prune store to ensure we only have parameter-defined historical entries.
	// In most cases, this will involve removing a single historical entry.
	// In the rare scenario when the historical entries gets reduced to a lower value k'
	// from the original value k. k - k' entries must be deleted from the store.
	// Since the entries to be deleted are always in a continuous range, we can iterate
	// over the historical entries starting from the most recent version to be pruned
	// and then return at the first empty entry.
	for i := ctx.BlockHeight() - numHistoricalEntries; i >= 0; i-- {
		_, found := k.GetHistoricalInfo(ctx, i)
		if found {
			k.DeleteHistoricalInfo(ctx, i)
		} else {
			break
		}
	}

	// if there is no need to persist historicalInfo, return
	if numHistoricalEntries == 0 {
		return
	}

	// Create HistoricalInfo struct
	lastVals := []stakingtypes.Validator{}
	for _, v := range k.GetAllSubscriberValidators(ctx) {
		pk, err := v.ConsPubKey()
		if err != nil {
			// This should never happen as the pubkey is assumed
			// to be stored correctly in ApplyCCValidatorChanges.
			panic(err)
		}
		val, err := stakingtypes.NewValidator(nil, pk, stakingtypes.Description{})
		if err != nil {
			// This should never happen as the pubkey is assumed
			// to be stored correctly in ApplyCCValidatorChanges.
			panic(err)
		}

		// Set validator to bonded status
		val.Status = stakingtypes.Bonded
		// Compute tokens from voting power
		val.Tokens = sdk.TokensFromConsensusPower(
			v.Power, sdk.DefaultPowerReduction,
		)
		lastVals = append(lastVals, val)
	}

	// Create historical info entry which sorts the validator set by voting power
	historicalEntry := stakingtypes.NewHistoricalInfo(
		ctx.BlockHeader(), lastVals, sdk.DefaultPowerReduction,
	)

	// Set latest HistoricalInfo at current height
	k.SetHistoricalInfo(ctx, ctx.BlockHeight(), &historicalEntry)
}
