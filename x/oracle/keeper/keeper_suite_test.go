package keeper_test

import (
	"context"
	"testing"

	math "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/testutil"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"

	"github.com/cosmos/cosmos-sdk/testutil/mock"
)

type KeeperSuite struct {
	testutil.BaseTestSuite

	t        *testing.T
	k        keeper.Keeper
	ctx      sdk.Context
	ms       types.MsgServer
	ctrl     *gomock.Controller
	valAddr1 sdk.ValAddress
	valAddr2 sdk.ValAddress

	mockValAddr1 []byte
	mockValAddr2 []byte
	mockValAddr3 []byte

	mockConsAddr1 sdk.AccAddress
	mockConsAddr2 sdk.AccAddress
	mockConsAddr3 sdk.AccAddress
}

var ks *KeeperSuite

func TestKeeper(t *testing.T) {
	var ctxW context.Context
	ks = &KeeperSuite{}
	ks.ms, ctxW, ks.k = setupMsgServer(t)
	ks.ctx = sdk.UnwrapSDKContext(ctxW)
	ks.t = t

	// setup validatorset info
	privVal1 := mock.NewPV()
	pubKey1, _ := privVal1.GetPubKey()
	ks.mockValAddr1 = pubKey1.Address().Bytes()
	//		operator1 = sdk.ValAddress(pubKey1.Address())
	ks.mockConsAddr1 = sdk.AccAddress(pubKey1.Address())

	privVal2 := mock.NewPV()
	pubKey2, _ := privVal2.GetPubKey()
	ks.mockValAddr2 = pubKey2.Address().Bytes()
	ks.mockConsAddr2 = sdk.AccAddress(pubKey2.Address())

	privVal3 := mock.NewPV()
	pubKey3, _ := privVal3.GetPubKey()
	ks.mockValAddr3 = pubKey3.Address().Bytes()
	ks.mockConsAddr3 = sdk.AccAddress(pubKey3.Address())

	suite.Run(t, ks)

	resetSingle(ks.App.OracleKeeper)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperSuite) Reset() {
	p4Test := DefaultParamsForTest()
	p4Test.TokenFeeders[1].StartBaseBlock = 1
	suite.k.SetParams(suite.ctx, p4Test)
	suite.ctx = suite.ctx.WithBlockHeight(12)

	suite.ctrl = gomock.NewController(suite.t)
	resetSingle(suite.App.OracleKeeper)
}

func (suite *KeeperSuite) SetupTest() {
	suite.DoSetupTest()

	depositAmountNST := math.NewInt(64)
	suite.App.AssetsKeeper.SetStakingAssetInfo(suite.Ctx, &assetstypes.StakingAssetInfo{
		AssetBasicInfo: assetstypes.AssetInfo{
			Name:             "Native Restaking ETH",
			Symbol:           "NSTETH",
			Address:          "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			Decimals:         18,
			LayerZeroChainID: suite.ClientChains[0].LayerZeroChainID,
			MetaInfo:         "native restaking token",
		},
		StakingTotalAmount: depositAmountNST,
	})

	validators := suite.ValSet.Validators
	suite.valAddr1, _ = sdk.ValAddressFromBech32(sdk.ValAddress(validators[0].Address).String())
	suite.valAddr2, _ = sdk.ValAddressFromBech32(sdk.ValAddress(validators[1].Address).String())
	resetSingle(suite.App.OracleKeeper)

	suite.k = suite.App.OracleKeeper
	suite.ms = keeper.NewMsgServerImpl(suite.App.OracleKeeper)
	suite.ctx = suite.Ctx
	// Initialize params
	p4Test := DefaultParamsForTest()
	p4Test.TokenFeeders[1].StartBaseBlock = 1
	suite.k.SetParams(suite.ctx, p4Test)
	suite.ctx = suite.ctx.WithBlockHeight(12)
}

func resetSingle(k keeper.Keeper) {
	k.ResetAggregatorContext()
	k.ResetCache()
}
