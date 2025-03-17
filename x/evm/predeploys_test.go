package evm_test

// This file contains tests for checking that:
// 1. All the contracts from DefaultPredeploys are included in genesis with nonce 1.
// 2. If a predeployed address has existing balance, it is retained.
// 3. CREATE2 can be used to deploy CREATE3 successfully.

import (
	"encoding/json"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmostypes "github.com/evmos/evmos/v16/types"
	evmosevmtypes "github.com/evmos/evmos/v16/x/evm/types"
	testutilcontracts "github.com/imua-xyz/imuachain/precompiles/testutil/contracts"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/evm/testdata"
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
	beforeSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, evmDenom).Amount
	targetBalance := sdkmath.NewInt(100)
	// set balance > 0 for all of the predeployed address, at genesis.
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
	expectedSupply := beforeSupply.Add(sdkmath.NewInt(int64(len(types.DefaultPredeploys))).Mul(targetBalance))
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
	afterSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, evmDenom).Amount
	suite.Require().Equal(expectedSupply, afterSupply)
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
	runCode := testdata.DeployedContract.BinRuntime
	runCodeBytes := runCode[:]
	// initCode includes constructor + some handling / prep + runCode
	initCode := testdata.DeployedContract.Bin
	initCodeBytes := initCode[:]
	// destination
	destination := common.HexToAddress("0xd26FCa167a00946c9D8eeCA081f6D0466fd5c7C7")
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
		[]byte(runCode),
	)
	// no funds are generated
	suite.Require().Equal(beforeBalance, suite.App.EvmKeeper.GetBalance(suite.Ctx, destination))
}

// the predeploys are blocked from receiving funds in app/app.go.
// however, this block should not deter them from forwarding funds
// to the contracts they deploy.
func (suite *KeeperTestSuite) TestBalanceForwarding() {
	// we will call Create3 predeploy with a random contract init code
	// and a msg.value > 0. we will then check the deployed contract
	// has received the value.
	create3Addr := common.HexToAddress("0x9fBB3DF7C40Da2e5A0dE984fFE2CCB7C47cd0ABf")
	// destination address
	txSalt := common.Hash{}
	unhashedSalt := append(suite.Address[:], txSalt[:]...)
	salt := crypto.Keccak256Hash(unhashedSalt)
	proxyRuntimeBytecode := common.FromHex("363d3d37363d34f0")
	proxyRuntimeBytecodeHash := crypto.Keccak256Hash(proxyRuntimeBytecode)
	proxyBytecode := common.FromHex("67363d3d37363d34f03d5260086018f3")
	proxyBytecodeHash := crypto.Keccak256Hash(proxyBytecode)
	proxyAddress := crypto.CreateAddress2(create3Addr, salt, proxyBytecodeHash.Bytes())
	destination := crypto.CreateAddress(proxyAddress, 1)
	// calculate the destination using ethCall
	method := testdata.Create3FactoryContract.ABI.Methods["getDeployed"]
	args, err := method.Inputs.Pack(suite.Address, txSalt)
	suite.Require().NoError(err)
	args = append(method.ID, args...)
	packedArgs := hexutil.Bytes(args)
	txArgs := evmosevmtypes.TransactionArgs{
		To:   &create3Addr,
		Data: &packedArgs,
	}
	marshalledArgs, err := json.Marshal(txArgs)
	suite.Require().NoError(err)
	ret, err := suite.QueryClientEVM.EthCall(sdk.WrapSDKContext(suite.Ctx), &evmosevmtypes.EthCallRequest{
		Args: marshalledArgs,
	})
	suite.Require().NoError(err)
	derivedDest := common.BytesToAddress(ret.Ret)
	suite.Require().Equal(destination.String(), derivedDest.String())
	// at t = 0, the contracts will not exist
	acc := suite.App.AccountKeeper.GetAccount(suite.Ctx, destination[:])
	suite.Require().Nil(acc)
	acc = suite.App.AccountKeeper.GetAccount(suite.Ctx, proxyAddress[:])
	suite.Require().Nil(acc)
	// we need the create3 interface to make the callArgs
	callArgs := testutilcontracts.CallArgs{
		ContractAddr: create3Addr,
		ContractABI:  testdata.Create3FactoryContract.ABI,
		PrivKey:      s.PrivKey,
	}
	callArgs = callArgs.
		WithMethodName("deploy").
		WithArgs(txSalt, []byte(testdata.DeployedContract.Bin[:])).
		WithAmount(common.Big1)
	_, _, err = testutilcontracts.Call(suite.Ctx, suite.App, callArgs)
	suite.Require().NoError(err)
	// post deployment, the 2 accounts should exist
	tests := []struct {
		Name     string
		Address  common.Address
		Balance  *big.Int
		CodeHash common.Hash
		Nonce    uint64
	}{
		{
			Name:     "destination",
			Address:  destination,
			Balance:  common.Big1,
			CodeHash: crypto.Keccak256Hash(testdata.DeployedContract.BinRuntime),
			Nonce:    suite.App.EvmKeeper.GetNewContractNonce(suite.Ctx),
		},
		{
			Name:     "proxy",
			Address:  proxyAddress,
			Balance:  common.Big0,
			CodeHash: proxyRuntimeBytecodeHash,
			// since the proxy deploys the destination contract,
			// the nonce of the proxy increases by 1
			Nonce: suite.App.EvmKeeper.GetNewContractNonce(suite.Ctx) + 1,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.Name, func() {
			acc = suite.App.AccountKeeper.GetAccount(suite.Ctx, tc.Address[:])
			suite.Require().NotNil(acc)
			ethAcc, ok := acc.(evmostypes.EthAccountI)
			suite.Require().True(ok)
			suite.Require().Equal(tc.Balance, suite.App.EvmKeeper.GetBalance(suite.Ctx, tc.Address))
			suite.Require().Equal(tc.CodeHash, ethAcc.GetCodeHash())
			suite.Require().Equal(tc.Nonce, ethAcc.GetSequence())
		})
	}
}
