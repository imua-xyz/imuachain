package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
)

// AddSubscriberChain adds a subscriber chain to the pending list. It will be started at the
// next epoch, with the identifier set in the `req.EpochIdentifier` field. The caller must
// have performed stateless validation on the request.
func (k Keeper) AddSubscriberChain(
	ctx sdk.Context,
	req *types.RegisterSubscriberChainRequest,
) (*types.RegisterSubscriberChainResponse, error) {
	if req == nil {
		return nil, types.ErrNilRequest
	}
	// We have already validated
	// 1. The unbonding epochs are not 0
	// 2. The chainID is not blank
	// 3. The duration is positive.
	// 4. Max validators is not 0
	// 5. Asset IDs are valid format, but may not be registered
	// 6. Min self delegation is not nil or not negative
	// The AVS keeper validates:
	// 1. Unique chainID, and it removes the version before this validation
	// 2. Epoch is registered in x/epochs
	// 3. All the assetIDs are registered in x/assets
	// We don't have to check for anything else, but we should store some of this stuff
	// within this module (or potentially load it from the AVS module).
	epochInfo, found := k.epochsKeeper.GetEpochInfo(ctx, req.EpochIdentifier)
	if !found {
		return nil, types.ErrInvalidRegistrationParams.Wrapf("epoch not found %s", req.EpochIdentifier)
	}
	// this value is required by the AVS module when making edits or deleting the AVS. as always, we round it up for the
	// current epoch. it can never be 0, because both the durations are positive.
	unbondingEpochs := (req.SubscriberParams.UnbondingPeriod + epochInfo.Duration - 1) / epochInfo.Duration
	if _, err := k.avsKeeper.RegisterAVSWithChainID(ctx, &avstypes.AVSRegisterOrDeregisterParams{
		AvsName:           req.ChainID,
		AssetIDs:          req.AssetIDs,
		UnbondingPeriod:   uint64(unbondingEpochs), // estimated
		MinSelfDelegation: req.MinSelfDelegationUsd,
		EpochIdentifier:   req.EpochIdentifier,
		ChainID:           req.ChainID,
		// TODO: remove the owner role and make it controllable by subscriber-governance
		AvsOwnerAddresses: []string{req.FromAddress},
	}); err != nil {
		return nil, types.ErrInvalidRegistrationParams.Wrap(err.Error())
	}
	// store the data here to generate the genesis state of the subscriber at the end of the current epoch, in the
	// AfterEpochEnd hook.
	k.AppendPendingSubChain(ctx, req.EpochIdentifier, uint64(epochInfo.CurrentEpoch), req)
	return &types.RegisterSubscriberChainResponse{}, nil
}

// AppendPendingSubChain appends a pending subscriber chain to be started at the epoch-th epochIdentifier.
func (k Keeper) AppendPendingSubChain(
	ctx sdk.Context,
	epochIdentifier string,
	epoch uint64,
	req *types.RegisterSubscriberChainRequest,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.PendingSubscriberChainKey(epochIdentifier, epoch)
	prev := k.GetPendingSubChains(ctx, epochIdentifier, epoch)
	// it is stored in the order of message processing
	prev.List = append(prev.List, *req)
	store.Set(key, k.cdc.MustMarshal(&prev))
}

// GetPendingSubChains gets the pending subscriber chains to be started at the epoch-th epochIdentifier.
func (k Keeper) GetPendingSubChains(
	ctx sdk.Context, epochIdentifier string, epoch uint64,
) types.PendingSubscriberChainRequests {
	store := ctx.KVStore(k.storeKey)
	key := types.PendingSubscriberChainKey(epochIdentifier, epoch)
	var res types.PendingSubscriberChainRequests
	if store.Has(key) {
		k.cdc.MustUnmarshal(store.Get(key), &res)
	}
	return res
}

// ClearPendingSubChains clears the pending subscriber chains to be started at the epoch-th epochIdentifier.
func (k Keeper) ClearPendingSubChains(
	ctx sdk.Context, epochIdentifier string, epoch uint64,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.PendingSubscriberChainKey(epochIdentifier, epoch)
	store.Delete(key)
}
