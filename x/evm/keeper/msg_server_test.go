package keeper_test

import (
	"math/big"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/evm/keeper/testdata"
	imuachainevmtypes "github.com/imua-xyz/imuachain/x/evm/types"
)

type MsgServerTestSuite struct {
	testutil.BaseTestSuite
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}

func (suite *MsgServerTestSuite) SetupTest() {
	// Set up the test suite
	suite.SetAuthority = true
	suite.DoSetupTest()
	// check authority is correctly set
	suite.Require().Equal(
		sdk.AccAddress(suite.Address.Bytes()),
		suite.App.EvmKeeper.GetAuthority(),
		"authority should be set correctly",
	)
	// the authority address is already funded in the setup
}

func (suite *MsgServerTestSuite) TestCallContract() {
	// Deploy the GroupDeployee contract
	contractAddr := suite.deployGroupDeployee()

	// Get the initial value from storage (should be 1 from constructor)
	initialValue := suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(big.NewInt(1), initialValue)

	// Prepare the setValue function call
	// setValue(uint256 _value) - we'll set it to 42
	newValue := big.NewInt(42)
	callData := suite.encodeSetValueCall(newValue)

	// Get the authority address (the keeper's authority)
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	suite.Require().Equal(nonce, uint64(1))

	// Create the MsgCallContract message
	msg := &imuachainevmtypes.MsgCallContract{
		Authority:       sdk.AccAddress(suite.Address.Bytes()).String(),
		ContractAddress: contractAddr.Hex(),
		Data:            common.Bytes2Hex(callData),
		GasLimit:        1_000_000,
	}

	res, err := testutil.DeliverTx(suite.Ctx, suite.App, suite.PrivKey, nil, msg)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	callContractRes := suite.extractCallContractResponse(res)
	suite.Require().NotNil(callContractRes, "CallContract response should not be nil")
	suite.Require().Empty(callContractRes.VmError, "contract deployment should not have VM error")

	// Verify the storage value was updated
	updatedValue := suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(newValue, updatedValue)

	// Check nonce - it should be incremented by the AnteHandler
	postNonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	suite.Require().Equal(nonce+1, postNonce, "nonce should be incremented by AnteHandler")

	// check failingFunction case
	callData = suite.encodeFailingFunctionCall()
	msg = &imuachainevmtypes.MsgCallContract{
		Authority:       sdk.AccAddress(suite.Address.Bytes()).String(),
		ContractAddress: contractAddr.Hex(),
		Data:            common.Bytes2Hex(callData),
		GasLimit:        1_000_000,
	}
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, suite.PrivKey, nil, msg)
	// when an EVM error is returned, the response of err is nil
	suite.Require().Nil(err)
	callContractRes = suite.extractCallContractResponse(res)
	// when evm error is returned, the error is wrapped in the response.
	suite.Require().NotNil(callContractRes, "CallContract response should not be nil")
	suite.Require().NotEmpty(callContractRes.VmError, "contract deployment should have VM error")
	suite.Require().Contains(callContractRes.VmError, vm.ErrExecutionReverted.Error())
	// now decode the error
	cause, err := abi.UnpackRevert(common.CopyBytes(callContractRes.Ret))
	suite.Require().NoError(err)
	suite.Require().Contains(cause, "This function is failing")
	// Note: When DeliverTx returns an error, BroadcastTxBytes returns an empty response
	// with Code == 0, so res.IsOK() would be true. We only check the error here.

	// check different caller case
	addr, priv := testutiltx.NewAddrKey()
	msg = &imuachainevmtypes.MsgCallContract{
		Authority:       sdk.AccAddress(addr.Bytes()).String(),
		ContractAddress: contractAddr.Hex(),
		Data:            common.Bytes2Hex(callData),
		GasLimit:        1_000_000,
	}
	// fund the address
	testutil.FundAccountWithBaseDenom(
		suite.Ctx, suite.App.BankKeeper, sdk.AccAddress(addr.Bytes()),
		1000000000000000000,
	)
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, priv, nil, msg)
	suite.Require().ErrorContains(err, govtypes.ErrInvalidSigner.Error())
	callContractRes = suite.extractCallContractResponse(res)
	// cosmos error is returned, so the response should be nil
	suite.Require().Nil(callContractRes, "CallContract response should be nil")
}

func (suite *MsgServerTestSuite) deployGroupDeployee() common.Address {
	// Deploy using the authority address so it becomes the owner
	// This allows CallContract (which uses authority as sender) to call setValue
	authorityAddr := suite.App.EvmKeeper.GetAuthority()
	deployerAddr := common.BytesToAddress(authorityAddr.Bytes())
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, deployerAddr)
	// check nonce is 0
	suite.Require().Equal(nonce, uint64(0))

	// Prepare contract deployment data (bytecode)
	contractData := []byte(testdata.GroupDeployeeContract.Bin)

	// Create a deployment message
	msg := &imuachainevmtypes.MsgCallContract{
		Authority:       sdk.AccAddress(suite.Address.Bytes()).String(),
		ContractAddress: "",
		Data:            common.Bytes2Hex(contractData),
		GasLimit:        1_000_000,
	}

	res, err := testutil.DeliverTx(suite.Ctx, suite.App, suite.PrivKey, nil, msg)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)

	// Extract and validate the message response
	callContractRes := suite.extractCallContractResponse(res)
	suite.Require().NotNil(callContractRes, "CallContract response should not be nil")
	// Validate that the contract deployment was successful (no VM error)
	suite.Require().Empty(callContractRes.VmError, "contract deployment should not have VM error")

	// Calculate the contract address
	contractAddr := crypto.CreateAddress(deployerAddr, nonce)

	// Verify the contract was deployed by checking if code exists
	acct := suite.App.EvmKeeper.GetAccountWithoutBalance(suite.Ctx, contractAddr)
	suite.Require().NotNil(acct, "contract account should exist")
	suite.Require().True(acct.IsContract(), "account should be a contract")

	code := suite.App.EvmKeeper.GetCode(suite.Ctx, common.BytesToHash(acct.CodeHash))
	suite.Require().NotEmpty(code, "contract code should not be empty")

	// validate that the owner is set correctly
	owner := suite.App.EvmKeeper.GetState(suite.Ctx, contractAddr, common.BigToHash(big.NewInt(1)))
	ownerHex := common.BytesToAddress(owner.Bytes()).Hex()
	authorityHex := common.BytesToAddress(authorityAddr.Bytes()).Hex()
	suite.Require().Equal(authorityHex, ownerHex)

	// check nonce
	postNonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, deployerAddr)
	suite.Require().Equal(nonce+1, postNonce)

	return contractAddr
}

func (suite *MsgServerTestSuite) encodeSetValueCall(value *big.Int) []byte {
	// Get the setValue method from the ABI
	method, ok := testdata.GroupDeployeeContract.ABI.Methods["setValue"]
	suite.Require().True(ok, "setValue method should exist")

	// Pack the arguments
	packed, err := method.Inputs.Pack(value)
	suite.Require().NoError(err)

	// Prepend the method ID (first 4 bytes of the keccak256 hash of the method signature)
	callData := append(method.ID, packed...)
	return callData
}

func (suite *MsgServerTestSuite) encodeFailingFunctionCall() []byte {
	// Get the failingFunction method from the ABI
	method, ok := testdata.GroupDeployeeContract.ABI.Methods["failingFunction"]
	suite.Require().True(ok, "failingFunction method should exist")

	// Pack the arguments
	packed, err := method.Inputs.Pack()
	suite.Require().NoError(err)

	callData := append(method.ID, packed...)
	return callData
}

func (suite *MsgServerTestSuite) getStorageValue(contractAddr common.Address, slot common.Hash) *big.Int {
	// Get the storage value at slot 0 (where the `value` variable is stored)
	// In Solidity, the first state variable is at slot 0
	storageValue := suite.App.EvmKeeper.GetState(suite.Ctx, contractAddr, slot)
	return storageValue.Big()
}

// extractCallContractResponse extracts the MsgEthereumTxResponse from the transaction response.
// CallContract internally uses ApplyMessage which returns MsgEthereumTxResponse.
// Returns nil if the response is empty or cannot be extracted (e.g., when there's a Cosmos-level error).
func (suite *MsgServerTestSuite) extractCallContractResponse(res abci.ResponseDeliverTx) *evmtypes.MsgEthereumTxResponse {
	// If the response data is empty, return nil
	if len(res.Data) == 0 {
		return nil
	}

	// First, try to decode directly using DecodeTxResponse (used for EVM transactions)
	// This works even when there's a VM error, as the response is still encoded in res.Data
	ethRes, err := evmtypes.DecodeTxResponse(res.Data)
	if err == nil {
		return ethRes
	}

	// If direct decoding fails, try extracting from TxMsgData (standard SDK message response format)
	var txData sdk.TxMsgData
	err = suite.App.AppCodec().Unmarshal(res.Data, &txData)
	if err != nil {
		// If unmarshaling fails, the response might be in a different format
		return nil
	}

	// If there are no message responses, return nil
	if len(txData.MsgResponses) == 0 {
		return nil
	}

	// Extract the first message response (should be the CallContract response)
	// CallContract returns MsgEthereumTxResponse internally
	var ethResFromMsg evmtypes.MsgEthereumTxResponse
	err = proto.Unmarshal(txData.MsgResponses[0].Value, &ethResFromMsg)
	if err != nil {
		// If unmarshaling fails, return nil
		return nil
	}

	return &ethResFromMsg
}
