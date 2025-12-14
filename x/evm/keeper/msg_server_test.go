package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/testutil"
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
	suite.DoSetupTest()

	// Note: In a production scenario, the authority would be set up as a group policy address
	// in genesis. For this test, we use the app's default authority (gov module).
	// The group policy address for a single address group with ID 1 would be:
	// groupPolicyId := uint64(1)
	// policyAddr := address.Module("cosmos.group.v1.GroupPolicy", []byte("1"))
	// authority := sdk.AccAddress(policyAddr)
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
	authorityAddr := suite.App.EvmKeeper.GetAuthority()

	// Call CallContract
	req := &imuachainevmtypes.MsgCallContract{
		Authority:       authorityAddr.String(),
		ContractAddress: contractAddr.Hex(),
		Data:            common.Bytes2Hex(callData),
	}

	resp, err := suite.App.EvmKeeper.CallContract(sdk.WrapSDKContext(suite.Ctx), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify the storage value was updated
	updatedValue := suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(newValue, updatedValue)
}

func (suite *MsgServerTestSuite) deployGroupDeployee() common.Address {
	// Deploy using the authority address so it becomes the owner
	// This allows CallContract (which uses authority as sender) to call setValue
	authorityAddr := suite.App.EvmKeeper.GetAuthority()
	deployerAddr := common.BytesToAddress(authorityAddr.Bytes())
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, deployerAddr)

	// Prepare contract deployment data (bytecode)
	contractData := []byte(testdata.GroupDeployeeContract.Bin)

	// Create a deployment message
	msg := ethtypes.NewMessage(
		deployerAddr,
		nil, // to is nil for contract creation
		nonce,
		big.NewInt(0), // value
		2000000,       // gas limit
		big.NewInt(1), // gas price
		nil,           // gas fee cap
		nil,           // gas tip cap
		contractData,  // data (contract bytecode)
		nil,           // access list
		true,          // is fake
	)

	// Apply the message to deploy the contract
	resp, err := suite.App.EvmKeeper.ApplyMessage(suite.Ctx, msg, evmtypes.NewNoOpTracer(), true)
	suite.Require().NoError(err)
	suite.Require().False(resp.Failed(), "contract deployment failed: %s", resp.VmError)

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

	return contractAddr
}

func (suite *MsgServerTestSuite) encodeSetValueCall(value *big.Int) []byte {
	// Get the setValue method from the ABI
	method := testdata.GroupDeployeeContract.ABI.Methods["setValue"]

	// Pack the arguments
	packed, err := method.Inputs.Pack(value)
	suite.Require().NoError(err)

	// Prepend the method ID (first 4 bytes of the keccak256 hash of the method signature)
	callData := append(method.ID, packed...)
	return callData
}

func (suite *MsgServerTestSuite) getStorageValue(contractAddr common.Address, slot common.Hash) *big.Int {
	// Get the storage value at slot 0 (where the `value` variable is stored)
	// In Solidity, the first state variable is at slot 0
	storageValue := suite.App.EvmKeeper.GetState(suite.Ctx, contractAddr, slot)
	return storageValue.Big()
}
