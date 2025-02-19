package reward_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"

	"github.com/imua-xyz/imuachain/precompiles/reward"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"
)

var s *RewardPrecompileTestSuite

type RewardPrecompileTestSuite struct {
	testutil.BaseTestSuite
	precompile *reward.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(RewardPrecompileTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reward Precompile Suite")
}

func (s *RewardPrecompileTestSuite) SetupTest() {
	s.DoSetupTest()
	precompile, err := reward.NewPrecompile(s.App.AssetsKeeper, s.App.RewardKeeper, s.App.AuthzKeeper)
	s.Require().NoError(err)
	s.precompile = precompile
}
