package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	erc20types "github.com/evmos/evmos/v16/x/erc20/types"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/utils"
)

// interface guards
var (
	_ erc20types.StakingKeeper = Keeper{}
	_ evmtypes.StakingKeeper   = Keeper{}
)

// GetValidatorByConsAddr is an implementation of the StakingKeeper interface
// expected by the EVM module. It returns a validator given a consensus address.
// The EVM module uses it to determine the proposer's AccAddress for the block.
// ConsAddress -> lookup AccAddress -> convert to ValAddress -> bech32ify.
// ValAddress string is then decoded back by the EVM module to get bytes, which
// make the 0x address of the coinbase.
func (k Keeper) GetValidatorByConsAddr(
	ctx sdk.Context, consAddr sdk.ConsAddress,
) (validator stakingtypes.Validator, found bool) {
	val := k.ValidatorByConsAddr(ctx, consAddr)
	if val == nil {
		return stakingtypes.Validator{}, false
	}
	return val.(stakingtypes.Validator), true
}

// BondDenom is an implementation of the StakingKeeper interface expected by the
// ERC20 module. It returns the bond denom for the module. The ERC20 module uses
// this function to determine whether a token sent (or received) over IBC is the
// staking (==native) token. If it is, then the module lets the token through.
// That is the behavior we wish to retain with our chain as well.
func (k Keeper) BondDenom(sdk.Context) string {
	return utils.BaseDenom
}
