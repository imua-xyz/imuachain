package common

import (
	"time"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	dogfoodkeeper "github.com/imua-xyz/imuachain/x/dogfood/keeper"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

type Price struct {
	Value   sdkmath.Int
	Decimal uint8
}
type SlashingKeeper interface {
	JailUntil(sdk.Context, sdk.ConsAddress, time.Time)
}
type KeeperOracle interface {
	KeeperDogfood
	SlashingKeeper

	Logger(ctx sdk.Context) log.Logger
	AddZeroNonceItemWithFeederIDsForValidators(ctx sdk.Context, feederIDs []uint64, validators []string)
	InitValidatorReportInfo(ctx sdk.Context, validator string, height int64)
	ClearAllValidatorReportInfo(ctx sdk.Context)
	ClearAllValidatorMissedRoundBitArray(ctx sdk.Context)
	GrowRoundID(ctx sdk.Context, tokenID, nextRoundID uint64) (price string, roundID uint64)
	AppendPriceTR(ctx sdk.Context, tokenID uint64, priceTR types.PriceTimeRound, detID string) bool
	GetValidatorReportInfo(ctx sdk.Context, validator string) (info types.ValidatorReportInfo, found bool)
	GetMaliciousJailDuration(ctx sdk.Context) (res time.Duration)
	ClearValidatorMissedRoundBitArray(ctx sdk.Context, validator string)
	GetReportedRoundsWindow(ctx sdk.Context) int64
	GetValidatorMissedRoundBitArray(ctx sdk.Context, validator string, index uint64) bool
	SetValidatorMissedRoundBitArray(ctx sdk.Context, validator string, index uint64, missed bool)
	GetMinReportedPerWindow(ctx sdk.Context) int64
	GetMissJailDuration(ctx sdk.Context) (res time.Duration)
	SetValidatorReportInfo(ctx sdk.Context, validator string, info types.ValidatorReportInfo)
	GetSlashFractionMalicious(ctx sdk.Context) (res sdk.Dec)
	SetValidatorUpdateForCache(sdk.Context, types.ValidatorUpdateBlock)
	SetParamsForCache(sdk.Context, types.RecentParams)
	SetMsgItemsForCache(sdk.Context, types.RecentMsg)
	GetRecentParamsWithinMaxNonce(ctx sdk.Context) (recentParamsList []*types.RecentParams, prev, latest types.RecentParams)
	GetAllRecentMsg(ctx sdk.Context) (list []types.RecentMsg)
	GetParams(sdk.Context) types.Params
	GetIndexRecentMsg(sdk.Context) (types.IndexRecentMsg, bool)
	GetAllRecentMsgAsMap(sdk.Context) map[int64][]*types.MsgItem

	GetIndexRecentParams(sdk.Context) (types.IndexRecentParams, bool)
	GetAllRecentParamsAsMap(sdk.Context) map[int64]*types.Params

	GetValidatorUpdateBlock(sdk.Context) (types.ValidatorUpdateBlock, bool)

	SetIndexRecentMsg(sdk.Context, types.IndexRecentMsg)
	SetRecentMsg(sdk.Context, types.RecentMsg)

	SetIndexRecentParams(sdk.Context, types.IndexRecentParams)
	SetRecentParams(sdk.Context, types.RecentParams)

	RemoveRecentParams(sdk.Context, uint64)
	RemoveRecentMsg(sdk.Context, uint64)

	RemoveNonceWithValidator(ctx sdk.Context, validator string)
	RemoveNonceWithFeederIDsForValidators(ctx sdk.Context, feederIDs []uint64, validators []string)
	RemoveNonceWithFeederIDsForAll(ctx sdk.Context, feederID []uint64)

	SetNonce(ctx sdk.Context, nonce types.ValidatorNonce)
	GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (types.Price, error)
	GetMultipleAssetsPrices(ctx sdk.Context, assetIDs map[string]interface{}) (map[string]types.Price, error)
}

var _ KeeperDogfood = dogfoodkeeper.Keeper{}

type KeeperDogfood = interface {
	GetLastTotalPower(ctx sdk.Context) sdkmath.Int
	IterateBondedValidatorsByPower(ctx sdk.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool))
	GetValidatorUpdates(ctx sdk.Context) []abci.ValidatorUpdate
	GetValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, found bool)

	GetAllImuachainValidators(ctx sdk.Context) (validators []dogfoodtypes.ImuachainValidator)
	ValidatorByConsAddr(ctx sdk.Context, addr sdk.ConsAddress) stakingtypes.ValidatorI
	SlashWithInfractionReason(ctx sdk.Context, addr sdk.ConsAddress, infractionHeight, power int64, slashFactor sdk.Dec, infraction stakingtypes.Infraction) sdkmath.Int
	Jail(ctx sdk.Context, addr sdk.ConsAddress)
}
