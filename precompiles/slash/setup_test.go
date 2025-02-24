package slash_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"

	"github.com/imua-xyz/imuachain/precompiles/slash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"
)

var s *SlashPrecompileTestSuite

type SlashPrecompileTestSuite struct {
	testutil.BaseTestSuite
	precompile *slash.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(SlashPrecompileTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Slash Precompile Suite")
}

func (s *SlashPrecompileTestSuite) SetupTest() {
	s.DoSetupTest()
	precompile, err := slash.NewPrecompile(s.App.AssetsKeeper, s.App.ImSlashKeeper, s.App.AuthzKeeper)
	s.Require().NoError(err)
	s.precompile = precompile
}
