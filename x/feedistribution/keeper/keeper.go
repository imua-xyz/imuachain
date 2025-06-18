package keeper

import (
	"fmt"
	"strings"

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

		feeCollectorName string

		StakingKeeper stakingkeeper.Keeper
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

func (k Keeper) SetAllOperatorOutstandingRewards(
	ctx sdk.Context, allOperatorOutstandingRewards []feedistributiontypes.KeyAndOperatorOutstandingRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorOutstandingRewards, allOperatorOutstandingRewards,
		func(item feedistributiontypes.KeyAndOperatorOutstandingRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorOutstandingRewards) codec.ProtoMarshaler {
			return &item.OperatorOutstandingRewards
		},
	)
}

func (k Keeper) GetAllOperatorOutstandingRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorOutstandingRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorOutstandingRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorOutstandingRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorOutstandingRewards {
			return feedistributiontypes.KeyAndOperatorOutstandingRewards{
				Key:                        string(key),
				OperatorOutstandingRewards: *value.(*feedistributiontypes.OperatorOutstandingRewards),
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

func (k Keeper) SetAllOperatorAccumulatedCommission(
	ctx sdk.Context, allOperatorAccumulatedCommission []feedistributiontypes.KeyAndOperatorAccumulatedCommission,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixOperatorAccumulatedCommission, allOperatorAccumulatedCommission,
		func(item feedistributiontypes.KeyAndOperatorAccumulatedCommission) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndOperatorAccumulatedCommission) codec.ProtoMarshaler {
			return &item.OperatorAccumulatedCommission
		},
	)
}

func (k Keeper) GetAllOperatorAccumulatedCommission(ctx sdk.Context) ([]feedistributiontypes.KeyAndOperatorAccumulatedCommission, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixOperatorAccumulatedCommission,
		func() codec.ProtoMarshaler { return &feedistributiontypes.OperatorAccumulatedCommission{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndOperatorAccumulatedCommission {
			return feedistributiontypes.KeyAndOperatorAccumulatedCommission{
				Key:                           string(key),
				OperatorAccumulatedCommission: *value.(*feedistributiontypes.OperatorAccumulatedCommission),
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

func (k Keeper) SetAllStakerOutstandingRewards(
	ctx sdk.Context, allOperatorSlashEvent []feedistributiontypes.KeyAndStakerOutstandingRewards,
) error {
	return GenericSetAllItems(
		ctx, k,
		feedistributiontypes.KeyPrefixStakerOutstandingRewards, allOperatorSlashEvent,
		func(item feedistributiontypes.KeyAndStakerOutstandingRewards) []byte {
			return []byte(item.Key)
		},
		func(item feedistributiontypes.KeyAndStakerOutstandingRewards) codec.ProtoMarshaler {
			return &item.StakerOutstandingRewards
		},
	)
}

func (k Keeper) GetAllStakerOutstandingRewards(ctx sdk.Context) ([]feedistributiontypes.KeyAndStakerOutstandingRewards, error) {
	return GenericGetAllItems(
		ctx, k, feedistributiontypes.KeyPrefixStakerOutstandingRewards,
		func() codec.ProtoMarshaler { return &feedistributiontypes.StakerOutstandingRewards{} },
		func(key []byte, value codec.ProtoMarshaler) feedistributiontypes.KeyAndStakerOutstandingRewards {
			return feedistributiontypes.KeyAndStakerOutstandingRewards{
				Key:                      string(key),
				StakerOutstandingRewards: *value.(*feedistributiontypes.StakerOutstandingRewards),
			}
		},
	)
}
