package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/testutil"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/suite"

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

	resetSingle()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperSuite) Reset() {
	var ctxW context.Context
	suite.ms, ctxW, suite.k = setupMsgServer(suite.t)
	suite.ctx = sdk.UnwrapSDKContext(ctxW)
	suite.ctrl = gomock.NewController(suite.t)
	resetSingle()
}

func (suite *KeeperSuite) SetupTest() {
	suite.DoSetupTest()

	depositAmountNST := math.NewInt(132)
	suite.App.AssetsKeeper.SetStakingAssetInfo(suite.Ctx, &assetstypes.StakingAssetInfo{
		AssetBasicInfo: assetstypes.AssetInfo{
			Name:             "Native Restaking ETH",
			Symbol:           "NSTETH",
			Address:          "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			Decimals:         6,
			LayerZeroChainID: suite.ClientChains[0].LayerZeroChainID,
			MetaInfo:         "native restaking token",
		},
		StakingTotalAmount: depositAmountNST,
	})

	stakerID, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[0].LayerZeroChainID,
		ks.StakerAddr, "",
	)
	NSTAssetAddr := assetstypes.GenerateNSTAddr(
		suite.ClientChains[0].AddressLength,
	)
	_, assetIDNST := assetstypes.GetStakerIDAndAssetID(suite.ClientChains[0].LayerZeroChainID, []byte{}, NSTAssetAddr)
	suite.App.AssetsKeeper.UpdateStakerAssetState(ks.Ctx, stakerID, assetIDNST, assetstypes.DeltaStakerSingleAsset{
		TotalDepositAmount:        depositAmountNST,
		WithdrawableAmount:        depositAmountNST,
		PendingUndelegationAmount: sdk.ZeroInt(),
	})

	validators := suite.ValSet.Validators
	suite.valAddr1, _ = sdk.ValAddressFromBech32(sdk.ValAddress(validators[0].Address).String())
	suite.valAddr2, _ = sdk.ValAddressFromBech32(sdk.ValAddress(validators[1].Address).String())
	resetSingle()
}

func resetSingle() {
	keeper.ResetAggregatorContext()
	keeper.ResetCache()
}
