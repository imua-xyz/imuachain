package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"

	"github.com/imua-xyz/imuachain/x/oracle/keeper"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
)

func TestGetStakerListNoCache(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNStakers(keeper, ctx, 10)
	sl := keeper.GetStakerList(ctx, "0xe_0x1")
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(sl.StakerAddrs),
	)
}

func createNStakers(k *keeper.Keeper, ctx sdk.Context, n int) []string {
	stakers := make([]*types.Staker, n)
	ret := make([]string, 0, n)
	for i := range stakers {
		sIndex := k.IncreaseLatestStakerIndex(ctx, 1)
		stakers[i] = &types.Staker{
			StakerIndex:   sIndex,
			ValidatorList: []string{hexutil.EncodeUint64(uint64(i))},
		}
		addr := testutiltx.GenerateAddress().String()

		k.SetStaker(ctx, 1, addr, stakers[i])

		k.SetStakerIndex(ctx, 1, sIndex, addr)

		ret = append(ret, addr)
	}

	return ret
}
