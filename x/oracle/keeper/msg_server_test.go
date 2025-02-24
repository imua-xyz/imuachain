package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"

	"github.com/stretchr/testify/require"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context, keeper.Keeper) {
	k, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(2)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx), *k
}

func TestMsgServer(t *testing.T) {
	ms, ctx, _ := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}

// TODO: re-enable these tests once fixed
// func (suite *KeeperSuite) TestCreatePriceSingleBlock() {
// 	router := suite.App.MsgServiceRouter()
// 	oServer := router.Handler(&types.MsgCreatePrice{})
// 	require.EqualValues(suite.T(), 2, suite.Ctx.BlockHeight())
// 	oServer(suite.Ctx, &types.MsgCreatePrice{
// 		Creator:    suite.valAddr1.String(),
// 		Nonce:      1,
// 		FeederID:   1,
// 		Prices:     testdata.PS1,
// 		BasedBlock: 1,
// 	})
// 	oServer(suite.Ctx, &types.MsgCreatePrice{
// 		Creator:    suite.valAddr2.String(),
// 		Nonce:      1,
// 		FeederID:   1,
// 		Prices:     testdata.PS2,
// 		BasedBlock: 1,
// 	})
// 	prices, found := suite.App.OracleKeeper.GetPrices(suite.Ctx, 1)
// 	if suite.Equal(true, found, "final price should be returned") {
// 		suite.EqualValues(prices.TokenID, 1, "final price has tokenID equals to 1")
// 		suite.Equal(2, len(prices.PriceList), "length of price list should be 2 including the 0 index with an empty element as placeholder")
// 		suite.Exactly(types.Prices{
// 			TokenID:     1,
// 			NextRoundID: 2,
// 			PriceList: []*types.PriceTimeRound{
// 				{},
// 				{
// 					Price:   testdata.PTD2.Price,
// 					Decimal: 18,
// 					// since timestamp is filled with realtime, so we use the value from result to fill the expected value here
// 					Timestamp: prices.PriceList[1].Timestamp,
// 					RoundID:   1,
// 				},
// 			},
// 		}, prices)
// 	}

// 	// run the endblock to seal and prepare for next block
// 	suite.NextBlock()
// 	require.EqualValues(suite.T(), 3, suite.Ctx.BlockHeight())
// 	_, err := oServer(suite.Ctx, &types.MsgCreatePrice{
// 		Creator:    suite.valAddr1.String(),
// 		Nonce:      1,
// 		FeederID:   1,
// 		Prices:     testdata.PS1,
// 		BasedBlock: 1,
// 	})
// 	codespace, code, log := sdkerrors.ABCIInfo(err, false)
// 	suite.Equal(codespace, types.ModuleName)
// 	suite.EqualValues(code, 1)
// 	suite.Equal(log, err.Error())
// }

// func (suite *KeeperSuite) TestCreatePriceTwoBlock() {
// 	router := suite.App.MsgServiceRouter()
// 	oServer := router.Handler(&types.MsgCreatePrice{})
// 	res, _ := oServer(suite.Ctx, &types.MsgCreatePrice{
// 		Creator:    suite.valAddr1.String(),
// 		Nonce:      1,
// 		FeederID:   1,
// 		Prices:     testdata.PS1,
// 		BasedBlock: 1,
// 	})
// 	proposerAttribute, _ := res.GetEvents().GetAttributes(types.AttributeKeyProposer)
// 	proposer := proposerAttribute[0].Value
// 	suite.Equal(suite.valAddr1.String(), proposer)
// 	_, found := suite.App.OracleKeeper.GetPrices(suite.Ctx, 1)
// 	require.Equal(suite.T(), false, found)
// 	if suite.Equal(false, found) {
// 		// run the endblock to seal and prepare for next block
// 		suite.NextBlock()
// 		oServer(suite.Ctx, &types.MsgCreatePrice{
// 			Creator:    suite.valAddr2.String(),
// 			Nonce:      1,
// 			FeederID:   1,
// 			Prices:     testdata.PS3,
// 			BasedBlock: 1,
// 		})
// 		prices, found := suite.App.OracleKeeper.GetPrices(suite.Ctx, 1)
// 		if suite.Equal(true, found) {
// 			suite.Exactly(types.Prices{
// 				TokenID:     1,
// 				NextRoundID: 2,
// 				PriceList: []*types.PriceTimeRound{
// 					{},
// 					{
// 						Price:   testdata.PTD1.Price,
// 						Decimal: 18,
// 						// since timestamp is filled with realtime, so we use the value from result to fill the expected value here
// 						Timestamp: prices.PriceList[1].Timestamp,
// 						RoundID:   1,
// 					},
// 				},
// 			}, prices)
// 		}
// 	}
// }
