package precompiles_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

	"github.com/imua-xyz/imuachain/precompiles/testdata"
	testutilprecompiles "github.com/imua-xyz/imuachain/precompiles/testutil"
	testutilcontracts "github.com/imua-xyz/imuachain/precompiles/testutil/contracts"

	imuaevmtypes "github.com/imua-xyz/imuachain/x/evm/types"
)

var s *UnknownMethodSuite

type UnknownMethodSuite struct {
	testutil.BaseTestSuite
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(UnknownMethodSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "unknown method Precompile Suite")
}

func (s *UnknownMethodSuite) SetupTest() {
	s.DoSetupTest()
}

// TestUnknownMethod tests the unknown method scenario in which a call to an unknown method
// on the precompile is made. p.IsTransaction panics upon receiving such a method name; however,
// it is only called after the method is validated. Hence, the node will not panic for such txs.
func (s *UnknownMethodSuite) TestUnknownMethod() {
	caller, err := s.DeployContract(testdata.UnknownMethodCallerContract)
	s.Require().NoError(err)

	callArgs := testutilcontracts.CallArgs{
		ContractAddr: caller,
		ContractABI:  testdata.UnknownMethodCallerContract.ABI,
		PrivKey:      s.PrivKey,
	}

	eventName := "UnknownMethodResult"
	event, ok := testdata.UnknownMethodCallerContract.ABI.Events[eventName]
	s.Require().True(ok)

	data, err := event.Inputs.Pack(false)
	s.Require().NoError(err)
	logCheckArgs := testutilprecompiles.LogCheckArgs{
		ABIEvents: map[string]abi.Event{
			eventName: event,
		},
	}.WithExpEvents(eventName).
		WithExpPass(true).
		WithExpData(data)

	// do not use all precompiles since those include Berlin fork
	// versions, which do not care if you call unknown methods
	// addrs := s.App.EvmKeeper.GetAvailablePrecompileAddrs()
	addrs := imuaevmtypes.ImuachainAvailableEVMExtensions
	for _, addrStr := range addrs {
		addr := common.HexToAddress(addrStr)
		args := callArgs.WithMethodName("callUnknownMethod").WithArgs(addr)
		_, _, err := testutilcontracts.CallContractAndCheckLogs(s.Ctx, s.App, args, logCheckArgs)
		s.Require().NoError(err, "addr: %s", addr.Hex())
	}

}
