package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// GetTokens returns a list of token-index mapping registered in params
func (k Keeper) GetTokens(ctx sdk.Context) []*types.TokenIndex {
	params := k.GetParams(ctx)
	ret := make([]*types.TokenIndex, 0, len(params.Tokens))
	for idx, token := range params.Tokens {
		ret = append(ret, &types.TokenIndex{
			Token: token.Name,
			// #nosec G115
			Index: uint64(idx),
		})
	}
	return ret
}
