package keeper_test

import (
	reflect "reflect"

	dogfoodkeeper "github.com/ExocoreNetwork/exocore/x/dogfood/keeper"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	. "github.com/agiledragon/gomonkey/v2"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MsgUpdateParams", Ordered, func() {
	var defaultParams types.Params
	var patcher *Patches
	chainIDtest := "exocoretestnet_233-1"
	AfterAll(func() {
		patcher.Reset()
		ks.Reset()
	})
	BeforeEach(func() {
		ks.Reset()
		Expect(ks.ms).ToNot(BeNil())
		defaultParams = ks.k.GetParams(ks.ctx)

		privVal1 := mock.NewPV()
		pubKey1, _ := privVal1.GetPubKey()

		privVal2 := mock.NewPV()
		pubKey2, _ := privVal2.GetPubKey()

		privVal3 := mock.NewPV()
		pubKey3, _ := privVal3.GetPubKey()

		patcher = ApplyMethod(reflect.TypeOf(dogfoodkeeper.Keeper{}), "GetAllExocoreValidators", func(k dogfoodkeeper.Keeper, ctx sdk.Context) []dogfoodtypes.ExocoreValidator {
			return []dogfoodtypes.ExocoreValidator{
				{
					Address: pubKey1.Address(),
					Power:   1,
				},
				{
					Address: pubKey2.Address(),
					Power:   1,
				},
				{
					Address: pubKey3.Address(),
					Power:   1,
				},
			}
		})
	})

	Context("Update Chains", func() {
		inputAddChains := []string{
			`{"chains":[{"name":"Bitcoin", "desc":"-"}]}`,
			`{"chains":[{"name":"Ethereum", "desc":"-"}]}`,
		}
		It("add chain with new name", func() {
			msg := types.NewMsgUpdateParams("", inputAddChains[0])
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), msg)
			Expect(err).Should(BeNil())
			p := ks.k.GetParams(ks.ctx)
			Expect(p.Chains[2].Name).Should(BeEquivalentTo("Bitcoin"))
		})
		It("add chain with duplicated name", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), types.NewMsgUpdateParams("", inputAddChains[1]))
			Expect(err).Should(MatchError(types.ErrInvalidParams.Wrap("invalid source to add, duplicated")))
		})
	})
	Context("Update Sources", func() {
		inputAddSources := []string{
			`{"sources":[{"name":"CoinGecko", "desc":"-", "valid":true}]}`,
			`{"sources":[{"name":"CoinGecko", "desc":"-"}]}`,
			`{"sources":[{"name":"Chainlink", "desc":"-", "valid":true}]}`,
		}
		It("add valid source with new name", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), types.NewMsgUpdateParams("", inputAddSources[0]))
			Expect(err).Should(BeNil())
			p := ks.k.GetParams(ks.ctx)
			Expect(p.Sources[2].Name).Should(BeEquivalentTo("CoinGecko"))
		})
		It("add invalid source with new name", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), types.NewMsgUpdateParams("", inputAddSources[1]))
			Expect(err).Should(MatchError(types.ErrInvalidParams.Wrap("invalid source to add, new source should be valid")))
		})
		It("add source with duplicated name", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), types.NewMsgUpdateParams("", inputAddSources[2]))
			Expect(err).Should(MatchError(types.ErrInvalidParams.Wrap("invalid source to add, duplicated")))
		})
	})
	Context("Update Tokens", func() {
		startBasedBlocks := []uint64{1, 3, 3, 3, 1, 1, 1}
		inputUpdateTokens := []string{
			`{"tokens":[{"name":"UNI", "chain_id":"1"}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"1", "decimal":8}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"1", "asset_id":"assetID"}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"1", "contract_address":"contractAddress"}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"1", "decimal":8}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"0"}]}`,
			`{"tokens":[{"name":"ETH", "chain_id":"3"}]}`,
		}
		errs := []error{
			nil,
			nil,
			nil,
			nil,
			nil,
			types.ErrInvalidParams.Wrap("invalid token to add, chain not found"),
			types.ErrInvalidParams.Wrap("invalid token to add, chain not found"),
		}
		token := types.DefaultParams().Tokens[1]
		token1 := *token
		token1.Decimal = 8

		token2 := *token
		token2.AssetID = "assetID"

		token3 := *token
		token3.ContractAddress = "0x123"

		updatedTokenETH := []*types.Token{
			nil,
			&token1,
			&token2,
			&token3,
			token,
			nil,
			nil,
		}

		for i, input := range inputUpdateTokens {
			It("", func() {
				if startBasedBlocks[i] > 1 {
					p := defaultParams
					p.TokenFeeders[1].StartBaseBlock = startBasedBlocks[i]
					ks.k.SetParams(ks.ctx, p)
				}
				_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), types.NewMsgUpdateParams("", input))
				if errs[i] == nil {
					Expect(err).Should(BeNil())
				} else {
					Expect(err).Should(MatchError(errs[i]))
				}
				if updatedTokenETH[i] != nil {
					p := ks.k.GetParams(ks.ctx)
					Expect(p.Tokens[1]).Should(BeEquivalentTo(updatedTokenETH[i]))
				}
			})
		}
	})

	Context("update maxSizePrices", func() {
		It("update maxSizePrices", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), &types.MsgUpdateParams{
				Params: types.Params{
					MaxSizePrices: 100,
				},
			})
			Expect(err).Should(BeNil())
			p := ks.k.GetParams(ks.ctx)
			Expect(p.MaxSizePrices).Should(BeEquivalentTo(100))
		})
	})

	Context("update TokenFeeders", func() {
		It("update StartBaseBlock for TokenFeeder", func() {
			ks.ctx = ks.ctx.WithBlockHeight(2)
			p := defaultParams
			p.TokenFeeders[1].StartBaseBlock = 10
			ks.k.SetParams(ks.ctx, p)
			p.TokenFeeders[1].StartBaseBlock = 5
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), &types.MsgUpdateParams{
				Params: types.Params{
					TokenFeeders: []*types.TokenFeeder{
						{
							TokenID:        1,
							StartBaseBlock: 5,
						},
					},
				},
			})
			Expect(err).Should(BeNil())
			p = ks.k.GetParams(ks.ctx)
			Expect(p.TokenFeeders[1].StartBaseBlock).Should(BeEquivalentTo(5))
		})
		It("Add AssetID for Token", func() {
			_, err := ks.ms.UpdateParams(ks.ctx.WithChainID(chainIDtest), &types.MsgUpdateParams{
				Params: types.Params{
					Tokens: []*types.Token{
						{
							Name:    "ETH",
							ChainID: 1,
							AssetID: "0x83e6850591425e3c1e263c054f4466838b9bd9e4_0x9ce1",
						},
					},
				},
			})
			Expect(err).Should(BeNil())
			p := ks.k.GetParams(ks.ctx)
			Expect(p.Tokens[1].AssetID).Should(BeEquivalentTo("0x83e6850591425e3c1e263c054f4466838b9bd9e4_0x9ce1"))
		})
	})
})
