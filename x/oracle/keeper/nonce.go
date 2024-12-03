package keeper

import (
	"errors"
	"fmt"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetNonce get the nonce for a specific validator
func (k Keeper) GetNonce(ctx sdk.Context, validator string) (nonce types.ValidatorNonce, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	return k.getNonce(store, validator)
}

// SetNonce set the nonce for a specific validator
func (k Keeper) SetNonce(ctx sdk.Context, nonce types.ValidatorNonce) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	k.setNonce(store, nonce)
}

// AddNonceItem add a nonce item for a specific validator
func (k Keeper) AddNonceItem(ctx sdk.Context, nonce types.ValidatorNonce) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	if n, found := k.getNonce(store, nonce.Validator); found {
		feederIDs := make(map[uint64]struct{})
		for _, v := range n.NonceList {
			feederIDs[v.FeederID] = struct{}{}
		}
		for _, v := range nonce.NonceList {
			if _, ok := feederIDs[v.FeederID]; ok {
				continue
			}
			n.NonceList = append(n.NonceList, v)
		}
		k.setNonce(store, n)
	} else {
		k.setNonce(store, nonce)
	}
}

// AddZeroNonceItemForValidators init the nonce of a specific feederID for a set of validators
func (k Keeper) AddZeroNonceItemWithFeederIDForValidators(ctx sdk.Context, feederID uint64, validators []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	for _, validator := range validators {
		if n, found := k.getNonce(store, validator); found {
			found := false
			for _, v := range n.NonceList {
				if v.FeederID == feederID {
					found = true
					break
				}
			}
			if !found {
				n.NonceList = append(n.NonceList, &types.Nonce{FeederID: feederID, Value: 0})
				k.setNonce(store, n)
			}
		} else {
			k.setNonce(store, types.ValidatorNonce{Validator: validator, NonceList: []*types.Nonce{{FeederID: feederID, Value: 0}}})
		}
	}
}

// RemoveNonceWithValidator remove the nonce for a specific validator
func (k Keeper) RemoveNonceWithValidator(ctx sdk.Context, validator string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	k.removeNonceWithValidator(store, validator)
}

// RemoveNonceWithValidatorAndFeederID remove the nonce for a specific validator and feederID
func (k Keeper) RemoveNonceWithValidatorAndFeederID(ctx sdk.Context, validator string, feederID uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	if nonce, found := k.GetNonce(ctx, validator); found {
		for i, n := range nonce.NonceList {
			if n.FeederID == feederID {
				nonce.NonceList = append(nonce.NonceList[:i], nonce.NonceList[i+1:]...)
				if len(nonce.NonceList) == 0 {
					k.removeNonceWithValidator(store, validator)
				} else {
					k.setNonce(store, nonce)
				}
				return true
			}
		}
	}
	return false
}

// RemoveNonceWithFeederIDForValidators remove the nonce for a specific feederID from a set of validators
func (k Keeper) RemoveNonceWithFeederIDForValidators(ctx sdk.Context, feederID uint64, validators []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	k.removeNonceWithFeederIDForValidators(store, feederID, validators)
}

// RemoveNonceWithFeederIDForAll remove the nonce for a specific feederID from all validators
func (k Keeper) RemoveNonceWithFeederIDForAll(ctx sdk.Context, feederID uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	var validators []string
	for ; iterator.Valid(); iterator.Next() {
		var nonce types.ValidatorNonce
		k.cdc.MustUnmarshal(iterator.Value(), &nonce)
		validators = append(validators, nonce.Validator)
	}
	k.removeNonceWithFeederIDForValidators(store, feederID, validators)
}

// CheckAndIncreaseNonce check and increase the nonce for a specific validator and feederID
func (k Keeper) CheckAndIncreaseNonce(ctx sdk.Context, validator string, feederID uint64, nonce uint32) (prevNonce uint32, err error) {
	if nonce > uint32(common.MaxNonce) {
		return 0, errors.New("nonce is too large")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	if n, found := k.getNonce(store, validator); found {
		for _, v := range n.NonceList {
			if v.FeederID == feederID {
				if v.Value+1 == nonce {
					v.Value++
					k.setNonce(store, n)
					return nonce - 1, nil
				}
				return v.Value, errors.New("nonce is not consecutive")
			}
		}
		return 0, errors.New("feeder not found")
	}
	return 0, fmt.Errorf("validator for the consKey which signed the create-price tx is not included in active validator set, signer consAddr:%s", validator)
}

// internal usage for avoiding duplicated 'NewStore'

func (k Keeper) getNonce(store prefix.Store, validator string) (types.ValidatorNonce, bool) {
	bz := store.Get(types.NonceKey(validator))
	if bz != nil {
		var nonce types.ValidatorNonce
		k.cdc.MustUnmarshal(bz, &nonce)
		return nonce, true
	}
	return types.ValidatorNonce{}, false
}

func (k Keeper) setNonce(store prefix.Store, nonce types.ValidatorNonce) {
	bz := k.cdc.MustMarshal(&nonce)
	store.Set(types.NonceKey(nonce.Validator), bz)
}

func (k Keeper) removeNonceWithValidator(store prefix.Store, validator string) {
	store.Delete(types.NonceKey(validator))
}

func (k Keeper) removeNonceWithValidatorAndFeederID(store prefix.Store, validator string, feederID uint64) bool {
	if nonce, found := k.getNonce(store, validator); found {
		// TODO: performance. key-value: validator_feederID:nonce
		for i, n := range nonce.NonceList {
			if n.FeederID == feederID {
				nonce.NonceList = append(nonce.NonceList[:i], nonce.NonceList[i+1:]...)
				if len(nonce.NonceList) == 0 {
					k.removeNonceWithValidator(store, validator)
				} else {
					k.setNonce(store, nonce)
				}
				return true
			}
		}
	}
	return false
}

func (k Keeper) removeNonceWithFeederIDForValidators(store prefix.Store, feederID uint64, validators []string) {
	for _, validator := range validators {
		k.removeNonceWithValidatorAndFeederID(store, validator, feederID)
	}
}
