package assets_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/precompiles/assets"

	"github.com/imua-xyz/imuachain/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"
)

var s *AssetsPrecompileSuite

type AssetsPrecompileSuite struct {
	testutil.BaseTestSuite

	precompile *assets.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(AssetsPrecompileSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "assets Precompile Suite")
}

func (s *AssetsPrecompileSuite) SetupTest() {
	s.DoSetupTest()
	precompile, err := assets.NewPrecompile(s.App.AssetsKeeper, s.App.AuthzKeeper)
	s.Require().NoError(err)
	s.precompile = precompile
}
