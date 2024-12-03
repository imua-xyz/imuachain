package keeper_test

import (
	reflect "reflect"

	math "cosmossdk.io/math"
	dogfoodkeeper "github.com/ExocoreNetwork/exocore/x/dogfood/keeper"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/cache"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/testdata"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	. "github.com/agiledragon/gomonkey/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -destination mock_validator_test.go -package keeper_test github.com/cosmos/cosmos-sdk/x/staking/types ValidatorI

var _ = Describe("MsgCreatePrice", func() {
	var c *cache.Cache
	var p *Patches
	BeforeEach(func() {
		ks.Reset()
		Expect(ks.ms).ToNot(BeNil())

		// TODO: remove monkey patch for test
		p = ApplyMethod(reflect.TypeOf(dogfoodkeeper.Keeper{}), "GetLastTotalPower", func(k dogfoodkeeper.Keeper, ctx sdk.Context) math.Int { return math.NewInt(3) })
		p.ApplyMethod(reflect.TypeOf(dogfoodkeeper.Keeper{}), "GetAllExocoreValidators", func(k dogfoodkeeper.Keeper, ctx sdk.Context) []dogfoodtypes.ExocoreValidator {
			return []dogfoodtypes.ExocoreValidator{
				{
					Address: ks.mockValAddr1,
					Power:   1,
				},
				{
					Address: ks.mockValAddr2,
					Power:   1,
				},
				{
					Address: ks.mockValAddr3,
					Power:   1,
				},
			}
		})

		Expect(ks.ctx.BlockHeight()).To(Equal(int64(12)))
	})

	AfterEach(func() {
		ks.ctrl.Finish()
		if p != nil {
			p.Reset()
		}
	})

	Context("3 validators with 1 voting power each", func() {
		BeforeEach(func() {
			ks.ms.CreatePrice(ks.ctx, &types.MsgCreatePrice{
				Creator:    ks.mockConsAddr1.String(),
				FeederID:   1,
				Prices:     testdata.PS1,
				BasedBlock: 11,
				Nonce:      1,
			})

			c = ks.App.OracleKeeper.GetCaches()
			// c = ks.ms.Keeper.GetCaches()
			var pRes cache.ItemP
			c.GetCache(&pRes)
			p4Test := types.DefaultParams()
			p4Test.TokenFeeders[1].StartBaseBlock = 1
			Expect(pRes).Should(BeEquivalentTo(p4Test))
		})

		It("success on 3rd message", func() {
			iRes := make([]*cache.ItemM, 0)
			c.GetCache(&iRes)
			Expect(iRes[0].Validator).Should(Equal(ks.mockConsAddr1.String()))

			ks.ms.CreatePrice(ks.ctx, &types.MsgCreatePrice{
				Creator:    ks.mockConsAddr2.String(),
				FeederID:   1,
				Prices:     testdata.PS2,
				BasedBlock: 11,
				Nonce:      1,
			},
			)
			ks.ms.CreatePrice(ks.ctx, &types.MsgCreatePrice{})
			c.GetCache(&iRes)
			Expect(len(iRes)).Should(Equal(2))

			ks.ms.CreatePrice(ks.ctx, &types.MsgCreatePrice{
				Creator:    ks.mockConsAddr3.String(),
				FeederID:   1,
				Prices:     testdata.PS4,
				BasedBlock: 11,
				Nonce:      1,
			},
			)
			c.GetCache(&iRes)
			Expect(len(iRes)).Should(Equal(0))
			prices := ks.k.GetAllPrices(sdk.UnwrapSDKContext(ks.ctx))
			Expect(prices[0]).Should(BeEquivalentTo(types.Prices{
				TokenID:     1,
				NextRoundID: 3,
				PriceList: []*types.PriceTimeRound{
					{
						Price:     "1",
						Decimal:   0,
						Timestamp: "",
						RoundID:   1,
					},
					{
						Price:     testdata.PTD2.Price,
						Decimal:   testdata.PTD2.Decimal,
						Timestamp: prices[0].PriceList[1].Timestamp,
						RoundID:   2,
					},
				},
			}))
		})
	})
})
