package keeper_test

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	utils "github.com/imua-xyz/imuachain/utils"
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
	suite.SetAuthority = testutil.SetAuthorityTypeGenAddress
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
	authAccAddr := sdk.AccAddress(suite.Address.Bytes())

	// Get the initial value from storage (should be 1 from constructor)
	initialValue := suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(big.NewInt(1), initialValue)

	// Prepare the setValue function call
	// setValue(uint256 _value) - we'll set it to 42
	newValue := big.NewInt(42)
	callData := suite.encodeCall("setValue", newValue)

	// Get the authority address (the keeper's authority)
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	suite.Require().Equal(nonce, uint64(1))

	// Create the MsgCallContract message
	msg := &imuachainevmtypes.MsgCallContract{
		Authority: authAccAddr.String(),
		To:        contractAddr.Hex(),
		Data:      callData,
		GasLimit:  1_000_000,
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
	callData = suite.encodeCall("failingFunction")
	msg = &imuachainevmtypes.MsgCallContract{
		Authority: authAccAddr.String(),
		To:        contractAddr.Hex(),
		Data:      callData,
		GasLimit:  1_000_000,
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

	// try giving it funds
	amount := sdkmath.NewInt(5)
	newValue = big.NewInt(43)
	callData = suite.encodeCall("setValueWithAmount", newValue)
	msg = &imuachainevmtypes.MsgCallContract{
		Authority: authAccAddr.String(),
		To:        contractAddr.Hex(),
		Data:      callData,
		GasLimit:  1_000_000,
		Amount:    &amount,
	}
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, suite.PrivKey, nil, msg)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	callContractRes = suite.extractCallContractResponse(res)
	suite.Require().NotNil(callContractRes, "CallContract response should not be nil")
	suite.Require().Empty(callContractRes.VmError, "setValueWithAmount should not have VM error")
	updatedValue = suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(newValue, updatedValue)
	// check balance of the contract
	balance := suite.App.BankKeeper.GetBalance(suite.Ctx, sdk.AccAddress(contractAddr.Bytes()), utils.BaseDenom)
	suite.Require().Equal(amount.BigInt(), balance.Amount.BigInt())
	// TODO check balance of the authority should change by 5, when authority is not the executor.

	// check different caller case
	addr, priv := testutiltx.NewAddrKey()
	msg = &imuachainevmtypes.MsgCallContract{
		Authority: sdk.AccAddress(addr.Bytes()).String(),
		To:        contractAddr.Hex(),
		Data:      callData,
		GasLimit:  1_000_000,
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
		Authority: sdk.AccAddress(suite.Address.Bytes()).String(),
		To:        "",
		Data:      contractData,
		GasLimit:  1_000_000,
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

func (suite *MsgServerTestSuite) encodeCall(methodName string, args ...interface{}) []byte {
	// Get the method from the ABI
	method, ok := testdata.GroupDeployeeContract.ABI.Methods[methodName]
	suite.Require().True(ok, fmt.Sprintf("%s method should exist", methodName))

	// Pack the arguments
	packed, err := method.Inputs.Pack(args...)
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


type MsgServerTestSuiteWithGroupPolicy struct {
	MsgServerTestSuite
}

func TestMsgServerTestSuiteWithGroupPolicy(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuiteWithGroupPolicy))
}

func (suite *MsgServerTestSuiteWithGroupPolicy) getPolicyAddress() sdk.AccAddress {
	cred, err := authtypes.NewModuleCredential(
		group.ModuleName,
		[]byte{groupkeeper.GroupPolicyTablePrefix},
		sdk.Uint64ToBigEndian(1),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create module credential: %s", err))
	}
	return sdk.AccAddress(cred.Address().Bytes())
}

func (suite *MsgServerTestSuiteWithGroupPolicy) SetupTest() {
	suite.SetAuthority = testutil.SetAuthorityTypeGroupPolicyAddress
	suite.DoSetupTest()
	suite.Require().Equal(
		suite.getPolicyAddress().String(),
		suite.App.EvmKeeper.GetAuthority().String(),
		"authority should be set correctly",
	)
	for _, key := range suite.GroupKeys {
		addr := sdk.AccAddress(key.PubKey().Address().Bytes())
		testutil.FundAccountWithBaseDenom(suite.Ctx, suite.App.BankKeeper, addr, 1000000000000000000)
	}
}

func (suite *MsgServerTestSuiteWithGroupPolicy) findProposalId(events []abci.Event) uint64 {
	for _, event := range events {
		if event.Type == "cosmos.group.v1.EventSubmitProposal" {
			for _, attr := range event.Attributes {
				if attr.Key == "proposal_id" {
					proposalId, err := strconv.ParseUint(
						strings.ReplaceAll(attr.Value, "\"", ""), 10, 64,
					)
					if err != nil {
						panic(fmt.Sprintf("failed to parse proposal id: %s", err))
					}
					return proposalId
				}
			}
		}
	}
	return 0
}

func (suite *MsgServerTestSuiteWithGroupPolicy) checkProposalExecution(
	events []abci.Event,
) {
	for _, event := range events {
		if event.Type == "cosmos.group.v1.EventExec" {
			for _, attr := range event.Attributes {
				if attr.Key == "result" {
					status := strings.ReplaceAll(attr.Value, "\"", "")
					suite.Require().Equal(status, "PROPOSAL_EXECUTOR_RESULT_SUCCESS")
				}
			}
		}
	}
}

func (suite *MsgServerTestSuiteWithGroupPolicy) TestCallContract() {
	// create proposal by group member 1 to deploy the contract
	policyAddr := suite.getPolicyAddress()
	groupMember1 := suite.GroupKeys[0]
	groupMember1Addr := sdk.AccAddress(groupMember1.PubKey().Address().Bytes())
	baseMsg := &imuachainevmtypes.MsgCallContract{
		Authority: policyAddr.String(),
		To: "",
		Data: testdata.GroupDeployeeContract.Bin,
		GasLimit: 1_000_000,
	}
	anyMsg, err := codectypes.NewAnyWithValue(baseMsg)
	if err != nil {
		panic(fmt.Sprintf("failed to create any value: %s", err))
	}
	proposal := &group.MsgSubmitProposal{
		GroupPolicyAddress: policyAddr.String(),
		Proposers: []string{groupMember1Addr.String()},
		Metadata: "Deploy GroupDeployee contract",
		Title: "Deploy GroupDeployee contract",
		Summary: "Deploy GroupDeployee contract",
		Messages: []*codectypes.Any{anyMsg},
	}
	res, err := testutil.DeliverTx(suite.Ctx, suite.App, groupMember1, nil, proposal)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	proposalId := suite.findProposalId(res.Events)
	suite.Require().NotEqual(proposalId, uint64(0), "proposal id should not be 0")
	suite.Commit()
	// now we need to vote on the proposal
	// we use members 2 and 3 to vote on the proposal
	for i := 1; i < len(suite.GroupKeys); i++ {
		groupMember := suite.GroupKeys[i]
		groupMemberAddr := sdk.AccAddress(groupMember.PubKey().Address().Bytes())
		vote := &group.MsgVote{
			ProposalId: proposalId,
			Voter: groupMemberAddr.String(),
			Option: group.VOTE_OPTION_YES,
		}
		res, err = testutil.DeliverTx(suite.Ctx, suite.App, groupMember, nil, vote)
		suite.Require().NoError(err)
		suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	}
	// now we need to execute the proposal
	execute := &group.MsgExec{
		ProposalId: proposalId,
		Executor: groupMember1Addr.String(),
	}
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, groupMember1, nil, execute)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	suite.checkProposalExecution(res.Events)
	// now validate contract existence
	deployerAddr := common.BytesToAddress(policyAddr.Bytes())
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, deployerAddr)
	nonce = nonce - 1
	contractAddr := crypto.CreateAddress(deployerAddr, nonce)
	acct := suite.App.EvmKeeper.GetAccountWithoutBalance(suite.Ctx, contractAddr)
	suite.Require().NotNil(acct, "contract account should exist")
	suite.Require().True(acct.IsContract(), "account should be a contract")
	code := suite.App.EvmKeeper.GetCode(suite.Ctx, common.BytesToHash(acct.CodeHash))
	suite.Require().NotEmpty(code, "contract code should not be empty")
	// now, call the contract
	callData := suite.encodeCall("setValue", big.NewInt(42))
	msg := &imuachainevmtypes.MsgCallContract{
		Authority: policyAddr.String(),
		To: contractAddr.Hex(),
		Data: callData,
		GasLimit: 1_000_000,
	}
	anyMsg, err = codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(fmt.Sprintf("failed to create any value: %s", err))
	}
	proposal = &group.MsgSubmitProposal{
		GroupPolicyAddress: policyAddr.String(),
		Proposers: []string{groupMember1Addr.String()},
		Metadata: "Call setValue function with group policy",
		Title: "Call setValue function with group policy",
		Summary: "Call setValue function with group policy",
		Messages: []*codectypes.Any{anyMsg},
	}
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, groupMember1, nil, proposal)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	proposalId = suite.findProposalId(res.Events)
	suite.Require().NotEqual(proposalId, uint64(0), "proposal id should not be 0")
	suite.Commit()
	// now we need to vote on the proposal
	for i := 1; i < len(suite.GroupKeys); i++ {
		groupMember := suite.GroupKeys[i]
		groupMemberAddr := sdk.AccAddress(groupMember.PubKey().Address().Bytes())
		vote := &group.MsgVote{
			ProposalId: proposalId,
			Voter: groupMemberAddr.String(),
			Option: group.VOTE_OPTION_YES,
		}
		res, err = testutil.DeliverTx(suite.Ctx, suite.App, groupMember, nil, vote)
		suite.Require().NoError(err)
		suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	}
	// now we need to execute the proposal
	execute = &group.MsgExec{
		ProposalId: proposalId,
		Executor: groupMember1Addr.String(),
	}
	res, err = testutil.DeliverTx(suite.Ctx, suite.App, groupMember1, nil, execute)
	suite.Require().NoError(err)
	suite.Require().True(res.IsOK(), "transaction should succeed: %s", res.Log)
	suite.checkProposalExecution(res.Events)
	// now validate storage value
	storageValue := suite.getStorageValue(contractAddr, common.Hash{})
	suite.Require().Equal(big.NewInt(42), storageValue)
	// now validate nonce
	// 0th nonce was deployment
	// 1st nonce was call
	// so the value will be 2
	nonce = suite.App.EvmKeeper.GetNonce(suite.Ctx, deployerAddr)
	suite.Require().Equal(nonce, uint64(2))
	// check contract nonce too
	contractNonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, contractAddr)
	suite.Require().Equal(contractNonce, uint64(1))
	// check owner of the contract
	owner := common.BytesToAddress(
		suite.App.EvmKeeper.GetState(
			suite.Ctx, contractAddr, common.BigToHash(big.NewInt(1)),
		).Bytes(),
	)
	suite.Require().Equal(deployerAddr, owner)
	
}