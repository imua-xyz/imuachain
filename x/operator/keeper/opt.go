package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AssetPriceAndDecimal struct {
	Price        sdkmath.Int
	PriceDecimal uint8
	Decimal      uint32
}

// OptIn call this function to opt in to an AVS.
// The caller must ensure that the operatorAddress passed is valid.
func (k *Keeper) OptIn(
	ctx sdk.Context, operatorAddress sdk.AccAddress, avsAddr string,
) error {
	// check that the operator is registered
	if !k.IsOperator(ctx, operatorAddress) {
		return errorsmod.Wrapf(delegationtypes.ErrOperatorNotExist, "operator is :%s", operatorAddress)
	}
	// check that the AVS is registered
	if isAvs, _ := k.avsKeeper.IsAVS(ctx, avsAddr); !isAvs {
		return types.ErrNoSuchAvs.Wrapf("AVS not found %s", avsAddr)
	}
	// check optedIn info
	if k.IsOptedIn(ctx, operatorAddress.String(), avsAddr) {
		return types.ErrAlreadyOptedIn
	}
	// Check if the USD value of the operator is greater than or equal to the self-delegation
	// configured by the AVS. This is used to prevent a DDOS attack from zero-USD value opting in.
	operatorUSDValues, err := k.GetOrCalculateOperatorUSDValues(ctx, operatorAddress, avsAddr)
	if err != nil {
		return errorsmod.Wrapf(err, "OptIn: error when calculating operator USD value, operator:%s avsAddr:%s", operatorAddress.String(), avsAddr)
	}
	minSelfDelegation, err := k.avsKeeper.GetAVSMinimumSelfDelegation(ctx, avsAddr)
	if err != nil {
		return errorsmod.Wrapf(err, "OptIn: error when getting minimum self delegation of AVS, avsAddr:%s", avsAddr)
	}
	if operatorUSDValues.SelfUSDValue.LT(minSelfDelegation) {
		return errorsmod.Wrapf(types.ErrMinDelegationNotMet, "operator:%s avs:%s selfUSDValue:%s minSelfDelegation:%s", operatorAddress.String(), avsAddr, operatorUSDValues.SelfUSDValue, minSelfDelegation)
	}

	// do not allow frozen operators to do anything meaningful
	if k.slashKeeper.IsOperatorFrozen(ctx, operatorAddress) {
		return delegationtypes.ErrOperatorIsFrozen
	}

	// call InitOperatorUSDValue to mark the operator has been opted into the AVS
	// but the actual voting power calculation and update will be performed at the
	// end of epoch of the AVS. So there isn't any reward in the opted-in epoch for the
	// operator
	err = k.InitOperatorUSDValue(ctx, avsAddr, operatorAddress.String())
	if err != nil {
		return err
	}

	// update opted-in info
	slashContract, err := k.avsKeeper.GetAVSSlashContract(ctx, avsAddr)
	if err != nil {
		return err
	}
	optedInfo := &types.OptedInfo{
		SlashContract: slashContract,
		// #nosec G701
		OptedInHeight:  uint64(ctx.BlockHeight()),
		OptedOutHeight: types.DefaultOptedOutHeight,
	}
	err = k.SetOptedInfo(ctx, operatorAddress.String(), avsAddr, optedInfo)
	if err != nil {
		return err
	}
	return nil
}

// OptInWithConsKey is a wrapper function to call OptIn and then SetOperatorConsKeyForChainID.
// The caller must ensure that the operatorAddress passed is valid and that the AVS is a chain-type AVS.
func (k Keeper) OptInWithConsKey(
	ctx sdk.Context, operatorAddress sdk.AccAddress, avsAddr string, key keytypes.WrappedConsKey,
) error {
	err := k.OptIn(ctx, operatorAddress, avsAddr)
	if err != nil {
		return err
	}
	chainID, _ := k.avsKeeper.GetChainIDByAVSAddr(ctx, avsAddr)
	k.Logger(ctx).Info("OptInWithConsKey", "chainID", chainID)
	return k.SetOperatorConsKeyForChainID(ctx, operatorAddress, chainID, key)
}

// OptOut call this function to opt out of AVS
func (k *Keeper) OptOut(ctx sdk.Context, operatorAddress sdk.AccAddress, avsAddr string) (err error) {
	// check that the operator is registered
	if !k.IsOperator(ctx, operatorAddress) {
		return delegationtypes.ErrOperatorNotExist
	}
	// check that the AVS is registered
	if isAvs, _ := k.avsKeeper.IsAVS(ctx, avsAddr); !isAvs {
		return types.ErrNoSuchAvs.Wrapf("AVS not found %s", avsAddr)
	}
	// check if the operator is active. It's not allowed to opt-out if the operator
	// isn't opted-in or is jailed.
	if !k.IsActive(ctx, operatorAddress, avsAddr) {
		return types.ErrNotOptedIn
	}
	// do not allow frozen operators to do anything meaningful
	if k.slashKeeper.IsOperatorFrozen(ctx, operatorAddress) {
		return delegationtypes.ErrOperatorIsFrozen
	}
	// check if it is the chain-type AVS
	chainIDWithoutRevision, isChainAvs := k.avsKeeper.GetChainIDByAVSAddr(ctx, avsAddr)
	// set up the deferred function to remove key and write cache
	defer func() {
		if err == nil && isChainAvs {
			// store.Delete... doesn't fail
			k.InitiateOperatorKeyRemovalForChainID(ctx, operatorAddress, chainIDWithoutRevision)
		}
	}()

	// DeleteOperatorUSDValue, delete the operator voting power, it can facilitate to
	// update the voting powers of all opted-in operators at the end of epoch.
	// There might still be a reward for the operator in this opted-out epoch,
	// which is determined by the reward logic.
	// #nosec G703 // already validated that operatorAddress is not ""
	_ = k.DeleteOperatorUSDValue(ctx, avsAddr, operatorAddress.String())
	if err != nil {
		return err
	}

	// set opted-out height
	handleFunc := func(info *types.OptedInfo) {
		// #nosec G701
		info.OptedOutHeight = uint64(ctx.BlockHeight())
		// the opt out, although is requested now, is made effective at the end of the current epoch.
		// so this is not necessarily the OptedOutHeight, rather, it is the OptOutRequestHeight.
		// the height is not directly used, beyond ascertaining whether the operator is currently opted in/out.
		// so the difference due to the epoch scheduling is not too big a concern.
	}
	err = k.HandleOptedInfo(ctx, operatorAddress.String(), avsAddr, handleFunc)
	if err != nil {
		return err
	}

	// mark opt out in the snapshot helper
	err = k.SetOptOutFlag(ctx, avsAddr, true)
	if err != nil {
		return err
	}
	return nil
}
