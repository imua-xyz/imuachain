package evm_test

// This file contains tests for checking that:
// 1. All the contracts from DefaultPredeploys are included in genesis with nonce 1.
// 2. If a predeployed address has existing balance, it is retained.
// 3. CREATE2 can be used to deploy CREATE3 successfully.

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ExocoreNetwork/exocore/testutil"
	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	"github.com/ExocoreNetwork/exocore/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmostypes "github.com/evmos/evmos/v16/types"
	"github.com/stretchr/testify/suite"
)

var s *KeeperTestSuite

type KeeperTestSuite struct {
	testutil.BaseTestSuite
}

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest()
}

func (suite *KeeperTestSuite) TestPredeploysExist() {
	// reset test to ensure that balance retention running first does not impact us
	suite.Balances = nil
	suite.SetupTest()
	expectedNonce := suite.App.EvmKeeper.GetNewContractNonce(suite.Ctx)
	expectedBalance := big.NewInt(0)
	evmParams := suite.App.EvmKeeper.GetParams(suite.Ctx)
	evmDenom := evmParams.GetEvmDenom()
	for _, predeploy := range types.DefaultPredeploys {
		bytesAddr := predeploy.GetByteAddress()
		acc := suite.App.AccountKeeper.GetAccount(suite.Ctx, bytesAddr[:])
		suite.Require().NotNil(acc)

		ethAcc, ok := acc.(evmostypes.EthAccountI)
		suite.Require().True(ok)
		// check code hash
		suite.Require().Equal(predeploy.GetCodeHash(), ethAcc.GetCodeHash())
		// check nonce
		suite.Require().Equal(expectedNonce, ethAcc.GetSequence())
		// check balance via evm keeper
		suite.Require().Equal(
			expectedBalance, suite.App.EvmKeeper.GetBalance(suite.Ctx, bytesAddr),
			predeploy.Address,
		)
		// check balance via bank keeper
		suite.Require().Equal(
			expectedBalance,
			suite.App.BankKeeper.GetBalance(
				suite.Ctx, bytesAddr[:], evmDenom,
			).Amount.BigInt(),
		)
		// check that code exists
		suite.Require().NotNil(suite.App.EvmKeeper.GetCode(suite.Ctx, predeploy.GetCodeHash()))
	}
}

func (suite *KeeperTestSuite) TestBalanceRetention() {
	evmParams := suite.App.EvmKeeper.GetParams(suite.Ctx)
	evmDenom := evmParams.GetEvmDenom()
	targetBalance := sdkmath.NewInt(100)
	// set balance > 0 for all of the predeployed address
	for _, predeploy := range types.DefaultPredeploys {
		addr := predeploy.GetByteAddress()
		suite.Balances = append(
			suite.Balances,
			banktypes.Balance{
				Address: sdk.AccAddress(addr.Bytes()).String(),
				Coins:   sdk.NewCoins(sdk.NewCoin(evmDenom, targetBalance)),
			},
		)
	}
	// now redo the genesis
	suite.SetupTest()
	// check the state of the predeploys
	expectedNonce := suite.App.EvmKeeper.GetNewContractNonce(suite.Ctx)
	for _, predeploy := range types.DefaultPredeploys {
		addr := predeploy.GetByteAddress()
		// check balance via evm keeper
		suite.Require().Equal(
			targetBalance.BigInt(), suite.App.EvmKeeper.GetBalance(suite.Ctx, addr),
		)
		// check balance via bank keeper
		suite.Require().Equal(
			targetBalance,
			suite.App.BankKeeper.GetBalance(
				suite.Ctx, addr[:], evmDenom,
			).Amount,
		)
		// check nonce
		acc := suite.App.AccountKeeper.GetAccount(suite.Ctx, addr[:])
		suite.Require().NotNil(acc)
		ethAcc, ok := acc.(evmostypes.EthAccountI)
		suite.Require().True(ok)
		suite.Require().Equal(expectedNonce, ethAcc.GetSequence())
		// check code hash
		suite.Require().Equal(predeploy.GetCodeHash(), ethAcc.GetCodeHash())
		// check that code exists
		suite.Require().NotNil(suite.App.EvmKeeper.GetCode(suite.Ctx, predeploy.GetCodeHash()))
	}
}

func (suite *KeeperTestSuite) TestCreate3() {
	// contract to call
	create2 := common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")
	// blank salt
	salt := common.Hash{}
	// runCode is the code fetched via eth_getCode.
	// it can be used directly in a predeploy but not to create a new contract, because
	// it does not have the constructor.
	runCode := "6080604052600436106100295760003560e01c806350f1c4641461002e578063cdcb760a14610077575b600080fd5b34801561003a57600080fd5b5061004e610049366004610489565b61008a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b61004e6100853660046104fd565b6100ee565b6040517fffffffffffffffffffffffffffffffffffffffff000000000000000000000000606084901b166020820152603481018290526000906054016040516020818303038152906040528051906020012091506100e78261014c565b9392505050565b6040517fffffffffffffffffffffffffffffffffffffffff0000000000000000000000003360601b166020820152603481018390526000906054016040516020818303038152906040528051906020012092506100e78383346102b2565b604080518082018252601081527f67363d3d37363d34f03d5260086018f30000000000000000000000000000000060209182015290517fff00000000000000000000000000000000000000000000000000000000000000918101919091527fffffffffffffffffffffffffffffffffffffffff0000000000000000000000003060601b166021820152603581018290527f21c35dbe1b344a2488cf3321d6ce542f8e9f305544ff09e4993a62319a497c1f60558201526000908190610228906075015b6040516020818303038152906040528051906020012090565b6040517fd69400000000000000000000000000000000000000000000000000000000000060208201527fffffffffffffffffffffffffffffffffffffffff000000000000000000000000606083901b1660228201527f010000000000000000000000000000000000000000000000000000000000000060368201529091506100e79060370161020f565b6000806040518060400160405280601081526020017f67363d3d37363d34f03d5260086018f30000000000000000000000000000000081525090506000858251602084016000f5905073ffffffffffffffffffffffffffffffffffffffff811661037d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601160248201527f4445504c4f594d454e545f4641494c454400000000000000000000000000000060448201526064015b60405180910390fd5b6103868661014c565b925060008173ffffffffffffffffffffffffffffffffffffffff1685876040516103b091906105d6565b60006040518083038185875af1925050503d80600081146103ed576040519150601f19603f3d011682016040523d82523d6000602084013e6103f2565b606091505b50509050808015610419575073ffffffffffffffffffffffffffffffffffffffff84163b15155b61047f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601560248201527f494e495449414c495a4154494f4e5f4641494c454400000000000000000000006044820152606401610374565b5050509392505050565b6000806040838503121561049c57600080fd5b823573ffffffffffffffffffffffffffffffffffffffff811681146104c057600080fd5b946020939093013593505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000806040838503121561051057600080fd5b82359150602083013567ffffffffffffffff8082111561052f57600080fd5b818501915085601f83011261054357600080fd5b813581811115610555576105556104ce565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f0116810190838211818310171561059b5761059b6104ce565b816040528281528860208487010111156105b457600080fd5b8260208601602083013760006020848301015280955050505050509250929050565b6000825160005b818110156105f757602081860181015185830152016105dd565b50600092019182525091905056fea2646970667358221220fd377c185926b3110b7e8a544f897646caf36a0e82b2629de851045e2a5f937764736f6c63430008100033"
	runCodeBytes := common.Hex2Bytes(runCode)
	// initCode includes constructor + some handling / prep + runCode
	initCode := "608060405234801561001057600080fd5b5061063b806100206000396000f3fe" + runCode
	initCodeBytes := common.Hex2Bytes(initCode)
	// create3 destination
	create3 := common.HexToAddress("0x6aA3D87e99286946161dCA02B97C5806fC5eD46F")
	// check that this matches the derived create3 destination
	derived := crypto.CreateAddress2(create2, salt, crypto.Keccak256Hash(initCodeBytes).Bytes())
	suite.Require().Equal(create3, derived)
	beforeBalance := suite.App.EvmKeeper.GetBalance(suite.Ctx, create3)
	// any address that has fees can call this deterministically
	addr := testutiltx.GenerateAddress()
	// evm keeper can mint with impunity. use it to generate gas fees
	err := suite.App.EvmKeeper.SetBalance(suite.Ctx, addr, big.NewInt(1000000000000000000))
	suite.Require().NoError(err)
	nonce, err := suite.App.AccountKeeper.GetSequence(suite.Ctx, sdk.AccAddress(addr.Bytes()))
	suite.Require().NoError(err)
	msg := ethtypes.NewMessage(
		addr, &create2, nonce, big.NewInt(0),
		2000000, big.NewInt(1), nil, nil,
		append(salt[:], initCodeBytes...), nil, true,
	)
	rsp, err := suite.App.EvmKeeper.ApplyMessage(suite.Ctx, msg, nil, true)
	suite.Require().NoError(err)
	suite.Require().False(rsp.Failed())
	// validate create3 destination
	acc := suite.App.AccountKeeper.GetAccount(suite.Ctx, create3.Bytes())
	suite.Require().NotNil(acc)
	ethAcc, ok := acc.(evmostypes.EthAccountI)
	suite.Require().True(ok)
	suite.Require().Equal(
		suite.App.EvmKeeper.GetNewContractNonce(suite.Ctx), ethAcc.GetSequence(),
	)
	suite.Require().Equal(crypto.Keccak256Hash(runCodeBytes), ethAcc.GetCodeHash())
	suite.Require().Equal(
		suite.App.EvmKeeper.GetCode(suite.Ctx, ethAcc.GetCodeHash()),
		runCodeBytes,
	)
	// no funds are generated
	suite.Require().Equal(beforeBalance, suite.App.EvmKeeper.GetBalance(suite.Ctx, create3))
}
