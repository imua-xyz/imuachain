package feedermanagement

import (
	"testing"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	. "github.com/smartystreets/goconvey/convey"
	gomock "go.uber.org/mock/gomock"
)

func TestAggregation(t *testing.T) {
	Convey("aggregation", t, func() {
		Convey("add priceSouce in priceSource", func() {
			ps := newPriceSource(1, true)
			Convey("add first priceSource, success", func() {
				psAdded, err := ps.Add(ps1)
				So(psAdded, ShouldResemble, ps1)
				So(err, ShouldBeNil)
				_, ok := ps.detIDs["1"]
				So(ok, ShouldBeTrue)
				Convey("add different sourceID, reject", func() {
					psAdded, err := ps.Add(ps3)
					So(psAdded, ShouldBeNil)
					So(err, ShouldNotBeNil)
				})
				Convey("add same sourceID with same DetID, reject", func() {
					psAdded, err := ps.Add(ps1)
					So(psAdded, ShouldBeNil)
					So(err, ShouldNotBeNil)
				})
				Convey("add same sourceID with different DetID, success", func() {
					psAdded, err := ps.Add(ps2)
					So(psAdded, ShouldResemble, ps2)
					So(err, ShouldBeNil)
					_, ok := ps.detIDs["2"]
					So(ok, ShouldBeTrue)
				})
				Convey("add same sourceID with different DetID, duplicated input, return the added one value", func() {
					psAdded, err := ps.Add(ps4)
					So(psAdded, ShouldResemble, ps2)
					So(err, ShouldBeNil)
				})
			})
		})
		Convey("add priceSource in priceValidator", func() {
			// Try
			pv := newPriceValidator("validator1", big1)
			Convey("add source1 with 2 detIDs, try:success", func() {
				// duplicated detID=1 in ps1_2 will be removed in returned 'added'
				updated, added, err := pv.TryAddPriceSources([]*priceSource{ps1_2, ps2})
				So(updated, ShouldResemble, map[int64]*priceSource{1: ps5})
				So(added, ShouldResemble, []*priceSource{ps1, ps2})
				So(err, ShouldBeNil)
				// 'try' will not actually update pv
				So(pv.priceSources, ShouldHaveLength, 0)
				Convey("apply changes, success", func() {
					pv.ApplyAddedPriceSources(updated)
					So(pv.priceSources, ShouldHaveLength, 1)
					So(pv.priceSources, ShouldResemble, map[int64]*priceSource{1: ps5})
					Convey("add source1 with detID 3, try:success", func() {
						updated, added, err := pv.TryAddPriceSources([]*priceSource{ps3_2})
						So(updated, ShouldResemble, map[int64]*priceSource{1: ps6})
						So(added, ShouldResemble, []*priceSource{ps3_2})
						So(err, ShouldBeNil)
						So(pv.priceSources[1].prices, ShouldHaveLength, 2)
						Convey("apply changes, success", func() {
							pv.ApplyAddedPriceSources(updated)
							So(pv.priceSources[1].prices, ShouldHaveLength, 3)
						})
					})
				})
			})
		})
		Convey("record msgs in recordsValidators", func() {
			rv := newRecordsValidators()
			// TODO: multiple sources(for V2)
			Convey("record valid msg, success", func() {
				msgAdded, err := rv.RecordMsg(msgItem1)
				So(msgAdded, ShouldResemble, msgItem1_2)
				So(err, ShouldBeNil)
				So(rv.records["validator1"], ShouldResemble, &priceValidator{validator: "validator1", power: big1, priceSources: map[int64]*priceSource{1: ps5}})
				So(rv.accumulatedPower, ShouldResemble, big1)
				Convey("record duplicated msg, reject", func() {
					msgAdded, err := rv.RecordMsg(msgItem1_3)
					So(msgAdded, ShouldBeNil)
					So(err, ShouldNotBeNil)
				})
				Convey("record msg from another validator, success", func() {
					msgAdded, err := rv.RecordMsg(msgItem2)
					So(msgAdded, ShouldResemble, msgItem2_2)
					So(err, ShouldBeNil)
					So(rv.records["validator2"], ShouldResemble, &priceValidator{validator: "validator2", power: big1, priceSources: map[int64]*priceSource{1: ps5}})
					So(rv.accumulatedPower, ShouldResemble, big2)
					Convey("calculate final price without confirmed ds price, fail", func() {
						finalPrice, err := rv.GetFinalPrice(defaultAggMedian)
						So(finalPrice, ShouldBeNil)
						So(err, ShouldBeFalse)
					})
					Convey("calculate final price with confirmed ds price, success", func() {
						Convey("update final price of ds, success", func() {
							So(rv.records["validator1"].priceSources[1].finalPrice, ShouldBeNil)
							rv.UpdateFinalPriceForDS(1, pr1)
							So(rv.records["validator1"].priceSources[1].finalPrice, ShouldResemble, pr1)
							So(rv.records["validator2"].priceSources[1].finalPrice, ShouldResemble, pr1)
							finalPrice, err := rv.GetFinalPrice(defaultAggMedian)
							So(finalPrice, ShouldResemble, pr1_2)
							So(err, ShouldBeTrue)
						})
					})
				})
			})
		})
		Convey("add msgs in recordsDS", func() {
			rds := newRecordsDS()

			Convey("add first msg with v1-power-1 for detID2, success", func() {
				rds.AddPrice(pw2)
				So(rds.accumulatedPowers, ShouldResemble, big1)
				So(rds.validators["validator1"], ShouldNotBeNil)
				So(rds.records, ShouldHaveLength, 1)
				So(rds.records[0], ShouldResemble, pw2)
				Convey("add second msg with v1-power- 1 for detID1", func() {
					rds.AddPrice(pw1)
					So(rds.accumulatedPowers, ShouldResemble, big1)
					So(rds.records, ShouldHaveLength, 2)
					So(rds.records[0], ShouldResemble, pw1)
					So(rds.records[1], ShouldResemble, pw2)
					Convey("add 3rd msg with v2-power-1 for detID2", func() {
						rds.AddPrice(pw3)
						So(rds.accumulatedPowers, ShouldResemble, big2)
						So(rds.validators["validator2"], ShouldNotBeNil)
						So(rds.records, ShouldHaveLength, 2)
						So(rds.records[0], ShouldResemble, pw1)
						So(rds.records[1], ShouldResemble, pw2_2)
						finalPrice, ok := rds.GetFinalPrice(th)
						So(finalPrice, ShouldBeNil)
						So(ok, ShouldBeFalse)
						Convey("add 4th msg with v3-power-1 for detID2", func() {
							rds.AddPrice(pw4)
							So(rds.accumulatedPowers, ShouldResemble, big3)
							So(rds.validators["validator3"], ShouldNotBeNil)
							So(rds.records, ShouldHaveLength, 2)
							So(rds.records[1], ShouldResemble, pw3_2)
							Convey("get finalPrice, success", func() {
								finalPrice, ok = rds.GetFinalPrice(th)
								So(finalPrice, ShouldResemble, &PriceResult{
									Price:     "999",
									Decimal:   8,
									DetID:     "2",
									Timestamp: timestamp,
								})
								So(ok, ShouldBeTrue)
								Convey("add 5th msg with v4-power-1 for detID2", func() {
									rds.AddPrice(pw5)
									So(rds.accumulatedPowers, ShouldResemble, big4)
									So(rds.validators["validator4"], ShouldNotBeNil)
									So(rds.records, ShouldHaveLength, 2)
									finalPrice, ok = rds.GetFinalPrice(th)
									So(finalPrice, ShouldResemble, &PriceResult{
										Price:     "999",
										Decimal:   8,
										DetID:     "2",
										Timestamp: timestamp,
									})
								})
							})
						})
					})
				})
			})
		})
		Convey("add msgs in recordsDSs", func() {
			rdss := newRecordsDSs(th)
			Convey("add 3 same detId=1 prices from v1,v2,v3", func() {
				rdss.AddPriceSource(ps1, big1, "validator1")
				rdss.AddPriceSource(ps1, big1, "validator2")
				finalPrice, ok := rdss.GetFinalPriceForSourceID(1)
				So(finalPrice, ShouldBeNil)
				So(ok, ShouldBeFalse)
				rdss.AddPriceSource(ps1, big1, "validator3")
				finalPrice, ok = rdss.GetFinalPriceForSourceID(1)
				So(finalPrice, ShouldNotBeNil)
				So(finalPrice, ShouldResemble, ps1.prices[0].PriceResult())
				So(ok, ShouldBeTrue)
			})
			Convey("add 3 same detId=1 prices and 2 same detID=2 prices from v1,v2,v3", func() {
				rdss.AddPriceSource(ps1, big1, "validator1")
				rdss.AddPriceSource(ps2, big1, "validator2")
				finalPrice, ok := rdss.GetFinalPriceForSourceID(1)
				So(finalPrice, ShouldBeNil)
				So(ok, ShouldBeFalse)
				rdss.AddPriceSource(ps1_3, big1, "validator3")
				finalPrice, ok = rdss.GetFinalPriceForSourceID(1)
				So(finalPrice, ShouldBeNil)
				So(ok, ShouldBeFalse)
				rdss.AddPriceSource(ps2, big1, "validator4")
				finalPrice, ok = rdss.GetFinalPriceForSourceID(1)
				So(finalPrice, ShouldResemble, ps2.prices[0].PriceResult())
				So(ok, ShouldBeTrue)
			})

		})
		Convey("add msgs in aggregator", func() {
			a := newAggregator(th, defaultAggMedian)
			err := a.AddMsg(msgItem1)
			So(err, ShouldBeNil)
			finalPrice, ok := a.GetFinalPrice()
			So(finalPrice, ShouldBeNil)
			So(ok, ShouldBeFalse)

			err = a.AddMsg(msgItem2)
			So(err, ShouldBeNil)
			finalPrice, ok = a.GetFinalPrice()
			So(finalPrice, ShouldBeNil)
			So(ok, ShouldBeFalse)

			// failed to add duplicated msg
			err = a.AddMsg(msgItem2)
			So(err, ShouldNotBeNil)

			// powe exceeds 2/3 on detID=2
			err = a.AddMsg(msgItem3)
			So(err, ShouldBeNil)
			finalPrice, ok = a.GetFinalPrice()
			So(finalPrice, ShouldResemble, &PriceResult{Price: "999", Decimal: 8})
			So(ok, ShouldBeTrue)
			So(a.ds.GetFinalDetIDForSourceID(1), ShouldEqual, "2")
		})
		Convey("tally in round", func() {
			ctrl := gomock.NewController(t)
			c := NewMockCacheReader(ctrl)
			c.EXPECT().
				GetPowerForValidator(gomock.Any()).
				Return(big1, true).
				AnyTimes()
			c.EXPECT().
				IsDeterministic(gomock.Eq(int64(1))).
				Return(true, nil).
				AnyTimes()
			c.EXPECT().
				GetThreshold().
				Return(th).
				AnyTimes()
			c.EXPECT().
				IsRuleV1(gomock.Any()).
				Return(true).
				AnyTimes()

			r := tData.NewRound(c)
			r.cache = c
			feederID := r.feederID
			Convey("add msg in closed quoting window", func() {
				pmsg1 := protoMsgItem1
				pmsg1.FeederID = uint64(feederID)
				finalPrice, addedMsgItem, err := r.Tally(pmsg1)
				// quoting window not open
				So(err, ShouldNotBeNil)
				So(finalPrice, ShouldBeNil)
				So(addedMsgItem, ShouldBeNil)
			})
			Convey("open quotingWindow", func() {
				r.PrepareForNextBlock(int64(params.TokenFeeders[r.feederID].StartBaseBlock))
				So(r.status, ShouldEqual, roundStatusOpen)
				Convey("add msg-v1-detID1 for source1", func() {
					pmsg1 := protoMsgItem1
					pmsg1.FeederID = uint64(feederID)
					finalPrice, addedMsgItem, err := r.Tally(pmsg1)
					So(finalPrice, ShouldBeNil)
					So(addedMsgItem, ShouldResemble, pmsg1)
					So(err, ShouldBeNil)
					Convey("add msg-v1-detID2, success ", func() {
						pmsg2 := protoMsgItem2
						pmsg2.FeederID = uint64(feederID)
						finalPrice, addedMsgItem, err = r.Tally(pmsg2)
						So(finalPrice, ShouldBeNil)
						So(addedMsgItem, ShouldResemble, pmsg2)
						So(err, ShouldBeNil)
						Convey("add msg-v2-detID2, success", func() {
							// v2,detID=2
							pmsg3 := protoMsgItem3
							pmsg3.FeederID = uint64(feederID)
							finalPrice, addedMsgItem, err = r.Tally(pmsg3)
							So(finalPrice, ShouldBeNil)
							So(addedMsgItem, ShouldResemble, pmsg3)
							So(err, ShouldBeNil)
							Convey("two cases:", func() {
								Convey("add msg-v3-detID2, finalPrice", func() {
									// v3,detID=2
									pmsg4 := protoMsgItem4
									pmsg4.FeederID = uint64(feederID)
									finalPrice, addedMsgItem, err = r.Tally(pmsg4)
									So(finalPrice, ShouldResemble, &PriceResult{
										Price:   "999",
										Decimal: 8,
										DetID:   "2",
									})
									So(addedMsgItem, ShouldResemble, pmsg4)
									So(err, ShouldBeNil)
									Convey("add msg-v4-detID2, recordOnly", func() {
										pmsg5 := protoMsgItem5
										pmsg5.FeederID = uint64(feederID)
										finalPrice, addedMsgItem, err = r.Tally(pmsg5)
										So(finalPrice, ShouldBeNil)
										So(addedMsgItem, ShouldResemble, pmsg5)
										So(err, ShouldBeError, oracletypes.ErrQuoteRecorded)
									})
								})
								Convey("add msg-v3-detID2-different-price, success", func() {
									pmsg4 := protoMsgItem4_2
									pmsg4.FeederID = uint64(feederID)
									finalPrice, addedMsgItem, err = r.Tally(pmsg4)
									So(finalPrice, ShouldBeNil)
									So(addedMsgItem, ShouldResemble, pmsg4)
									So(err, ShouldBeNil)
									Convey("add msg-v4-detID2, success", func() {
										pmsg5 := protoMsgItem5
										pmsg5.FeederID = uint64(feederID)
										finalPrice, addedMsgItem, err = r.Tally(pmsg5)
										So(finalPrice, ShouldResemble, &PriceResult{
											Price:   "999",
											Decimal: 8,
											DetID:   "2",
										})
										So(addedMsgItem, ShouldResemble, pmsg5)
										So(err, ShouldBeNil)
									})

								})
							})
						})
					})
				})
			})
		})
	})
}
