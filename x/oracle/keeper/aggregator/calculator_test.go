package aggregator

import (
	"math/big"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
	1-10, 	2-12, 	3-15

ps1: 1-10,	2-12
ps2: 		2-12,	3-15
ps3: 1-10,	2-11(m)
---
ps4: 		2-12,	3-19(m)
ps5: 1-10, 			3-19(m)
----
ps1, ps2, ps3, ps4 ---> 2-12
ps2, ps2, ps3, ps5 ---> 1-10
*/
func TestCalculator(t *testing.T) {
	one := big.NewInt(1)
	Convey("fill prices into calculator", t, func() {
		c := newCalculator(5, big.NewInt(4))
		Convey("fill prices from single deterministic source", func() {
			c.fillPrice(pS1, "v1", one) // 1-10, 2-12
			c.fillPrice(pS2, "v2", one) // 2-12, 3-15
			c.fillPrice(pS3, "v3", one) // 1-10, 2-11
			Convey("consensus on detid=2 and price=12", func() {
				confirmed := c.fillPrice(pS4, "v4", one) // 2-12, 3-19
				So(confirmed[0].detID, ShouldEqual, "2")
				So(confirmed[0].price, ShouldResemble, "12")
			})
			Convey("consensus on detid=1 and price=10", func() {
				confirmed := c.fillPrice(pS5, "v5", one) // 1-10, 3-19
				So(confirmed[0].detID, ShouldEqual, "1")
				So(confirmed[0].price, ShouldResemble, "10")

				confirmed = c.fillPrice(pS4, "v4", one)
				So(confirmed, ShouldBeNil)
			})
		})
		Convey("fill prices from multiple deterministic sources", func() {
			c.fillPrice(pS21, "v1", one)
			c.fillPrice(pS22, "v2", one)
			c.fillPrice(pS23, "v3", one)
			Convey("consensus on both source 1 and source 2", func() {
				confirmed := c.fillPrice(pS24, "v4", one)
				So(len(confirmed), ShouldEqual, 2)
				i := 0
				if confirmed[0].sourceID == 2 {
					i = 1
				}
				So(confirmed[i].detID, ShouldEqual, "2")
				So(confirmed[i].price, ShouldResemble, "12")

				So(confirmed[1-i].detID, ShouldEqual, "3")
				So(confirmed[1-i].price, ShouldResemble, "15")
			})
			Convey("consenus on source 1 only", func() {
				confirmed := c.fillPrice(pS25, "v5", one)
				So(len(confirmed), ShouldEqual, 1)
				So(confirmed[0].detID, ShouldEqual, "1")
				So(confirmed[0].price, ShouldResemble, "10")
			})
		})
	})
}
