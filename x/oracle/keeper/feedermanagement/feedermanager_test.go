package feedermanagement

import (
	"math/big"
	"testing"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	gomock "go.uber.org/mock/gomock"

	. "github.com/smartystreets/goconvey/convey"
)

//go:generate mockgen -destination mock_cachereader_test.go -package feedermanagement github.com/ExocoreNetwork/exocore/x/oracle/keeper/feedermanagement CacheReader

func TestFeederManagement(t *testing.T) {
	Convey("compare FeederManager", t, func() {
		fm := NewFeederManager(nil)
		ctrl := gomock.NewController(t)
		c := NewMockCacheReader(ctrl)
		c.EXPECT().
			GetThreshold().
			Return(&threshold{big.NewInt(4), big.NewInt(1), big.NewInt(3)}).
			AnyTimes()
		Convey("add a new round", func() {
			ps1 := priceSource{deterministic: true, prices: []*PriceInfo{{Price: "123"}}}
			ps2 := ps1
			fm2 := *fm

			fm.rounds[1] = newRound(1, oracletypes.DefaultParams().TokenFeeders[1], 3, c)
			fm.rounds[1].PrepareForNextBlock(20)
			fm.sortedFeederIDs.add(1)
			fm.rounds[1].a.ds.AddPriceSource(&ps1, big.NewInt(1), "v1")

			fm2.rounds = make(map[int64]*round)
			fm2.sortedFeederIDs = make([]int64, 0)
			fm2.rounds[1] = newRound(1, oracletypes.DefaultParams().TokenFeeders[1], 3, c)
			fm2.rounds[1].PrepareForNextBlock(20)
			fm2.rounds[1].a.ds.AddPriceSource(&ps2, big.NewInt(5), "v1")
		})
	})
}
