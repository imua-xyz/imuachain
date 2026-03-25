package keeper

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/imua-xyz/imuachain/utils"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// interface guards
var (
	_ slashingtypes.StakingKeeper = Keeper{}
	_ evidencetypes.StakingKeeper = Keeper{}
	_ genutiltypes.StakingKeeper  = Keeper{}
	_ clienttypes.StakingKeeper   = Keeper{} // implemented in `validators.go`
	_ govtypes.StakingKeeper      = Keeper{}
)

// GetParams is an implementation of the staking interface expected by the SDK's evidence
// module. The module does not use it, but it is part of the interface.
func (k Keeper) GetParams(sdk.Context) stakingtypes.Params {
	return stakingtypes.Params{}
}

// IterateValidators is an implementation of the staking interface expected by the SDK's
// slashing module. The slashing module uses it at genesis to initialize its own state to save
// a look up from consensus address to pub key.
// Elsewhere, this function is used in invariants of the distribution and the SDK's staking
// modules, which we do not use in app.go.
func (k Keeper) IterateValidators(
	ctx sdk.Context,
	f func(index int64, validator stakingtypes.ValidatorI,
	) (stop bool),
) {
	// The SDK iterates over these validators in the operator address order first, so we
	// determine the order first.
	validators := make([]stakingtypes.ValidatorI, 0)
	k.IterateBondedValidatorsByPower(ctx, func(_ int64, v stakingtypes.ValidatorI) bool {
		validators = append(validators, v)
		return false
	})
	// we are guaranteed to have a unique operator address
	sort.SliceStable(validators, func(i, j int) bool {
		return bytes.Compare(validators[i].GetOperator(), validators[j].GetOperator()) < 0
	})
	for i, v := range validators {
		if f(int64(i), v) {
			break
		}
	}
}

// Validator is an implementation of the staking interface expected by the SDK's
// slashing module. The slashing module uses it to obtain a validator's information upon
// its addition to the list of validators, and then to unjail a validator. During the addition
// it stores a reverse lookup from consensus address to pub key, which is why we need the
// pub key to be set in this call.
func (k Keeper) Validator(ctx sdk.Context, address sdk.ValAddress) stakingtypes.ValidatorI {
	val, found := k.GetValidator(ctx, address)
	if !found {
		return nil
	}
	return val
}

// ValidatorByConsAddr is an implementation of the staking interface expected by the SDK's
// slashing and evidence modules.
// The slashing module calls this function when it observes downtime. The only requirement on
// the returned value is that it isn't nil, and the jailed status is accurately set (to prevent
// re-jailing of the same operator).
// The evidence module calls this function when it handles equivocation evidence. The returned
// value must not be nil and must not have an UNBONDED validator status (the default is
// unspecified), or evidence will reject it.
func (k Keeper) ValidatorByConsAddr(
	ctx sdk.Context,
	addr sdk.ConsAddress,
) stakingtypes.ValidatorI {
	// this validator has the following items initialized:
	// jailed status, operator address, bonded status == unspecified,
	// consensus public key.
	// the operator address is used by our EVM module, and its presence triggers
	// a call to Validator(ctx, addr) in the slashing module, which is implemented in this file.
	// after that call, the ConsPubKey is fetched, which is also set by the below call.
	val, found := k.operatorKeeper.ValidatorByConsAddrForChainID(
		ctx, addr, utils.ChainIDWithoutRevision(ctx.ChainID()),
	)
	if !found {
		return nil
	}
	// TODO: alter the status of the validator?
	return val
}

// Slash is an implementation of the staking interface expected by the SDK's slashing module.
// It forwards the call to SlashWithInfractionReason with Infraction_INFRACTION_UNSPECIFIED.
// It is not called within the slashing module, but is part of the interface.
func (k Keeper) Slash(
	ctx sdk.Context, addr sdk.ConsAddress,
	infractionHeight, power int64,
	slashFactor sdk.Dec,
) math.Int {
	return k.SlashWithInfractionReason(
		ctx, addr, infractionHeight, power,
		slashFactor, stakingtypes.Infraction_INFRACTION_UNSPECIFIED,
	)
}

// SlashWithInfractionReason is an implementation of the staking interface expected by the
// SDK's slashing module. It is called when the slashing module observes an infraction
// of either downtime or equivocation (which is via the evidence module).
func (k Keeper) SlashWithInfractionReason(
	ctx sdk.Context, addr sdk.ConsAddress, infractionHeight, power int64,
	slashFactor sdk.Dec, infraction stakingtypes.Infraction,
) math.Int {
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	found, accAddress := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		ctx, chainIDWithoutRevision, addr,
	)
	if !found {
		// TODO(mm): already slashed and removed from the set?
		return math.NewInt(0)
	}

	slashingOldKey := false
	var currentConsAddr sdk.ConsAddress
	found, currentKey, err := k.operatorKeeper.GetOperatorConsKeyForChainID(
		ctx, accAddress, chainIDWithoutRevision,
	)
	if err == nil {
		if found {
			currentConsAddr = currentKey.ToConsAddr()
			// the current key is different from the one being slashed
			// we should save the post slash status into the new key as well.
			slashingOldKey = !bytes.Equal(currentConsAddr, addr)
		}
	} else {
		// the two errors returned by GetOperatorConsKeyForChainID are
		// 1. delegationtypes.ErrOperatorNotExist, impossible because an operator address
		// is returned above by GetOperatorAddressForChainIDAndConsAddr
		// 2. types.ErrUnknownChainID, also impossible because the chain ID is
		// already validated above by GetOperatorAddressForChainIDAndConsAddr
		// both of these can be considered a terrible violation, and thus,
		// the only way out is to panic.
		panic(
			fmt.Sprintf(
				"Logic error: failed to get operator cons key: %s %s %v",
				addr.String(), chainIDWithoutRevision, err,
			),
		)
	}
	res := k.operatorKeeper.SlashWithInfractionReason(
		ctx, accAddress, infractionHeight,
		power, slashFactor, infraction,
	)
	// copy over to new consensus key
	if slashingOldKey {
		// copy from old key to new key
		// the slashing keeper tombstones after calling this function. hence,
		// copy directly will not succeed. we must do it ourselves.
		k.CopyValidatorSigningInfo(
			ctx, addr, currentConsAddr, infraction,
		)
	}
	if infraction == stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN {
		// Permanently freeze the operator globally
		if err := k.operatorKeeper.FreezeOperator(
			ctx, accAddress,
		); err != nil {
			panic(
				fmt.Sprintf(
					"Logic error: failed to freeze operator %s: %v",
					accAddress.String(), err,
				),
			)
		}
	}
	return res
}

// CopyValidatorSigningInfo copies a validator's signing info from old consensus address
// to new consensus address. If `markTombstone` is true, the address is tombstoned and
// jailed forever.
func (k Keeper) CopyValidatorSigningInfo(
	ctx sdk.Context,
	oldConsAddr sdk.ConsAddress,
	newConsAddr sdk.ConsAddress,
	infraction stakingtypes.Infraction,
) {
	info, found := k.slashingKeeper.GetValidatorSigningInfo(
		ctx, oldConsAddr,
	)
	if found {
		info = slashingtypes.NewValidatorSigningInfo(
			newConsAddr,
			info.StartHeight,
			info.IndexOffset,
			info.JailedUntil,
			info.Tombstoned,
			info.MissedBlocksCounter,
		)
	} else {
		// initialize a default value
		info = slashingtypes.NewValidatorSigningInfo(
			newConsAddr,
			ctx.BlockHeight(),
			0,
			time.Unix(0, 0),
			false,
			0,
		)
	}
	switch infraction {
	case stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN:
		// permanently block the new key in addition to the old key,
		// which is done a few lines after by the calling function in
		// x/slashing
		info.Tombstoned = true
		info.JailedUntil = evidencetypes.DoubleSignJailEndTime
	case stakingtypes.Infraction_INFRACTION_DOWNTIME:
		// jail the new key
		info.JailedUntil = ctx.BlockTime().Add(
			k.slashingKeeper.DowntimeJailDuration(ctx),
		)
		// reset so the operator doesn't get slashed immediately upon unjail.
		// x/slashing does this for the old key a few lines after calling this
		// function.
		info.MissedBlocksCounter = 0
		info.IndexOffset = 0
		k.slashingKeeper.ClearValidatorMissedBlockBitArray(
			ctx, newConsAddr,
		)
	// called by hook, not by SlashWithInfractionReason
	case stakingtypes.Infraction_INFRACTION_UNSPECIFIED:
		// it is permitted for a validator to change from key A
		// to key B back to key A again over a long period of time.
		// so, the new key may have an outdated bit array.
		// set the new key to a fresh slate, and then...
		k.slashingKeeper.ClearValidatorMissedBlockBitArray(
			ctx, newConsAddr,
		)
		// ...transplant the debt of the old key over.
		// placement does not matter as much for us; this helps
		// reduce the size of the loop from O(window) to
		// O(MissedBlocksCounter)
		for i := int64(0); i < info.MissedBlocksCounter; i++ {
			// even though this loop is being called many times,
			// the cost for db write is being paid by the tx sender
			// so it is acceptable to us as sybil attacks are avoided
			k.slashingKeeper.SetValidatorMissedBlockBitArray(
				ctx, newConsAddr, i, true,
			)
		}
		// then, we ask x/slashing to count from offset 0 again,
		// where it will observe that [0, missedBlocksCounter)
		// are missed blocks.
		info.IndexOffset = 0
		// there are two ways to game this currently, which, cannot
		// be solved without design changes in x/slashing
		// 1. a downtime of 4,000 is copied from old key to new key.
		// old key stops signing, goes to 10,000 and is slashed and
		// jailed. unjail duration is completed, new key is unjailed.
		// epoch changes, new key can be active but it has 4,000 missed.
		// old key, which was jailed, had 0 missed. wrong, even new
		// key will have 0 missed because of the line above.
		// 2. a downtime of 4,000 spread across the window is recovered
		// miraculously within the first 4,000 blocks by being active.
		// i don't see this as an issue.
	}
	k.slashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, info)
}

// Jail is an implementation of the staking interface expected by the SDK's slashing module.
// It delegates the call to the operator module. Alternatively, this may be handled
// by the slashing module depending upon the design decisions.
func (k Keeper) Jail(ctx sdk.Context, addr sdk.ConsAddress) {
	// once jailed, the operator module runs a hook, which lets concerned modules,
	// such as this one, that the operator can be removed from the validator set.
	k.operatorKeeper.Jail(ctx, addr, utils.ChainIDWithoutRevision(ctx.ChainID()))
}

// Unjail is an implementation of the staking interface expected by the SDK's slashing module.
// The function is called by the slashing module only when it receives a request from the
// operator to do so.
func (k Keeper) Unjail(ctx sdk.Context, addr sdk.ConsAddress) {
	chainID := utils.ChainIDWithoutRevision(ctx.ChainID())
	// Find the operator tied to this consAddr
	found, accAddr := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(ctx, chainID, addr)
	if found && k.operatorKeeper.IsOperatorFrozen(ctx, accAddr) {
		// Operator is permanently frozen, silently ignore the unjail request
		// or log it. We cannot return an error because the SDK's StakingKeeper interface doesn't allow it.
		k.Logger(ctx).Info("blocked attempt to unjail a permanently frozen operator", "operator", accAddr)
		return
	}
	k.operatorKeeper.Unjail(ctx, addr, chainID)
}

// Delegation is an implementation of the staking interface expected by the SDK's slashing
// module. The slashing module uses it to obtain the delegation information of a validator
// before unjailing it. If the slashing module's unjail function is never called, this
// function will never be called either.
// NOTE: this is not a universal function, it not actually get delegation for
// {delegator, validator}, but only returns {validator}'s self delegation.
func (k Keeper) Delegation(
	ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress,
) stakingtypes.DelegationI {
	// This interface is only used for unjail to retrieve the self delegation value,
	// so the delegator and validator are the same.
	operator := delegator
	avsAddr := utils.GenerateAVSAddress(
		utils.ChainIDWithoutRevision(ctx.ChainID()),
	)
	operatorUSDValues, err := k.operatorKeeper.GetOrCalculateOperatorUSDValues(
		ctx, operator, avsAddr,
	)
	if err != nil {
		k.Logger(ctx).Error(
			"Delegation: failed to get or calculate the operator USD values",
			"operator", operator.String(),
			"chainID", ctx.ChainID(),
			"error", err,
		)
		return nil
	}
	return stakingtypes.Delegation{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
		Shares: sdk.TokensFromConsensusPower(
			operatorUSDValues.SelfUSDValue.TruncateInt64(),
			sdk.DefaultPowerReduction,
		).ToLegacyDec(),
	}
}

// MaxValidators is an implementation of the staking interface expected by the SDK's slashing
// module. It is not called within the slashing module, but is part of the interface.
// It returns the maximum number of validators allowed in the network.
func (k Keeper) MaxValidators(ctx sdk.Context) uint32 {
	return k.GetMaxValidators(ctx)
}

// GetAllValidators is an implementation of the staking interface expected by the SDK's
// slashing module. It is not called within the slashing module, but is part of the interface.
// Hence, it is not implemented meaningfully.
func (k Keeper) GetAllValidators(sdk.Context) (validators []stakingtypes.Validator) {
	return []stakingtypes.Validator{}
}

// IsValidatorJailed is an implementation of the staking interface expected by the SDK's
// slashing module. It is called by the slashing module to record validator signatures
// for downtime tracking. We delegate the call to the operator keeper.
func (k Keeper) IsValidatorJailed(ctx sdk.Context, addr sdk.ConsAddress) bool {
	return k.operatorKeeper.IsOperatorJailedForChainID(ctx, addr, utils.ChainIDWithoutRevision(ctx.ChainID()))
}

// ApplyAndReturnValidatorSetUpdates is an implementation of the staking interface expected
// by the SDK's genutil module. It is used in the gentx command, which we do not need to
// support. So this function does nothing.
func (k Keeper) ApplyAndReturnValidatorSetUpdates(
	sdk.Context,
) (updates []abci.ValidatorUpdate, err error) {
	return
}

// IterateBondedValidatorsByPower is an implementation of the staking interface expected by
// the SDK's gov module and by our oracle module. It iterates only over the bonded, that is,
// currently active validators, sorted by power, from highest to lowest.
func (k Keeper) IterateBondedValidatorsByPower(
	ctx sdk.Context, f func(int64, stakingtypes.ValidatorI) (stop bool),
) {
	// this is the bonded validators, that is, those that are currently in this module.
	prevList := k.GetAllImuachainValidators(ctx)
	sort.SliceStable(prevList, func(i, j int) bool {
		return prevList[i].Power > prevList[j].Power
	})
	for i, v := range prevList {
		pk, err := v.ConsPubKey()
		if err != nil {
			ctx.Logger().Error("Failed to deserialize public key; skipping", "error", err, "i", i)
			continue
		}
		val, found := k.operatorKeeper.ValidatorByConsAddrForChainID(
			ctx, sdk.GetConsAddress(pk), utils.ChainIDWithoutRevision(ctx.ChainID()),
		)
		if !found {
			ctx.Logger().Error("Operator address not found; skipping", "consAddress", sdk.GetConsAddress(pk), "i", i)
			continue
		}
		// the voting power is fetched from this module and not the operator module
		// because it is applied at the end of an epoch, whereas that from the operator
		// module is more recent.
		val.Tokens = sdk.TokensFromConsensusPower(v.Power, sdk.DefaultPowerReduction)
		// since the validator object was fetched from this module, we should set it to bonded.
		val.Status = stakingtypes.Bonded
		// items passed are:
		// jailed status, tokens quantity, operator address as ValAddress, bonded status
		if f(int64(i), val) {
			break
		}
	}
}

// TotalBondedTokens is an implementation of the staking interface expected by the SDK's
// gov module. This is not implemented intentionally, since the tokens securing this chain
// are many and span across multiple chains and assets.
func (k Keeper) TotalBondedTokens(ctx sdk.Context) math.Int {
	// TODO: return totalBondedPower(virtual tokens from power) compatible with multi-assets staking
	totalPower := math.ZeroInt()
	k.IterateBondedValidatorsByPower(ctx, func(_ int64, v stakingtypes.ValidatorI) bool {
		totalPower = totalPower.Add(v.GetTokens())
		return false
	})
	return totalPower
}

// IterateDelegations is an implementation of the staking interface expected by the SDK's
// gov module. See note above to understand why this is not implemented.
func (k Keeper) IterateDelegations(
	ctx sdk.Context, _ sdk.AccAddress,
	_ func(int64, stakingtypes.DelegationI) bool,
) {
	// for now we don't have mechabnism to bond delegatorAddress from clientChain to their cosmossdk address(for EVM when can force user use the same ethsecp256k1 instead of secp256k1 to retrieve default bonded addresses, but that not a universal approach), we'll just ignore any delegator's vote, and validator will have the total power including all the delegated assts, and that's reasonanble
	ctx.Logger().Info("IterateDelegations from gov tally will just return, which means operator/validator will have all the powers delegated to them when voting proposal")
}
