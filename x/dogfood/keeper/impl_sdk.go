package keeper

import (
	"sort"

	"cosmossdk.io/math"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
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
// slashing module. The slashing module uses it for two purposes: once at genesis to
// store a mapping of pub key to cons address (which is done by our operator module),
// and then during the invariants check to ensure that the total delegated amount
// matches that of each validator. Ideally, this invariant should be implemented
// by the delegation and/or deposit module(s) instead.
func (k Keeper) IterateValidators(sdk.Context,
	func(index int64, validator stakingtypes.ValidatorI) (stop bool),
) {
	// not intentionally implemented, since unused by the importing modules
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
		ctx, addr, avstypes.ChainIDWithoutRevision(ctx.ChainID()),
	)
	if !found {
		return nil
	}
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
	found, accAddress := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		ctx, avstypes.ChainIDWithoutRevision(ctx.ChainID()), addr,
	)
	if !found {
		// TODO(mm): already slashed and removed from the set?
		return math.NewInt(0)
	}
	// TODO(mm): add list of assets to be slashed (and not just all of them).
	// based on yet to be finalized slashing design.
	return k.operatorKeeper.SlashWithInfractionReason(
		ctx, accAddress, infractionHeight,
		power, slashFactor, infraction,
	)
}

// Jail is an implementation of the staking interface expected by the SDK's slashing module.
// It delegates the call to the operator module. Alternatively, this may be handled
// by the slashing module depending upon the design decisions.
func (k Keeper) Jail(ctx sdk.Context, addr sdk.ConsAddress) {
	k.operatorKeeper.Jail(ctx, addr, avstypes.ChainIDWithoutRevision(ctx.ChainID()))
	// TODO(mm)
	// once the operator module jails someone, a hook should be triggered
	// and the validator removed from the set. same for unjailing.
}

// Unjail is an implementation of the staking interface expected by the SDK's slashing module.
// The function is called by the slashing module only when it receives a request from the
// operator to do so. TODO(mm): We need to use the SDK's slashing module to allow for downtime
// slashing but somehow we need to prevent its Unjail function from being called by anyone.
func (k Keeper) Unjail(ctx sdk.Context, addr sdk.ConsAddress) {
	k.operatorKeeper.Unjail(ctx, addr, avstypes.ChainIDWithoutRevision(ctx.ChainID()))
}

// Delegation is an implementation of the staking interface expected by the SDK's slashing
// module. The slashing module uses it to obtain the delegation information of a validator
// before unjailing it. If the slashing module's unjail function is never called, this
// function will never be called either.
// NOTE: this is not a universal function, it not actually get delegation for {delegator, validator}, but only returns {validator}'s self delegation, only suites for special invoke
func (k Keeper) Delegation(
	ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress,
) stakingtypes.DelegationI {
	// This interface is only used for unjail to retrieve the self delegation value,
	// so the delegator and validator are the same.
	operator := delegator
	avsAddr := avstypes.GenerateAVSAddr(avstypes.ChainIDWithoutRevision(ctx.ChainID()))
	operatorUSDValues, err := k.operatorKeeper.GetOrCalculateOperatorUSDValues(ctx, operator, avsAddr)
	if err != nil {
		k.Logger(ctx).Error("Delegation: failed to get or calculate the operator USD values", "operator", operator.String(), "chainID", ctx.ChainID(), "error", err)
		return nil
	}
	return stakingtypes.Delegation{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
		Shares:           sdk.TokensFromConsensusPower(operatorUSDValues.SelfUSDValue.TruncateInt64(), sdk.DefaultPowerReduction).ToLegacyDec(),
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
	return k.operatorKeeper.IsOperatorJailedForChainID(ctx, addr, avstypes.ChainIDWithoutRevision(ctx.ChainID()))
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
	prevList := k.GetAllExocoreValidators(ctx)
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
			ctx, sdk.GetConsAddress(pk), avstypes.ChainIDWithoutRevision(ctx.ChainID()),
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
