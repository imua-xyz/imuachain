package types_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/assets/types"
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
	params := types.DefaultParams()
	params.Gateways = []string{"0x0000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000002"}
	newGen := types.NewGenesis(
		params, nil,
		nil, nil, nil,
	)
	// genesis data that is hardcoded for use in the tests
	ethClientChain := types.ClientChainInfo{
		Name:               "ethereum",
		MetaInfo:           "ethereum blockchain",
		ChainId:            1,
		FinalizationBlocks: 10,
		LayerZeroChainID:   101,
		AddressLength:      20,
	}
	// do not hardcode the address to avoid gitleaks complaining.
	tokenAddress := strings.ToLower(utiltx.GenerateAddress().String())
	usdtClientChainAsset := types.AssetInfo{
		Name:             "Tether USD",
		Symbol:           "USDT",
		Address:          tokenAddress,
		Decimals:         6,
		LayerZeroChainID: ethClientChain.LayerZeroChainID,
		MetaInfo:         "Tether USD token",
	}
	stakingInfo := types.StakingAssetInfo{
		AssetBasicInfo:     usdtClientChainAsset,
		StakingTotalAmount: math.NewInt(100),
	}
	// generated information
	ethAddress := utiltx.GenerateAddress()
	// csmAddress := sdk.AccAddress(ethAddress.Bytes())
	stakerID, assetID := types.GetStakerIDAndAssetIDFromStr(
		usdtClientChainAsset.LayerZeroChainID, ethAddress.String(), usdtClientChainAsset.Address,
	)
	genesisDeposit := types.DepositsByStaker{
		StakerID: stakerID,
		Deposits: []types.DepositByAsset{
			{
				AssetID: assetID,
				Info: types.StakerAssetInfo{
					TotalDepositAmount:        math.NewInt(100),
					WithdrawableAmount:        math.NewInt(100),
					PendingUndelegationAmount: math.NewInt(0),
				},
			},
		},
	}

	testCases := []struct {
		name       string
		genState   *types.GenesisState
		expPass    bool
		malleate   func(*types.GenesisState)
		unmalleate func(*types.GenesisState)
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
			name: "valid genesis created here",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
			},
			expPass: true,
		},
		{
			name: "invalid genesis due to duplicate client chain",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain, ethClientChain,
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to zero address length",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
			},
			malleate: func(gs *types.GenesisState) {
				gs.ClientChains[0].AddressLength = 0
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.ClientChains[0].AddressLength = 20
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to missing client chain",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to wrong token address format",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
			},
			malleate: func(gs *types.GenesisState) {
				gs.Tokens[0].AssetBasicInfo.Address = "fakeTokenAddress"
			},
			unmalleate: func(gs *types.GenesisState) {
				// gs.Tokens[0].AssetBasicInfo is a pointer, so we undo the change manually
				gs.Tokens[0].AssetBasicInfo.Address = tokenAddress
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to duplicate token",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo, stakingInfo,
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to upper case staker id",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = strings.ToUpper(gs.Deposits[0].StakerID)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = strings.ToLower(gs.Deposits[0].StakerID)
			},
		},
		{
			name: "invalid genesis due to invalid staker id",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = "fakeStaker"
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = stakerID
			},
		},
		{
			name: "invalid genesis due to staker id from unknown chain",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				stakerID, _ := types.GetStakerIDAndAssetIDFromStr(
					usdtClientChainAsset.LayerZeroChainID+1,
					ethAddress.String(), usdtClientChainAsset.Address,
				)
				gs.Deposits[0].StakerID = stakerID
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = stakerID
			},
		},
		{
			name: "invalid genesis due to non hex staker id",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = "fakeNonHexStaker_0x65"
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].StakerID = stakerID
			},
		},
		{
			name: "invalid genesis due to duplicate staker id",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit, genesisDeposit},
			},
			expPass: false,
		},
		{
			name: "invalid genesis due to unknown asset id for staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits[0].AssetID = "fakeAssetID"
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits[0].AssetID = assetID
			},
		},
		{
			name: "invalid genesis due to different chain ids for asset and staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain, ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo, stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit, genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				// new chain with different layer zero chain id
				gs.ClientChains[1].LayerZeroChainID += 1
				// new asset (old asset is a pointer so can't alter that)
				tokenAddress := utiltx.GenerateAddress().String()
				usdcClientChainAsset := types.AssetInfo{
					Name:             "Circle USD",
					Symbol:           "USDC",
					Address:          tokenAddress,
					Decimals:         18,
					LayerZeroChainID: ethClientChain.LayerZeroChainID + 1,
					MetaInfo:         "Circle USD token",
				}
				stakingInfo := types.StakingAssetInfo{
					AssetBasicInfo:     usdcClientChainAsset,
					StakingTotalAmount: math.NewInt(0),
				}
				gs.Tokens[1] = stakingInfo
				stakerID, _ := types.GetStakerIDAndAssetIDFromStr(
					usdtClientChainAsset.LayerZeroChainID+1,
					ethAddress.String(), usdtClientChainAsset.Address,
				)
				// change stakerID to be that of the second chain
				gs.Deposits[1].StakerID = stakerID
				// but keep the assetID the same
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.ClientChains[1].LayerZeroChainID -= 1
				gs.Tokens[1].AssetBasicInfo.LayerZeroChainID -= 1
				gs.Deposits[1].StakerID = stakerID
			},
		},
		{
			name: "invalid genesis due to duplicate asset id for staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits = append(
					gs.Deposits[0].Deposits,
					genesisDeposit.Deposits[0],
				)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits = gs.Deposits[0].Deposits[:1]
			},
		},
		{
			name: "invalid genesis due to nil values for staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits[0].Info = types.StakerAssetInfo{}
			},
			unmalleate: func(gs *types.GenesisState) {
				genesisDeposit.Deposits[0].Info = types.StakerAssetInfo{
					TotalDepositAmount:        math.NewInt(100),
					WithdrawableAmount:        math.NewInt(0),
					PendingUndelegationAmount: math.NewInt(0),
				}
				gs.Deposits[0].Deposits[0].Info = genesisDeposit.Deposits[0].Info
			},
		},
		{
			name: "invalid genesis due to negative amount for staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits[0].Info.TotalDepositAmount = math.NewInt(-1)
			},
			unmalleate: func(gs *types.GenesisState) {
				genesisDeposit.Deposits[0].Info = types.StakerAssetInfo{
					TotalDepositAmount:        math.NewInt(100),
					WithdrawableAmount:        math.NewInt(0),
					PendingUndelegationAmount: math.NewInt(0),
				}
				gs.Deposits[0].Deposits[0].Info = genesisDeposit.Deposits[0].Info
			},
		},
		{
			name: "invalid genesis due to excess withdrawable amount for staker",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: false,
			malleate: func(gs *types.GenesisState) {
				gs.Deposits[0].Deposits[0].Info.TotalDepositAmount = math.NewInt(1)
				gs.Deposits[0].Deposits[0].Info.WithdrawableAmount = math.NewInt(2)
			},
			unmalleate: func(gs *types.GenesisState) {
				genesisDeposit.Deposits[0].Info = types.StakerAssetInfo{
					TotalDepositAmount:        math.NewInt(100),
					WithdrawableAmount:        math.NewInt(0),
					PendingUndelegationAmount: math.NewInt(0),
				}
				gs.Deposits[0].Deposits[0].Info = genesisDeposit.Deposits[0].Info
			},
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClientChains: []types.ClientChainInfo{
					ethClientChain,
				},
				Tokens: []types.StakingAssetInfo{
					stakingInfo,
				},
				Deposits: []types.DepositsByStaker{genesisDeposit},
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.malleate != nil {
			tc.malleate(tc.genState)
			// check that unmalleate is not nil
			suite.Require().NotNil(tc.unmalleate, tc.name)
		}
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
		if tc.unmalleate != nil {
			tc.unmalleate(tc.genState)
		}
	}
}
