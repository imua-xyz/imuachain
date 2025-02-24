package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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

// RemoveNonceWithValidator remove the nonce for a specific validator
func (k Keeper) RemoveNonceWithValidator(ctx sdk.Context, validator string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	k.removeNonceWithValidator(store, validator)
}

func (k Keeper) CheckAndIncreaseNonce(ctx sdk.Context, validator string, feederID uint64, nonce uint32) (prevNonce uint32, err error) {
	maxNonce := k.GetMaxNonceFromCache()
	// #nosec G115  // safe conversion
	if nonce > uint32(maxNonce) {
		return 0, fmt.Errorf("nonce_check_failed: max_exceeded: limit=%d received=%d", maxNonce, nonce)
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
				return v.Value, fmt.Errorf("nonce_check_failed: non_consecutive: expected=%d received=%d", v.Value+1, nonce)
			}
		}
		return 0, fmt.Errorf("nonce_check_failed: feeder_not_found: validator=%s feeder_id=%d", validator, feederID)
	}
	return 0, fmt.Errorf("nonce_check_failed: validator_not_active: validator=%s tx_type=create-price", validator)
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

// AddZeroNonceItemWithFeederIDsForValidators init the nonce of a batch of feederIDs for a set of validators
// feederIDs must be ordered
func (k Keeper) AddZeroNonceItemWithFeederIDsForValidators(ctx sdk.Context, feederIDs []uint64, validators []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	for _, validator := range validators {
		if n, found := k.getNonce(store, validator); found {
			fIDs := make(map[uint64]struct{})
			for _, v := range n.NonceList {
				fIDs[v.FeederID] = struct{}{}
			}
			updated := false
			// added feederIDs are kept ordered
			for _, feederID := range feederIDs {
				if _, ok := fIDs[feederID]; !ok {
					n.NonceList = append(n.NonceList, &types.Nonce{FeederID: feederID, Value: 0})
					fIDs[feederID] = struct{}{}
					updated = true
				}
			}
			if updated {
				k.setNonce(store, n)
			}
		} else {
			n := types.ValidatorNonce{Validator: validator, NonceList: make([]*types.Nonce, 0, len(feederIDs))}
			// ordered feederIDs
			for _, feederID := range feederIDs {
				n.NonceList = append(n.NonceList, &types.Nonce{FeederID: feederID, Value: 0})
			}
			k.setNonce(store, n)
		}
	}
}

// RemoveNonceWithFeederIDsForValidators remove the nonce for a batch of feederIDs from a set of validators
func (k Keeper) RemoveNonceWithFeederIDsForValidators(ctx sdk.Context, feederIDs []uint64, validators []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	k.removeNonceWithFeederIDsForValidators(store, feederIDs, validators)
}

func (k Keeper) RemoveNonceWithFeederIDsForAll(ctx sdk.Context, feederIDs []uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.NonceKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	var validators []string
	for ; iterator.Valid(); iterator.Next() {
		var nonce types.ValidatorNonce
		k.cdc.MustUnmarshal(iterator.Value(), &nonce)
		validators = append(validators, nonce.Validator)
	}

	k.removeNonceWithFeederIDsForValidators(store, feederIDs, validators)
}

func (k Keeper) removeNonceWithFeederIDsForValidators(store prefix.Store, feederIDs []uint64, validators []string) {
	fIDs := make(map[uint64]struct{})
	for _, feederID := range feederIDs {
		fIDs[feederID] = struct{}{}
	}
	for _, validator := range validators {
		if nonce, found := k.getNonce(store, validator); found {
			l := len(nonce.NonceList)
			// the order in nonceList is kept after removed
			for i := 0; i < l; i++ {
				n := nonce.NonceList[i]
				if _, ok := fIDs[n.FeederID]; ok {
					nonce.NonceList = append(nonce.NonceList[:i], nonce.NonceList[i+1:]...)
				}
				i--
				l--
			}

			if len(nonce.NonceList) == 0 {
				k.removeNonceWithValidator(store, validator)
			} else {
				k.setNonce(store, nonce)
			}
		}
	}
}
