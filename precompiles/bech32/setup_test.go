package bech32_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/precompiles/bech32"

	"github.com/imua-xyz/imuachain/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"
)

var s *Bech32PrecompileSuite

type Bech32PrecompileSuite struct {
	testutil.BaseTestSuite
	precompile *bech32.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(Bech32PrecompileSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "bech32 Precompile Suite")
}

func (s *Bech32PrecompileSuite) SetupTest() {
	s.DoSetupTest()
	precompile, err := bech32.NewPrecompile(s.App.AuthzKeeper)
	s.Require().NoError(err)
	s.precompile = precompile
}
