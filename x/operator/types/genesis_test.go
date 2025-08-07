package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/operator/types"
	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
}

func (suite *GenesisTestSuite) SetupTest() {
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	key := hexutil.Encode(ed25519.GenPrivKey().PubKey().Bytes())
	accAddress1 := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	accAddress2 := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	params := types.DefaultParams()
	newGen := &types.GenesisState{Params: params}

	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
		malleate func(*types.GenesisState)
	}{
		{
			name:     "valid genesis constructor",
			genState: newGen,
			expPass:  true,
		},
		{
			name:     "default",
			genState: types.DefaultGenesis(),
			expPass:  true,
		},
		{
			name: "invalid genesis state due to non bech32 operator address",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: "invalid",
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to duplicate operator address",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to duplicate lz chain id",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr: accAddress1.String(),
							ClientChainEarningsAddr: &types.ClientChainEarningAddrList{
								EarningInfoList: []*types.ClientChainEarningAddrInfo{
									{
										LzClientChainID:        1,
										ClientChainEarningAddr: utiltx.GenerateAddress().String(),
									},
									{
										LzClientChainID:        1,
										ClientChainEarningAddr: utiltx.GenerateAddress().String(),
									},
								},
							},
						},
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to invalid cons key operator address",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: "invalid",
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to unregistered operator in cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress2.String(),
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to duplicate operator in cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
					{
						OperatorAddress: accAddress2.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: hexutil.Encode(ed25519.GenPrivKey().PubKey().Bytes()),
							},
						},
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to invalid cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key + "fake",
							},
						},
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis state due to duplicate cons key for the same chain id",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
					{
						OperatorAddress: accAddress2.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
					{
						OperatorAddress: accAddress2.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
				},
				Params: params,
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to negative duration",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
				},
				Params: types.NewParams(
					types.DefaultMinCommissionUpdateInterval*-1,
					types.DefaultMinCommissionRate,
				),
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to negative rate",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
				},
				Params: types.NewParams(
					types.DefaultMinCommissionUpdateInterval,
					types.DefaultMinCommissionRate.Neg(),
				),
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to nil rate",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress1.String(),
						Chains: []types.ChainDetails{
							{
								ChainID:      utils.TestnetChainID,
								ConsensusKey: key,
							},
						},
					},
				},
				Params: types.NewParams(
					types.DefaultMinCommissionUpdateInterval,
					sdk.Dec{},
				),
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.malleate != nil {
			tc.malleate(tc.genState)
		}
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}
