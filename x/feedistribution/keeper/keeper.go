package keeper

import (
	"fmt"
	"sort"
	"strings"

	delegationkeeper "github.com/imua-xyz/imuachain/x/delegation/keeper"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	stakingkeeper "github.com/imua-xyz/imuachain/x/dogfood/keeper"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		logger   log.Logger
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority        string
		authKeeper       feedistributiontypes.AccountKeeper
		bankKeeper       feedistributiontypes.BankKeeper
		epochsKeeper     feedistributiontypes.EpochsKeeper
		operatorKeeper   feedistributiontypes.OperatorKeeper
		avsKeeper        feedistributiontypes.AVSKeeper
		assetsKeeper     feedistributiontypes.AssetsKeeper
		delegationKeeper feedistributiontypes.DelegationKeeper
		SlashKeeper      feedistributiontypes.SlashKeeper
		StakingKeeper    feedistributiontypes.StakingKeeper
		OracleKeeper     feedistributiontypes.OracleKeeper

		feeCollectorName string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	logger log.Logger,
	feeCollectorName, authority string,
	storeKey storetypes.StoreKey,
	bankKeeper feedistributiontypes.BankKeeper,
	accountKeeper feedistributiontypes.AccountKeeper,
	stakingkeeper stakingkeeper.Keeper,
	epochKeeper feedistributiontypes.EpochsKeeper,
	operatorKeeper feedistributiontypes.OperatorKeeper,
	avsKeeper feedistributiontypes.AVSKeeper,
	assetsKeeper feedistributiontypes.AssetsKeeper,
	delegationKeeper feedistributiontypes.DelegationKeeper,
	slashKeeper feedistributiontypes.SlashKeeper,
	oracleKeeper feedistributiontypes.OracleKeeper,
) Keeper {
	// ensure distribution module account is set
	if addr := accountKeeper.GetModuleAddress(feedistributiontypes.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", feedistributiontypes.ModuleName))
	}

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	k := &Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		logger:           logger,
		authority:        authority,
		authKeeper:       accountKeeper,
		bankKeeper:       bankKeeper,
		epochsKeeper:     epochKeeper,
		feeCollectorName: feeCollectorName,
		StakingKeeper:    stakingkeeper,
		operatorKeeper:   operatorKeeper,
		avsKeeper:        avsKeeper,
		assetsKeeper:     assetsKeeper,
		delegationKeeper: delegationKeeper,
		SlashKeeper:      slashKeeper,
		OracleKeeper:     oracleKeeper,
	}

	return *k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", feedistributiontypes.ModuleName))
}

// GenericSetAllItems stores a list of key-value pairs in a prefixed store.
// it's used for the genesis import.
func GenericSetAllItems[T any](
	ctx sdk.Context, keeper Keeper,
	keyPrefix []byte, items []T,
	getKey func(T) []byte,
	getValue func(T) codec.ProtoMarshaler,
) error {
	store := prefix.NewStore(ctx.KVStore(keeper.storeKey), keyPrefix)

	for _, item := range items {
		value := getValue(item)
		if value == nil {
			return fmt.Errorf("nil value returned for item")
		}
		bz := keeper.cdc.MustMarshal(value)
		store.Set(getKey(item), bz)
	}

	return nil
}

// GenericGetAllItems retrieves all key-value pairs from a prefixed store.
// it's used for the genesis export.
func GenericGetAllItems[T any](
	ctx sdk.Context,
	keeper Keeper, keyPrefix []byte,
	newValue func() codec.ProtoMarshaler,
	createItem func(key []byte, value codec.ProtoMarshaler) T,
) ([]T, error) {
	store := prefix.NewStore(ctx.KVStore(keeper.storeKey), keyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]T, 0)
	for ; iterator.Valid(); iterator.Next() {
		value := newValue()
		keeper.cdc.MustUnmarshal(iterator.Value(), value)
		ret = append(ret, createItem(iterator.Key(), value))
	}

	return ret, nil
}

func (k Keeper) SetAllAVSRewardParams(ctx sdk.Context, allAVSRewardParams []feedistributiontypes.AVSAddrAndRewardParam) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixAVSRewardParam, allAVSRewardParams,
		func(item feedistributiontypes.AVSAddrAndRewardParam) []byte {
			return common.HexToAddress(item.Avs).Bytes()
		},
		func(item feedistributiontypes.AVSAddrAndRewardParam) codec.ProtoMarshaler {
			return &item.AvsRewardParam
		},
	)
}

func (k Keeper) GetAllAVSRewardParams(ctx sdk.Context) ([]feedistributiontypes.AVSAddrAndRewardParam, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixAVSRewardParam,
		func() codec.ProtoMarshaler { return &feedistributiontypes.AVSRewardParam{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.AVSAddrAndRewardParam {
			return feedistributiontypes.AVSAddrAndRewardParam{
				Avs:            strings.ToLower(common.BytesToAddress(key).String()),
				AvsRewardParam: *value.(*feedistributiontypes.AVSRewardParam),
			}
		},
	)
}

func (k Keeper) SetAllAVSFeePools(ctx sdk.Context, allAVSFeePools []feedistributiontypes.AVSAddrAndFeePool) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixFeePools, allAVSFeePools,
		func(item feedistributiontypes.AVSAddrAndFeePool) []byte {
			return common.HexToAddress(item.Avs).Bytes()
		},
		func(item feedistributiontypes.AVSAddrAndFeePool) codec.ProtoMarshaler {
			return &item.AvsFeePool
		},
	)
}

func (k Keeper) GetAllAVSFeePools(ctx sdk.Context) ([]feedistributiontypes.AVSAddrAndFeePool, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixFeePools,
		func() codec.ProtoMarshaler { return &feedistributiontypes.FeePool{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.AVSAddrAndFeePool {
			return feedistributiontypes.AVSAddrAndFeePool{
				Avs:        strings.ToLower(common.BytesToAddress(key).String()),
				AvsFeePool: *value.(*feedistributiontypes.FeePool),
			}
		},
	)
}

func (k Keeper) SetAllAVSRewardDistributions(ctx sdk.Context, allAVSRewardDistributions []feedistributiontypes.AVSAddrAndRewardDistribution) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixAVSRewardDistribution, allAVSRewardDistributions,
		func(item feedistributiontypes.AVSAddrAndRewardDistribution) []byte {
			return common.HexToAddress(item.Avs).Bytes()
		},
		func(item feedistributiontypes.AVSAddrAndRewardDistribution) codec.ProtoMarshaler {
			return &item.AvsRewardDistribution
		},
	)
}

func (k Keeper) GetAllAVSRewardDistributions(ctx sdk.Context) ([]feedistributiontypes.AVSAddrAndRewardDistribution, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixAVSRewardDistribution,
		func() codec.ProtoMarshaler { return &feedistributiontypes.AVSRewardDistribution{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.AVSAddrAndRewardDistribution {
			return feedistributiontypes.AVSAddrAndRewardDistribution{
				Avs:                   strings.ToLower(common.BytesToAddress(key).String()),
				AvsRewardDistribution: *value.(*feedistributiontypes.AVSRewardDistribution),
			}
		},
	)
}

func (k Keeper) SetAllOperatorUnclaimedRewards(
	ctx sdk.Context, allOperatorUnclaimedRewards []feedistributiontypes.KeyAndOperatorUnclaimedRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorUnclaimedRewards, allOperatorUnclaimedRewards,
		func(item feedistributiontypes.KeyAndOperatorUnclaimedRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorUnclaimedRewards) codec.ProtoMarshaler {
			return &item.OperatorUnclaimedRewards
		},
	)
}

func (k Keeper) GetAllOperatorUnclaimedRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorUnclaimedRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorUnclaimedRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorUnclaimedRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorUnclaimedRewards {
			return feedistributiontypes.KeyAndOperatorUnclaimedRewards{
				Key:                      string(key),
				OperatorUnclaimedRewards: *value.(*feedistributiontypes.OperatorUnclaimedRewards),
			}
		},
	)
}

func (k Keeper) SetAllDelegationChangeInfo(
	ctx sdk.Context, allDelegationChangeInfos []feedistributiontypes.KeyAndDelegationChangeInfo,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixStakeChangeDelegations, allDelegationChangeInfos,
		func(item feedistributiontypes.KeyAndDelegationChangeInfo) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndDelegationChangeInfo) codec.ProtoMarshaler {
			return &item.DelegationChangeInfo
		},
	)
}

func (k Keeper) GetAllDelegationChangeInfo(ctx sdk.Context) ([]feedistributiontypes.KeyAndDelegationChangeInfo, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixStakeChangeDelegations,
		func() codec.ProtoMarshaler { return &feedistributiontypes.DelegationChangeInfo{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndDelegationChangeInfo {
			return feedistributiontypes.KeyAndDelegationChangeInfo{
				Key:                  string(key),
				DelegationChangeInfo: *value.(*feedistributiontypes.DelegationChangeInfo),
			}
		},
	)
}

func (k Keeper) SetAllDelegationStartingInfo(
	ctx sdk.Context, allDelegationStartingInfos []feedistributiontypes.KeyAndDelegationStartingInfo,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixDelegationStartingInfo, allDelegationStartingInfos,
		func(item feedistributiontypes.KeyAndDelegationStartingInfo) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndDelegationStartingInfo) codec.ProtoMarshaler {
			return &item.DelegationStartingInfo
		},
	)
}

func (k Keeper) GetAllDelegationStartingInfo(ctx sdk.Context) ([]feedistributiontypes.KeyAndDelegationStartingInfo, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixDelegationStartingInfo,
		func() codec.ProtoMarshaler { return &feedistributiontypes.DelegationStartingInfo{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndDelegationStartingInfo {
			return feedistributiontypes.KeyAndDelegationStartingInfo{
				Key:                    string(key),
				DelegationStartingInfo: *value.(*feedistributiontypes.DelegationStartingInfo),
			}
		},
	)
}

func (k Keeper) SetAllOperatorHistoricalRewards(
	ctx sdk.Context, allOperatorHistoricalRewards []feedistributiontypes.KeyAndOperatorHistoricalRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorHistoricalRewards, allOperatorHistoricalRewards,
		func(item feedistributiontypes.KeyAndOperatorHistoricalRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorHistoricalRewards) codec.ProtoMarshaler {
			return &item.OperatorHistoricalRewards
		},
	)
}

func (k Keeper) GetAllOperatorHistoricalRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorHistoricalRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorHistoricalRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorHistoricalRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorHistoricalRewards {
			return feedistributiontypes.KeyAndOperatorHistoricalRewards{
				Key:                       string(key),
				OperatorHistoricalRewards: *value.(*feedistributiontypes.OperatorHistoricalRewards),
			}
		},
	)
}

func (k Keeper) SetAllOperatorCurrentRewards(
	ctx sdk.Context, allOperatorCurrentRewards []feedistributiontypes.KeyAndOperatorCurrentRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorCurrentRewards, allOperatorCurrentRewards,
		func(item feedistributiontypes.KeyAndOperatorCurrentRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorCurrentRewards) codec.ProtoMarshaler {
			return &item.OperatorCurrentRewards
		},
	)
}

func (k Keeper) GetAllOperatorCurrentRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorCurrentRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorCurrentRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorCurrentRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorCurrentRewards {
			return feedistributiontypes.KeyAndOperatorCurrentRewards{
				Key:                    string(key),
				OperatorCurrentRewards: *value.(*feedistributiontypes.OperatorCurrentRewards),
			}
		},
	)
}

func (k Keeper) SetAllOperatorCommission(
	ctx sdk.Context, allOperatorCommission []feedistributiontypes.KeyAndOperatorCommission,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorCommission, allOperatorCommission,
		func(item feedistributiontypes.KeyAndOperatorCommission) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorCommission) codec.ProtoMarshaler {
			return &item.OperatorCommission
		},
	)
}

func (k Keeper) GetAllOperatorCommission(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorCommission, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorCommission,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorCommission{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorCommission {
			return feedistributiontypes.KeyAndOperatorCommission{
				Key:                string(key),
				OperatorCommission: *value.(*feedistributiontypes.OperatorCommission),
			}
		},
	)
}

func (k Keeper) SetAllOperatorSlashEvent(
	ctx sdk.Context, allOperatorSlashEvent []feedistributiontypes.KeyAndOperatorSlashEvent,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorSlashEvent, allOperatorSlashEvent,
		func(item feedistributiontypes.KeyAndOperatorSlashEvent) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorSlashEvent) codec.ProtoMarshaler {
			return &item.OperatorSlashEvent
		},
	)
}

func (k Keeper) GetAllOperatorSlashEvent(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorSlashEvent, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorSlashEvent,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorSlashEvent{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorSlashEvent {
			return feedistributiontypes.KeyAndOperatorSlashEvent{
				Key:                string(key),
				OperatorSlashEvent: *value.(*feedistributiontypes.OperatorSlashEvent),
			}
		},
	)
}

func (k Keeper) SetAllStakerClaimedRewards(
	ctx sdk.Context, allStakerClaimedRewards []feedistributiontypes.KeyAndStakerClaimedRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixStakerClaimedRewards, allStakerClaimedRewards,
		func(item feedistributiontypes.KeyAndStakerClaimedRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndStakerClaimedRewards) codec.ProtoMarshaler {
			return &item.StakerClaimedRewards
		},
	)
}

func (k Keeper) GetAllStakerClaimedRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndStakerClaimedRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixStakerClaimedRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.StakerClaimedRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndStakerClaimedRewards {
			return feedistributiontypes.KeyAndStakerClaimedRewards{
				Key:                  string(key),
				StakerClaimedRewards: *value.(*feedistributiontypes.StakerClaimedRewards),
			}
		},
	)
}

func (k Keeper) GetAllStakerRewardParams(ctx sdk.Context) ([]feedistributiontypes.KeyAndStakerRewardParams, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixStakerRewardParams,
		func() codec.ProtoMarshaler { return &feedistributiontypes.StakerRewardParams{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndStakerRewardParams {
			return feedistributiontypes.KeyAndStakerRewardParams{
				Key:                string(key),
				StakerRewardParams: *value.(*feedistributiontypes.StakerRewardParams),
			}
		},
	)
}

func (k Keeper) SetAllStakerRewardParams(
	ctx sdk.Context, allStakerRewardParams []feedistributiontypes.KeyAndStakerRewardParams,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixStakerRewardParams, allStakerRewardParams,
		func(item feedistributiontypes.KeyAndStakerRewardParams) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndStakerRewardParams) codec.ProtoMarshaler {
			return &item.StakerRewardParams
		},
	)
}

func (k Keeper) NormalizeRewardDecCoins(ctx sdk.Context, avsAddr string, rewards sdk.DecCoins) (sdk.DecCoins, error) {
	normalized := append(sdk.DecCoins(nil), rewards...)
	for i := range normalized {
		// get the decimal of reward asset
		_, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, avsAddr, normalized[i].Denom)
		if err != nil {
			return nil, err
		}
		normalized[i].Amount = feedistributiontypes.TruncateSDKDec(normalized[i].Amount, rewardAsset.RewardAssetInfo.DenominationExponent)
	}
	return normalized, nil
}

func (k Keeper) BatchNormalizeClaimedRewardDecimals(ctx sdk.Context, rewards []feedistributiontypes.StakerClaimedRewardsPerAVS) ([]feedistributiontypes.StakerClaimedRewardsPerAVS, error) {
	out := make([]feedistributiontypes.StakerClaimedRewardsPerAVS, len(rewards))
	copy(out, rewards)
	for i := range out {
		normalizedOutstandingRewards, err := k.NormalizeRewardDecCoins(ctx, out[i].AVSAddress, out[i].ClaimedRewards.OutstandingRewards)
		if err != nil {
			return nil, err
		}
		normalizedWithdrawnRewards, err := k.NormalizeRewardDecCoins(ctx, out[i].AVSAddress, out[i].ClaimedRewards.WithdrawnRewards)
		if err != nil {
			return nil, err
		}
		out[i].ClaimedRewards.OutstandingRewards = normalizedOutstandingRewards
		out[i].ClaimedRewards.WithdrawnRewards = normalizedWithdrawnRewards
	}
	return out, nil
}

func (k Keeper) BatchNormalizeRewardDecimals(ctx sdk.Context, rewards feedistributiontypes.CommonAVSRewards) (feedistributiontypes.CommonAVSRewards, error) {
	out := append(feedistributiontypes.CommonAVSRewards(nil), rewards...)
	for i := range out {
		normalized, err := k.NormalizeRewardDecCoins(ctx, out[i].AVSAddress, out[i].Rewards)
		if err != nil {
			return nil, err
		}
		out[i].Rewards = normalized
	}
	return out, nil
}

func (k Keeper) DecCoinsToRewardInfos(ctx sdk.Context, avsAddr string, rewards sdk.DecCoins) ([]feedistributiontypes.RewardInfo, error) {
	rewardInfos := make([]feedistributiontypes.RewardInfo, len(rewards))
	for i := range rewards {
		// get the decimal of reward asset
		assetID, assetInfo, err := k.GetAVSRewardAssetByDenomination(ctx, avsAddr, rewards[i].Denom)
		if err != nil {
			return nil, err
		}
		rewardAmount := feedistributiontypes.UnscaleDecToInt(rewards[i].Amount, assetInfo.RewardAssetInfo.DenominationExponent)
		rewardInfos[i] = feedistributiontypes.RewardInfo{
			AssetId: assetID,
			Decimal: assetInfo.RewardAssetInfo.Decimals,
			Amount:  rewardAmount,
		}
	}
	return rewardInfos, nil
}

// MergeStakerRewards merges outstanding and unclaimed rewards by AVS address,
// and returns a list of StakerRewardsPerAVS.
//
// Each AVS address will have its corresponding outstanding and unclaimed rewards
// grouped under the same StakerRewardsPerAVS entry.
func (k Keeper) MergeStakerRewards(
	ctx sdk.Context,
	claimedRewards []feedistributiontypes.StakerClaimedRewardsPerAVS,
	unclaimedStakingRewards, unclaimedCompoundingRewards []feedistributiontypes.CommonAVSRewardData,
) ([]feedistributiontypes.StakerRewardsPerAVS, error) {
	// Create a map to aggregate rewards by AVS address
	rewardMap := make(map[string]feedistributiontypes.StakerRewardsPerAVS)

	// Process outstanding rewards
	for _, claimedRewardsPerAVS := range claimedRewards {
		// Convert DecCoins to RewardInfo
		outstandingRewardInfos, err := k.DecCoinsToRewardInfos(ctx, claimedRewardsPerAVS.AVSAddress, claimedRewardsPerAVS.ClaimedRewards.OutstandingRewards)
		if err != nil {
			return nil, err
		}
		withdrawnRewardInfos, err := k.DecCoinsToRewardInfos(ctx, claimedRewardsPerAVS.AVSAddress, claimedRewardsPerAVS.ClaimedRewards.WithdrawnRewards)
		if err != nil {
			return nil, err
		}
		historicalTotalRewardInfos, err := k.DecCoinsToRewardInfos(ctx, claimedRewardsPerAVS.AVSAddress, claimedRewardsPerAVS.ClaimedRewards.HistoricalTotalRewards)
		if err != nil {
			return nil, err
		}
		// calculate the delegation rewards amount from shares.
		delegatedRewardsMap := make(map[string]feedistributiontypes.RewardInfo)
		for _, rewardDelegation := range claimedRewardsPerAVS.ClaimedRewards.DelegationRewardsShares {
			operatorAccAddr, err := sdk.AccAddressFromBech32(rewardDelegation.OperatorAddr)
			if err != nil {
				return nil, err
			}
			for _, singleRewardShare := range rewardDelegation.Shares {
				assetID, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, claimedRewardsPerAVS.AVSAddress, singleRewardShare.Denom)
				if err != nil {
					return nil, err
				}
				operatorAssets, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operatorAccAddr, assetID)
				if err != nil {
					return nil, err
				}
				assetAmount, err := delegationkeeper.TokensFromShares(singleRewardShare.Amount, operatorAssets.TotalShare, operatorAssets.TotalAmount)
				if err != nil {
					return nil, err
				}
				delegatedReward, ok := delegatedRewardsMap[assetID]
				if !ok {
					delegatedRewardsMap[assetID] = feedistributiontypes.RewardInfo{
						AssetId: assetID,
						Decimal: rewardAsset.RewardAssetInfo.Decimals,
						Amount:  assetAmount,
					}
				} else {
					delegatedReward.Amount = delegatedReward.Amount.Add(assetAmount)
					delegatedRewardsMap[assetID] = delegatedReward
				}
			}
		}
		// Convert the delegation rewards map into a slice
		delegatedRewardsSlice := make([]feedistributiontypes.RewardInfo, 0)
		for _, v := range delegatedRewardsMap {
			delegatedRewardsSlice = append(delegatedRewardsSlice, v)
		}
		// sort the slice by assetID
		sort.Slice(delegatedRewardsSlice, func(i, j int) bool { return delegatedRewardsSlice[i].AssetId < delegatedRewardsSlice[j].AssetId })

		pendingUndelegationRewardInfos, err := k.DecCoinsToRewardInfos(ctx, claimedRewardsPerAVS.AVSAddress, claimedRewardsPerAVS.ClaimedRewards.PendingUndelegationRewards)
		if err != nil {
			return nil, err
		}
		withdrawableRewardInfos, err := k.DecCoinsToRewardInfos(ctx, claimedRewardsPerAVS.AVSAddress, claimedRewardsPerAVS.ClaimedRewards.WithdrawableRewards)
		if err != nil {
			return nil, err
		}
		// assign to outstanding and withdrawn rewards
		entry, exists := rewardMap[claimedRewardsPerAVS.AVSAddress]
		if !exists {
			entry = feedistributiontypes.StakerRewardsPerAVS{
				AVSAddress: claimedRewardsPerAVS.AVSAddress,
			}
		}
		entry.OutstandingRewards = outstandingRewardInfos
		entry.WithdrawnRewards = withdrawnRewardInfos
		entry.HistoricalTotalRewards = historicalTotalRewardInfos
		entry.DelegationRewards = delegatedRewardsSlice
		entry.PendingUndelegationRewards = pendingUndelegationRewardInfos
		entry.WithdrawableRewards = withdrawableRewardInfos

		rewardMap[claimedRewardsPerAVS.AVSAddress] = entry
	}

	// Process unclaimed staking rewards
	for _, data := range unclaimedStakingRewards {
		// Convert DecCoins to RewardInfo
		rewardInfos, err := k.DecCoinsToRewardInfos(ctx, data.AVSAddress, data.Rewards)
		if err != nil {
			return nil, err
		}
		entry, exists := rewardMap[data.AVSAddress]
		if !exists {
			// Initialize the entry if it doesn't exist
			entry = feedistributiontypes.StakerRewardsPerAVS{
				AVSAddress: data.AVSAddress,
			}
		}
		// assign to unclaimed_rewards
		entry.UnclaimedRewards = rewardInfos
		rewardMap[data.AVSAddress] = entry
	}

	// Process unclaimed compounding rewards
	for _, data := range unclaimedCompoundingRewards {
		// Convert DecCoins to RewardInfo
		rewardInfos, err := k.DecCoinsToRewardInfos(ctx, data.AVSAddress, data.Rewards)
		if err != nil {
			return nil, err
		}
		entry, exists := rewardMap[data.AVSAddress]
		if !exists {
			// Initialize the entry if it doesn't exist
			entry = feedistributiontypes.StakerRewardsPerAVS{
				AVSAddress: data.AVSAddress,
			}
		}
		// assign to unclaimed_compounding_rewards
		entry.UnclaimedCompoundingRewards = rewardInfos
		rewardMap[data.AVSAddress] = entry
	}

	// Convert the aggregated map into a slice
	result := make([]feedistributiontypes.StakerRewardsPerAVS, 0)
	for _, v := range rewardMap {
		result = append(result, v)
	}
	// sort the slice by avs address
	sort.Slice(result, func(i, j int) bool { return result[i].AVSAddress < result[j].AVSAddress })
	return result, nil
}
