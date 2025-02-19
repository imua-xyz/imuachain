package aggregator

import (
	"testing"

	"github.com/imua-xyz/imuachain/x/oracle/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFilter(t *testing.T) {
	Convey("test aggregator_filter", t, func() {
		f := newFilter(3, 5)
		ptd1 := newPTD("1", "600000")
		ptd2 := newPTD("2", "600050")
		ptd3 := newPTD("3", "600070")
		ptd4 := newPTD("4", "601000")
		ptd5 := newPTD("5", "602000")
		ptd6 := newPTD("6", "603000")

		ps1 := &types.PriceSource{
			SourceID: 1,
			Prices: []*types.PriceTimeDetID{
				ptd1,
				ptd2,
			},
		}

		ps := []*types.PriceSource{ps1}
		msg := &types.MsgCreatePrice{
			Creator:    "v1",
			FeederID:   1,
			Prices:     ps,
			BasedBlock: 10,
			Nonce:      1,
		}
		l4c, l4a := f.filtrate(msg)

		Convey("add first valid msg", func() {
			So(l4c, ShouldResemble, ps)
			So(l4a, ShouldResemble, ps)
		})

		Convey("add duplicate nonce msg", func() {
			ps1.Prices[0] = ptd3
			l4c, l4a = f.filtrate(msg)
			So(l4c, ShouldBeNil)
			So(l4a, ShouldBeNil)
		})

		Convey("add duplicate detId", func() {
			msg.Nonce = 2
			l4c, l4a = f.filtrate(msg)
			Convey("add with new nonce", func() {
				So(l4c, ShouldBeNil)
				So(l4a, ShouldBeNil)
			})
			Convey("update with new detId but use duplicate nonce", func() {
				msg.Nonce = 2
				ps1.Prices[0] = ptd3
				l4c, l4a := f.filtrate(msg)
				So(l4c, ShouldBeNil)
				So(l4a, ShouldBeNil)
			})
		})

		Convey("add new detId with new nonce", func() {
			msg.Nonce = 2
			ps1.Prices[0] = ptd3
			l4c, l4a = f.filtrate(msg)
			ps1.Prices = ps1.Prices[:1]
			ps1.Prices[0] = ptd3
			psReturn := []*types.PriceSource{ps1}
			So(l4c, ShouldResemble, psReturn)
			So(l4a, ShouldResemble, psReturn)
		})

		Convey("add too many nonce", func() {
			msg.Nonce = 2
			ps1.Prices[0] = ptd3
			f.filtrate(msg)

			msg.Nonce = 3
			ps1.Prices[0] = ptd4
			l4c, _ = f.filtrate(msg)
			So(l4c[0].Prices, ShouldContain, ptd4)

			msg.Nonce = 4
			ps1.Prices[0] = ptd5
			l4c, _ = f.filtrate(msg)
			So(l4c, ShouldBeNil)
		})

		Convey("add too many DetIds", func() {
			msg.Nonce = 2
			ps1.Prices = []*types.PriceTimeDetID{ptd3, ptd4, ptd5, ptd6}
			l4c, l4a = f.filtrate(msg)
			So(l4c, ShouldResemble, l4a)
			So(l4c[0].Prices, ShouldContain, ptd3)
			So(l4c[0].Prices, ShouldContain, ptd4)
			So(l4c[0].Prices, ShouldContain, ptd5)
			So(l4c[0].Prices, ShouldNotContain, ptd6)
		})
	})
}
