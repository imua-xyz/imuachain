package types_test

import (
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	commission := stakingtypes.NewCommission(params.MinCommissionRate, sdk.OneDec(), sdk.OneDec())

	testCases := []struct {
		name           string
		genState       *types.GenesisState
		expPass        bool
		malleate       func(*types.GenesisState)
		expErrContains string
	}{
		{
			name:           "valid genesis constructor",
			genState:       newGen,
			expPass:        true,
			expErrContains: "",
		},
		{
			name:           "default",
			genState:       types.DefaultGenesis(),
			expPass:        true,
			expErrContains: "",
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
			expPass:        false,
			expErrContains: "invalid bech32 address",
		},
		{
			name: "invalid genesis state due to duplicate operator address",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							ClientChainEarningsAddr: &types.ClientChainEarningAddrList{
								EarningInfoList: []*types.ClientChainEarningAddrInfo{
									{
										LzClientChainID:        1,
										ClientChainEarningAddr: utiltx.GenerateAddress().String(),
									},
								},
							},
							Commission: commission,
						},
					},
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator2",
							ClientChainEarningsAddr: &types.ClientChainEarningAddrList{
								EarningInfoList: []*types.ClientChainEarningAddrInfo{
									{
										LzClientChainID:        1,
										ClientChainEarningAddr: utiltx.GenerateAddress().String(),
									},
								},
							},
							Commission: commission,
						},
					},
				},
				Params: params,
			},
			expPass:        false,
			expErrContains: "duplicate operator address",
		},
		{
			name: "invalid genesis state due to duplicate lz chain id",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
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
							Commission: commission,
						},
					},
				},
				Params: params,
			},
			expPass:        false,
			expErrContains: "duplicate lz client chain id",
		},
		{
			name: "invalid genesis state due to invalid cons key operator address",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: "invalid",
					},
				},
				Params: params,
			},
			expPass:        false,
			expErrContains: "ValidateOperatorConsKeyRecords: invalid operator address",
		},
		{
			name: "invalid genesis state due to unregistered operator in cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
				},
				OperatorRecords: []types.OperatorConsKeyRecord{
					{
						OperatorAddress: accAddress2.String(),
					},
				},
				Params: params,
			},
			expPass:        false,
			expErrContains: "un-registered operator",
		},
		{
			name: "invalid genesis state due to duplicate operator in cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
					{
						OperatorAddress: accAddress2.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress2.String(),
							ApproveAddr:      accAddress2.String(),
							OperatorMetaInfo: "operator2",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "duplicate operator record for operator",
		},
		{
			name: "invalid genesis state due to invalid cons key",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "ValidateOperatorConsKeyRecords: invalid consensus key",
		},
		{
			name: "invalid genesis state due to duplicate cons key for the same chain id",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
					{
						OperatorAddress: accAddress2.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress2.String(),
							ApproveAddr:      accAddress2.String(),
							OperatorMetaInfo: "operator2",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "ValidateOperatorConsKeyRecords: duplicate consensus key",
		},
		{
			name: "invalid genesis due to negative duration",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "duration must be non-negative",
		},
		{
			name: "invalid genesis due to negative rate",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "dec must be non-negative",
		},
		{
			name: "invalid genesis due to nil rate",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
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
			expPass:        false,
			expErrContains: "dec must be non-nil",
		},
		{
			name: "invalid genesis due to nil operator name",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "",
							Commission:       commission,
						},
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
					types.DefaultMinCommissionRate,
				),
			},
			expPass:        false,
			expErrContains: "operator meta info is empty",
		},
		{
			name: "invalid genesis due to large operator name",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: strings.Repeat("a", stakingtypes.MaxMonikerLength+1),
							Commission:       commission,
						},
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
					types.DefaultMinCommissionRate,
				),
			},
			expPass:        false,
			expErrContains: "info length exceeds",
		},
		{
			name: "invalid genesis due to nil commission",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       stakingtypes.NewCommission(sdk.Dec{}, sdk.Dec{}, sdk.Dec{}),
						},
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
					types.DefaultMinCommissionRate,
				),
			},
			expPass:        false,
			expErrContains: "commission rate is nil",
		},
		{
			name: "invalid genesis due to invalid commission",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       stakingtypes.NewCommission(sdk.OneDec(), sdk.ZeroDec(), sdk.OneDec()),
						},
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
					types.DefaultMinCommissionRate,
				),
			},
			expPass:        false,
			expErrContains: "invalid commission rate",
		},
		{
			name: "invalid genesis due to duplicate operator name",
			genState: &types.GenesisState{
				Operators: []types.OperatorDetail{
					{
						OperatorAddress: accAddress1.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress1.String(),
							ApproveAddr:      accAddress1.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
					{
						OperatorAddress: accAddress2.String(),
						OperatorInfo: types.OperatorInfo{
							EarningsAddr:     accAddress2.String(),
							ApproveAddr:      accAddress2.String(),
							OperatorMetaInfo: "operator1",
							Commission:       commission,
						},
					},
				},
				Params: params,
			},
			expPass:        false,
			expErrContains: "duplicate operator name",
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
			suite.Require().True(len(tc.expErrContains) > 0, tc.name)
			suite.Require().Error(err, tc.name)
			suite.Require().Contains(err.Error(), tc.expErrContains, tc.name)
		}
	}
}
