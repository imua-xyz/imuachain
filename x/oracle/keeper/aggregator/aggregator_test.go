package aggregator

import (
	"math/big"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAggregator(t *testing.T) {
	Convey("fill prices into aggregator", t, func() {
		a := newAggregator(5, big.NewInt(4))
		// a.fillPrice(pS1, "v1", one) //v1:{1, 2}

		Convey("fill v1's report", func() {
			a.fillPrice(pS1, "v1", one) // v1:{1, 2}
			report := a.getReport("v1")
			So(report.prices[1].price, ShouldEqual, "")
			Convey("fill v2's report", func() {
				a.fillPrice(pS2, "v2", one)
				report := a.getReport("v2")
				So(report.prices[1].price, ShouldEqual, "")
				Convey("fill more v1's report", func() {
					a.fillPrice(pS21, "v1", one)
					report := a.getReport("v1")
					So(report.prices[1].price, ShouldEqual, "")
					So(report.prices[2].price, ShouldEqual, "")
					Convey("confirm deterministic source_1 and source 2", func() {
						a.confirmDSPrice([]*confirmedPrice{
							{
								sourceID:  1,
								detID:     "9",
								price:     "10",
								timestamp: "-",
							},
							{
								sourceID:  2,
								detID:     "3",
								price:     "20",
								timestamp: "-",
							},
						})
						reportV1 := a.getReport("v1")
						reportV2 := a.getReport("v2")
						So(reportV1.prices[1].price, ShouldResemble, "10")
						So(reportV1.prices[1].detRoundID, ShouldEqual, "9")

						So(reportV2.prices[1].price, ShouldResemble, "10")
						So(reportV2.prices[1].detRoundID, ShouldEqual, "9")

						So(reportV1.prices[2].price, ShouldResemble, "20")
						So(reportV1.prices[2].detRoundID, ShouldEqual, "3")

						// current implementation only support v1's single source
						Convey("aggregate after all source confirmed", func() {
							a.fillPrice(pS6, "v3", one)
							a.aggregate() // v1:{s1:9-10, s2:3-20}:15, v2:{s1:9-10}:10
							So(a.getReport("v1").price, ShouldEqual, "15")
							So(a.getReport("v2").price, ShouldEqual, "10")
							So(a.getReport("v3").price, ShouldEqual, "20")
							So(a.finalPrice, ShouldEqual, "15")
						})
					})
				})
			})
		})
	})
}
