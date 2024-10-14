package keeper

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetTokens returns a list of token-index mapping registered in params
func (k Keeper) GetTokens(ctx sdk.Context) []*types.TokenIndex {
	params := k.GetParams(ctx)
	ret := make([]*types.TokenIndex, 0, len(params.Tokens))
	for idx, token := range params.Tokens {
		ret = append(ret, &types.TokenIndex{
			Token: token.Name,
			Index: uint64(idx),
		})
	}
	return ret
}
