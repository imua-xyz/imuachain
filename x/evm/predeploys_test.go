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

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmostypes "github.com/evmos/evmos/v16/types"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/evm/types"
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

// Tests if create2 can be used to deploy another contract successfully.
func (suite *KeeperTestSuite) TestCreate2Deployment() {
	// contract to call
	create2 := common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")
	// blank salt
	salt := common.Hash{}
	// runCode is the code fetched via eth_getCode.
	// it can be used directly in a predeploy but not to create a new contract, because
	// it does not have the constructor.
	runCode := "6080604052348015600f57600080fd5b506004361060285760003560e01c806301ffc9a714602d575b600080fd5b604e60383660046062565b6001600160e01b0319166301ffc9a760e01b1490565b604051901515815260200160405180910390f35b600060208284031215607357600080fd5b81356001600160e01b031981168114608a57600080fd5b939250505056fea2646970667358221220a2767d05726e53026a408e98af9d44da774b561ee9ba08de62018a7745fb373564736f6c63430008170033"
	runCodeBytes := common.Hex2Bytes(runCode)
	// initCode includes constructor + some handling / prep + runCode
	initCode := "608060405234801561001057600080fd5b5060c78061001f6000396000f3fe" + runCode
	initCodeBytes := common.Hex2Bytes(initCode)
	// destination
	destination := common.HexToAddress("0x5c4A7c0B21C8D4166d891A9372Ce016401980Ae0")
	// check that this matches the derived create3 destination
	derived := crypto.CreateAddress2(create2, salt, crypto.Keccak256Hash(initCodeBytes).Bytes())
	suite.Require().Equal(destination.String(), derived.String())
	beforeBalance := suite.App.EvmKeeper.GetBalance(suite.Ctx, destination)
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
	// validate destination
	acc := suite.App.AccountKeeper.GetAccount(suite.Ctx, destination.Bytes())
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
	suite.Require().Equal(beforeBalance, suite.App.EvmKeeper.GetBalance(suite.Ctx, destination))
}
