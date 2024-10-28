package assets_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"

	"github.com/ExocoreNetwork/exocore/testutil"

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
	depositAmountNST := math.NewInt(32)
	s.App.AssetsKeeper.SetStakingAssetInfo(s.Ctx, &assetstypes.StakingAssetInfo{
		AssetBasicInfo: assetstypes.AssetInfo{
			Name:             "Native Restaking ETH",
			Symbol:           "NSTETH",
			Address:          "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			Decimals:         18,
			LayerZeroChainID: s.ClientChains[0].LayerZeroChainID,
			MetaInfo:         "native restaking token",
		},
		StakingTotalAmount: depositAmountNST,
	})
}
