package feedermanagement

import (
	"math/big"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/testdata"
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

			fm.rounds[1] = newRound(1, testdata.DefaultParamsForTest().TokenFeeders[1], 3, c, defaultAggMedian)
			fm.rounds[1].PrepareForNextBlock(20)
			fm.sortedFeederIDs.add(1)
			fm.rounds[1].a.ds.AddPriceSource(&ps1, big.NewInt(1), "v1")

			fm2.rounds = make(map[int64]*round)
			fm2.sortedFeederIDs = make([]int64, 0)
			fm2.rounds[1] = newRound(1, testdata.DefaultParamsForTest().TokenFeeders[1], 3, c, defaultAggMedian)
			fm2.rounds[1].PrepareForNextBlock(20)
			fm2.sortedFeederIDs.add(1)
			fm2.rounds[1].a.ds.AddPriceSource(&ps2, big.NewInt(1), "v1")

			So(fm.Equals(&fm2), ShouldBeTrue)
		})
	})
	Convey("check copy results", t, func() {
		ctrl := gomock.NewController(t)
		c := NewMockCacheReader(ctrl)
		c.EXPECT().
			GetThreshold().
			Return(&threshold{big.NewInt(4), big.NewInt(1), big.NewInt(3)}).
			AnyTimes()

		// feedermanager
		Convey("copy of feedermanager", func() {
			f := tData.NewFeederManager(c)
			f.updateCheckTx()
			fc := f.fCheckTx
			f.fCheckTx = nil
			So(f.Equals(fc), ShouldBeTrue)
		})
		Convey("copy of round", func() {
			r := tData.NewRound(c)
			rc := r.CopyForCheckTx()
			So(r.Equals(rc), ShouldBeTrue)
		})
		Convey("copy of aggregagtor", func() {
			a := tData.NewAggregator(true)
			ac := a.CopyForCheckTx()
			So(a.Equals(ac), ShouldBeTrue)
		})
		Convey("copy of recordsValidators", func() {
			v := tData.NewRecordsValidators(true)
			vc := v.Cpy()
			So(v.Equals(vc), ShouldBeTrue)
		})
		Convey("copy of recordsDSs", func() {
			dss := tData.NewRecordsDSs(true)
			dssc := dss.Cpy()
			So(dss.Equals(dssc), ShouldBeTrue)
		})
		Convey("copy of recordsDS", func() {
			ds := tData.NewRecordsDS(true)
			dsc := ds.Cpy()
			So(ds.Equals(dsc), ShouldBeTrue)
		})
		Convey("copy of priceValidator", func() {
			pv := tData.NewPriceValidator(true)
			pvc := pv.Cpy()
			So(pv.Equals(pvc), ShouldBeTrue)
		})
		Convey("copy of priceSource", func() {
			ps := tData.NewPriceSource(true, true)
			psc := ps.Cpy()
			So(ps.Equals(psc), ShouldBeTrue)
		})
		Convey("copy of pricePower", func() {
			pw := tData.NewPricePower()
			pwc := pw.Cpy()
			So(pw.Equals(pwc), ShouldBeTrue)
		})
	})
}
