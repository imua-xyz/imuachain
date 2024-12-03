package common

import (
	sdkmath "cosmossdk.io/math"
	dogfoodkeeper "github.com/ExocoreNetwork/exocore/x/dogfood/keeper"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Price struct {
	Value   sdkmath.Int
	Decimal uint8
}

type KeeperOracle interface {
	KeeperDogfood

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

	SetValidatorUpdateBlock(sdk.Context, types.ValidatorUpdateBlock)

	RemoveRecentParams(sdk.Context, uint64)
	RemoveRecentMsg(sdk.Context, uint64)

	RemoveNonceWithValidator(ctx sdk.Context, validator string)
	RemoveNonceWithValidatorAndFeederID(ctx sdk.Context, validator string, feederID uint64) bool
	RemoveNonceWithFeederIDForValidators(ctx sdk.Context, feederID uint64, validators []string)
	RemoveNonceWithFeederIDForAll(ctx sdk.Context, feederID uint64)

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

	GetAllExocoreValidators(ctx sdk.Context) (validators []dogfoodtypes.ExocoreValidator)
	ValidatorByConsAddr(ctx sdk.Context, addr sdk.ConsAddress) stakingtypes.ValidatorI
	SlashWithInfractionReason(ctx sdk.Context, addr sdk.ConsAddress, infractionHeight, power int64, slashFactor sdk.Dec, infraction stakingtypes.Infraction) sdkmath.Int
	Jail(ctx sdk.Context, addr sdk.ConsAddress)
}
