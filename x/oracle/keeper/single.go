package keeper

import (
	"math/big"
	"strconv"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/aggregator"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/cache"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var updatedFeederIDs []string

var cs *cache.Cache

var agc, agcCheckTx *aggregator.AggregatorContext

func GetCaches() *cache.Cache {
	if cs != nil {
		return cs
	}
	cs = cache.NewCache()
	return cs
}

// GetAggregatorContext returns singleton aggregatorContext used to calculate final price for each round of each tokenFeeder
func GetAggregatorContext(ctx sdk.Context, k Keeper) *aggregator.AggregatorContext {
	if ctx.IsCheckTx() {
		if agcCheckTx != nil {
			return agcCheckTx
		}
		if agc == nil {
			c := GetCaches()
			c.ResetCaches()
			agcCheckTx = aggregator.NewAggregatorContext()
			if ok := recacheAggregatorContext(ctx, agcCheckTx, k, c); !ok {
				// this is the very first time oracle has been started, fill relalted info as initialization
				initAggregatorContext(ctx, agcCheckTx, k, c)
			}
			return agcCheckTx
		}
		agcCheckTx = agc.Copy4CheckTx()
		return agcCheckTx
	}

	if agc != nil {
		return agc
	}

	c := GetCaches()
	c.ResetCaches()
	agc = aggregator.NewAggregatorContext()
	if ok := recacheAggregatorContext(ctx, agc, k, c); !ok {
		// this is the very first time oracle has been started, fill relalted info as initialization
		initAggregatorContext(ctx, agc, k, c)
	} else {
		// this is when a node restart and use the persistent state to refill cache, we don't need to commit these data again
		c.SkipCommit()
	}
	return agc
}

func recacheAggregatorContext(ctx sdk.Context, agc *aggregator.AggregatorContext, k Keeper, c *cache.Cache) bool {
	logger := k.Logger(ctx)
	from := ctx.BlockHeight() - int64(common.MaxNonce) + 1
	to := ctx.BlockHeight()

	h, ok := k.GetValidatorUpdateBlock(ctx)
	recentParamsMap := k.GetAllRecentParamsAsMap(ctx)
	if !ok || len(recentParamsMap) == 0 {
		logger.Info("no validatorUpdateBlock found, go to initial process", "height", ctx.BlockHeight())
		// no cache, this is the very first running, so go to initial process instead
		return false
	}
	// #nosec G115
	if int64(h.Block) >= from {
		from = int64(h.Block) + 1
	}

	logger.Info("recacheAggregatorContext", "from", from, "to", to, "height", ctx.BlockHeight())
	totalPower := big.NewInt(0)
	validatorPowers := make(map[string]*big.Int)
	validatorSet := k.GetAllExocoreValidators(ctx)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
		totalPower = new(big.Int).Add(totalPower, big.NewInt(v.Power))
	}
	agc.SetValidatorPowers(validatorPowers)

	// reset validators
	c.AddCache(cache.ItemV(validatorPowers))

	recentMsgs := k.GetAllRecentMsgAsMap(ctx)
	var p *types.Params
	var b int64
	if from >= to {
		// backwards compatible for that the validatorUpdateBlock updated every block
		prev := int64(0)
		for b = range recentParamsMap {
			if b > prev {
				prev = b
			}
		}
		p = recentParamsMap[prev]
		agc.SetParams(p)
		setCommonParams(p)
	} else {
		prev := int64(0)
		for ; from < to; from++ {
			// fill params
			for b, p = range recentParamsMap {
				// find the params which is the latest one before the replayed block height since prepareRoundEndBlock will use it and it should be the latest one before current block
				if b < from && b > prev {
					agc.SetParams(p)
					prev = b
					setCommonParams(p)
					delete(recentParamsMap, b)
				}
			}

			agc.PrepareRoundEndBlock(uint64(from - 1))

			if msgs := recentMsgs[from]; msgs != nil {
				for _, msg := range msgs {
					// these messages are retreived for recache, just skip the validation check and fill the memory cache
					//nolint
					agc.FillPrice(&types.MsgCreatePrice{
						Creator:  msg.Validator,
						FeederID: msg.FeederID,
						Prices:   msg.PSources,
					})
				}
			}
			ctxReplay := ctx.WithBlockHeight(from)
			agc.SealRound(ctxReplay, false)
		}

		for b, p = range recentParamsMap {
			// use the latest params before the current block height
			if b < to && b > prev {
				agc.SetParams(p)
				prev = b
				setCommonParams(p)
			}
		}

		agc.PrepareRoundEndBlock(uint64(to - 1))
	}

	var pRet cache.ItemP
	if updated := c.GetCache(&pRet); !updated {
		c.AddCache(cache.ItemP(*p))
	}
	// TODO: these 4 lines are mainly used for hot fix
	// since the latest params stored in KV for recache should be the same with the latest params, so these lines are just duplicated actions if everything is fine.
	*p = k.GetParams(ctx)
	agc.SetParams(p)
	setCommonParams(p)
	c.AddCache(cache.ItemP(*p))

	return true
}

func initAggregatorContext(ctx sdk.Context, agc *aggregator.AggregatorContext, k common.KeeperOracle, c *cache.Cache) {
	ctx.Logger().Info("initAggregatorContext", "height", ctx.BlockHeight())
	// set params
	p := k.GetParams(ctx)
	agc.SetParams(&p)
	// set params cache
	c.AddCache(cache.ItemP(p))
	setCommonParams(&p)

	totalPower := big.NewInt(0)
	validatorPowers := make(map[string]*big.Int)
	validatorSet := k.GetAllExocoreValidators(ctx)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
		totalPower = new(big.Int).Add(totalPower, big.NewInt(v.Power))
	}

	agc.SetValidatorPowers(validatorPowers)
	// set validatorPower cache
	c.AddCache(cache.ItemV(validatorPowers))

	agc.PrepareRoundEndBlock(uint64(ctx.BlockHeight()) - 1)
}

func ResetAggregatorContext() {
	agc = nil
}

func ResetCache() {
	cs = nil
}

func ResetAggregatorContextCheckTx() {
	agcCheckTx = nil
}

func setCommonParams(p *types.Params) {
	common.MaxNonce = p.MaxNonce
	common.ThresholdA = p.ThresholdA
	common.ThresholdB = p.ThresholdB
	common.MaxDetID = p.MaxDetId
	common.Mode = p.Mode
}

func ResetUpdatedFeederIDs() {
	if updatedFeederIDs != nil {
		updatedFeederIDs = nil
	}
}

func GetUpdatedFeederIDs() []string {
	return updatedFeederIDs
}

func AppendUpdatedFeederIDs(id uint64) {
	updatedFeederIDs = append(updatedFeederIDs, strconv.FormatUint(id, 10))
}
