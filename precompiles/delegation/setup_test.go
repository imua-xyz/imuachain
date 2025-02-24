package delegation_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/precompiles/delegation"
	"github.com/imua-xyz/imuachain/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"
)

var s *DelegationPrecompileSuite

type DelegationPrecompileSuite struct {
	testutil.BaseTestSuite

	precompile *delegation.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(DelegationPrecompileSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "delegation Precompile Suite")
}

func (s *DelegationPrecompileSuite) SetupTest() {
	s.DoSetupTest()
	precompile, err := delegation.NewPrecompile(s.App.AssetsKeeper, s.App.DelegationKeeper, s.App.AuthzKeeper)
	s.Require().NoError(err)
	s.precompile = precompile
}
