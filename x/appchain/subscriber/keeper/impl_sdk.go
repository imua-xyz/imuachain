package keeper

import (
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// This file contains the implementations of the Comos SDK level expected keepers
// for the subscriber's Keeper. This allows us to use the subscriber's keeper
// as an input into the slashing and the evidence modules. These modules then
// handle the slashing calls so that we do not have to implement them separately.
// Note that the subscriber chain is deemed to be trusted because the coordinator
// will not verify the evidence any further. An upgrade overcoming this has just
// been merged into interchain-security, which we can pick up later.
// https://github.com/orgs/cosmos/projects/28/views/11?pane=issue&itemId=21248976

// interface guards
var (
	_ slashingtypes.StakingKeeper = Keeper{}
	_ evidencetypes.StakingKeeper = Keeper{}
	_ clienttypes.StakingKeeper   = Keeper{}
	_ genutiltypes.StakingKeeper  = Keeper{}
	_ minttypes.StakingKeeper     = Keeper{}
)

// GetParams returns an empty staking params. It is used by the interfaces above, but the returned
// value is never examined.
func (k Keeper) GetParams(sdk.Context) stakingtypes.Params {
	return stakingtypes.Params{}
}

// This function is used by the slashing module to store the validator public keys into the
// state. These were previously verified in the evidence module but have since been removed.
func (k Keeper) IterateValidators(sdk.Context,
	func(int64, stakingtypes.ValidatorI) bool,
) {
	// no op
}

// simply unimplemented because it is not needed
func (k Keeper) Validator(sdk.Context, sdk.ValAddress) stakingtypes.ValidatorI {
	panic("unimplemented on this keeper")
}

// ValidatorByConsAddr returns an empty validator
func (k Keeper) ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI {
	/*
		NOTE:

		The evidence module will call this function when it handles equivocation evidence.
		The returned value must not be nil and must not have an UNBONDED validator status,
		or evidence will reject it.

		Also, the slashing module will call this function when it observes downtime. In that case
		the only requirement on the returned value is that it isn't null.
	*/
	return stakingtypes.Validator{}
}

// Calls SlashWithInfractionReason with Infraction_INFRACTION_UNSPECIFIED.
// It should not be called anywhere.
func (k Keeper) Slash(
	ctx sdk.Context,
	addr sdk.ConsAddress,
	infractionHeight, power int64,
	slashFactor sdk.Dec,
) math.Int {
	return k.SlashWithInfractionReason(
		ctx, addr, infractionHeight,
		power, slashFactor,
		stakingtypes.Infraction_INFRACTION_UNSPECIFIED,
	)
}

// Slash queues a slashing request for the the coordinator chain.
// All queued slashing requests will be cleared in EndBlock.
// Called by slashing keeper.
func (k Keeper) SlashWithInfractionReason(
	ctx sdk.Context,
	addr sdk.ConsAddress,
	infractionHeight, power int64,
	_ sdk.Dec,
	infraction stakingtypes.Infraction,
) math.Int {
	if infraction == stakingtypes.Infraction_INFRACTION_UNSPECIFIED {
		return math.ZeroInt()
	}

	// get VSC ID for infraction height
	vscID := k.GetValsetUpdateIDForHeight(ctx, infractionHeight)

	k.Logger(ctx).Debug(
		"vscID obtained from mapped infraction height",
		"infraction height", infractionHeight,
		"vscID", vscID,
	)

	// this is the most important step in the function
	// everything else is just here to implement StakingKeeper interface
	// IBC packets are created from slash data and sent to the coordinator during EndBlock
	k.QueueSlashPacket(
		ctx,
		abci.Validator{
			Address: addr.Bytes(),
			Power:   power,
		},
		vscID, infraction,
	)

	// Only return to comply with the interface restriction
	return math.ZeroInt()
}

// Unimplemented because jailing happens on the coordinator chain.
func (k Keeper) Jail(sdk.Context, sdk.ConsAddress) {}

// Same as above.
func (k Keeper) Unjail(sdk.Context, sdk.ConsAddress) {}

// Cannot delegate on this chain, and this should not be called by either the subscriber or the
// coordinator.
func (k Keeper) Delegation(
	sdk.Context,
	sdk.AccAddress,
	sdk.ValAddress,
) stakingtypes.DelegationI {
	panic("unimplemented on this keeper")
}

// Unused by evidence and slashing. However, I have set it up to report 0.
// Alternatively we could panic here as well.
func (k Keeper) MaxValidators(sdk.Context) uint32 {
	return 0
}

// In interchain-security, this does seem to have been implemented. However, I did not see
// the validators being persisted in the first place so I just returned an empty list.
// I also did not see this being used anywhere within the slashing module.
func (k Keeper) GetAllValidators(sdk.Context) []stakingtypes.Validator {
	return []stakingtypes.Validator{}
}

// IsJailed returns the outstanding slashing flag for the given validator adddress
func (k Keeper) IsValidatorJailed(ctx sdk.Context, addr sdk.ConsAddress) bool {
	return k.HasOutstandingDowntime(ctx, addr)
}

func (k Keeper) UnbondingTime(ctx sdk.Context) time.Duration {
	return k.GetUnbondingPeriod(ctx)
}

// implement interface method needed for x/genutil in sdk v47
// returns empty updates and err
func (k Keeper) ApplyAndReturnValidatorSetUpdates(
	sdk.Context,
) (updates []abci.ValidatorUpdate, err error) {
	return
}

// BondedRatio is the amount of tokens currently bonded, as a proportion of the total supply.
// It refers to only the staking tokens, of course, and is correctly set to zero since
// subscribers do not have staking tokens. As a consequence of this approach, subscribers must
// instead use a custom mint function if they use the x/mint module.
func (k Keeper) BondedRatio(sdk.Context) sdk.Dec {
	return sdk.ZeroDec()
}

// StakingTokenSupply returns the total amount of staking tokens in the system. See
// `BondedRatio` to understand why this is zero, and its consequences.
func (k Keeper) StakingTokenSupply(sdk.Context) math.Int {
	return sdk.ZeroInt()
}
