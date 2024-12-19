package aggregator

import (
	"math/big"

	. "github.com/agiledragon/gomonkey/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	// . "github.com/smartystreets/goconvey/convey"
)

// func TestAggregatorContext(t *testing.T) {
// 	Convey("init aggregatorContext with default params", t, func() {
// 		agc := initAggregatorContext4Test()
// 		var ctx sdk.Context
// 		Convey("prepare round to gengerate round info of feeders for next block", func() {
// 			Convey("pepare within the window", func() {
// 				p := patchBlockHeight(12)
// 				agc.PrepareRoundEndBlock(ctx, 11, 0)
//
// 				Convey("for empty round list", func() {
// 					So(*agc.rounds[1], ShouldResemble, roundInfo{10, 2, 1})
// 				})
//
// 				Convey("update already exist round info", func() {
// 					p.Reset()
// 					time.Sleep(1 * time.Second)
// 					patchBlockHeight(10 + int64(common.MaxNonce) + 1)
//
// 					agc.PrepareRoundEndBlock(ctx, 10+int64(common.MaxNonce), 0)
// 					So(agc.rounds[1].status, ShouldEqual, 2)
// 				})
// 				p.Reset()
// 				time.Sleep(1 * time.Second)
// 			})
// 			Convey("pepare outside the window", func() {
// 				Convey("for empty round list", func() {
// 					p := patchBlockHeight(10 + int64(common.MaxNonce) + 1)
// 					agc.PrepareRoundEndBlock(ctx, 10+int64(common.MaxNonce), 0)
// 					So(agc.rounds[1].status, ShouldEqual, 2)
// 					p.Reset()
// 					time.Sleep(1 * time.Second)
// 				})
// 			})
// 		})
//
// 		Convey("seal existing round without any msg recieved", func() {
// 			p := patchBlockHeight(11)
// 			agc.PrepareRoundEndBlock(ctx, 10, 0)
// 			Convey("seal when exceed the window", func() {
// 				So(agc.rounds[1].status, ShouldEqual, 1)
// 				p.Reset()
// 				time.Sleep(1 * time.Second)
// 				patchBlockHeight(13)
// 				agc.SealRound(ctx, false)
// 				So(agc.rounds[1].status, ShouldEqual, 2)
// 			})
//
// 			Convey("force seal by required", func() {
// 				p.Reset()
// 				time.Sleep(1 * time.Second)
// 				patchBlockHeight(12)
// 				agc.SealRound(ctx, false)
// 				So(agc.rounds[1].status, ShouldEqual, 1)
// 				agc.SealRound(ctx, true)
// 				So(agc.rounds[1].status, ShouldEqual, 2)
// 			})
// 			p.Reset()
// 			time.Sleep(1 * time.Second)
// 		})
// 	})
// }

func initAggregatorContext4Test() *AggregatorContext {
	agc := NewAggregatorContext()

	validatorPowers := map[string]*big.Int{
		"v1": big.NewInt(1),
		"v2": big.NewInt(1),
		"v3": big.NewInt(1),
	}

	p := defaultParams

	agc.SetValidatorPowers(validatorPowers)
	agc.SetParams(&p)
	return agc
}

func patchBlockHeight(h int64) *Patches {
	return ApplyMethod(sdk.Context{}, "BlockHeight", func(sdk.Context) int64 {
		return h
	})
}
