package keeper

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) MarkStakeChangeDelegations(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	impactfulEpochs, err := k.operatorKeeper.GetImpactfulEpochsForOperator(ctx, operator.String())
	if err != nil {
		return err
	}
	delegationKey := assetstype.GetJoinedStoreKey(stakerID, assetID, operator.String())
	delegationKeys := &feedistributiontypes.StakeChangeDelegations{
		DelegationKeys: make([]string, 0),
	}
	for _, epochIdentifier := range impactfulEpochs {
		value := store.Get([]byte(epochIdentifier))
		if value != nil {
			k.cdc.MustUnmarshal(value, delegationKeys)
		}
		delegationKeys.AppendUniqueDelegationKey(string(delegationKey))
		bz := k.cdc.MustMarshal(delegationKeys)
		store.Set([]byte(epochIdentifier), bz)
	}
	return nil
}

func (k Keeper) DeleteStakeChangeDelegations(ctx sdk.Context, epochIdentifier string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	store.Delete([]byte(epochIdentifier))
	return nil
}
