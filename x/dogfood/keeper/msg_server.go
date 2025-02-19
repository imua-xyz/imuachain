package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/errors"
	avskeeper "github.com/imua-xyz/imuachain/x/avs/keeper"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/evmos/evmos/v16/x/epochs/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams is used to trigger a params update.
// TODO: It must be signed by the authority.
func (k Keeper) UpdateParams(
	ctx context.Context, msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	// if k.authority != msg.Authority {
	// 	return nil, errorsmod.Wrapf(
	// 		govtypes.ErrInvalidSigner,
	// 		"invalid authority; expected %s, got %s",
	// 		k.authority, msg.Authority,
	// 	)
	// }
	prevParams := k.GetDogfoodParams(c)
	nextParams := msg.Params
	logger := k.Logger(c)
	if nextParams.EpochsUntilUnbonded == 0 {
		logger.Info(
			"UpdateParams",
			"overriding EpochsUntilUnbonded with value", prevParams.EpochsUntilUnbonded,
		)
		nextParams.EpochsUntilUnbonded = prevParams.EpochsUntilUnbonded
	}
	if nextParams.MaxValidators == 0 {
		logger.Info(
			"UpdateParams",
			"overriding MaxValidators with value", prevParams.MaxValidators,
		)
		nextParams.MaxValidators = prevParams.MaxValidators
	}
	if err := epochstypes.ValidateEpochIdentifierInterface(
		nextParams.EpochIdentifier,
	); err != nil {
		logger.Info(
			"UpdateParams",
			"overriding EpochIdentifier with value", prevParams.EpochIdentifier,
		)
		nextParams.EpochIdentifier = prevParams.EpochIdentifier
	}
	if nextParams.HistoricalEntries == 0 {
		logger.Info(
			"UpdateParams",
			"overriding HistoricalEntries with value", prevParams.HistoricalEntries,
		)
		nextParams.HistoricalEntries = prevParams.HistoricalEntries
	}
	if len(nextParams.AssetIDs) == 0 {
		logger.Info(
			"UpdateParams",
			"overriding AssetIDs with value", prevParams.AssetIDs,
		)
		nextParams.AssetIDs = prevParams.AssetIDs
	}
	if nextParams.MinSelfDelegation.IsNil() || nextParams.MinSelfDelegation.IsNegative() {
		logger.Info(
			"UpdateParams",
			"overriding MinSelfDelegation with value", prevParams.MinSelfDelegation,
		)
		nextParams.MinSelfDelegation = prevParams.MinSelfDelegation
	}
	// now do stateful validations
	if _, found := k.epochsKeeper.GetEpochInfo(c, nextParams.EpochIdentifier); !found {
		logger.Info(
			"UpdateParams",
			"overriding EpochIdentifier with value", prevParams.EpochIdentifier,
		)
		nextParams.EpochIdentifier = prevParams.EpochIdentifier
	}
	override := false
	for _, assetID := range nextParams.AssetIDs {
		if !k.restakingKeeper.IsStakingAsset(c, strings.ToLower(assetID)) {
			override = true
			break
		}
	}
	if override {
		logger.Info(
			"UpdateParams",
			"overriding AssetIDs with value", prevParams.AssetIDs,
		)
		nextParams.AssetIDs = prevParams.AssetIDs
	}
	k.SetParams(c, nextParams)

	// update the related info in the AVS module
	isAVS, avsAddr := k.avsKeeper.IsAVSByChainID(c, avstypes.ChainIDWithoutRevision(c.ChainID()))
	if !isAVS {
		return nil, errors.Wrapf(types.ErrNotAVSByChainID, "chainID:%s avsAddr:%s", c.ChainID(), avsAddr)
	}
	err := k.avsKeeper.UpdateAVSInfo(c, &avstypes.AVSRegisterOrDeregisterParams{
		AvsName:           c.ChainID(),
		AvsAddress:        avsAddr,
		AssetID:           nextParams.AssetIDs,
		UnbondingPeriod:   uint64(nextParams.EpochsUntilUnbonded),
		MinSelfDelegation: nextParams.MinSelfDelegation.Uint64(),
		EpochIdentifier:   nextParams.EpochIdentifier,
		ChainID:           c.ChainID(),
		Action:            avskeeper.UpdateAction,
	})
	if err != nil {
		return nil, errors.Wrap(types.ErrUpdateAVSInfo, err.Error())
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
