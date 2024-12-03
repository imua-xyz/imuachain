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

func (k *Keeper) GetCaches() *cache.Cache {
	if k.memStore.cs != nil {
		return k.memStore.cs
	}
	k.memStore.cs = cache.NewCache()
	return k.memStore.cs
}

// GetAggregatorContext returns singleton aggregatorContext used to calculate final price for each round of each tokenFeeder
func (k *Keeper) GetAggregatorContext(ctx sdk.Context) *aggregator.AggregatorContext {
	if ctx.IsCheckTx() {
		if k.memStore.agcCheckTx != nil {
			return k.memStore.agcCheckTx
		}
		if k.memStore.agc == nil {
			c := k.GetCaches()
			c.ResetCaches()
			k.memStore.agcCheckTx = aggregator.NewAggregatorContext()
			if ok := k.recacheAggregatorContext(ctx, k.memStore.agcCheckTx, c); !ok {
				// this is the very first time oracle has been started, fill relalted info as initialization
				initAggregatorContext(ctx, k.memStore.agcCheckTx, k, c)
			}
			return k.memStore.agcCheckTx
		}
		k.memStore.agcCheckTx = k.memStore.agc.Copy4CheckTx()
		return k.memStore.agcCheckTx
	}

	if k.memStore.agc != nil {
		return k.memStore.agc
	}

	c := k.GetCaches()
	c.ResetCaches()
	k.memStore.agc = aggregator.NewAggregatorContext()
	if ok := k.recacheAggregatorContext(ctx, k.memStore.agc, c); !ok {
		// this is the very first time oracle has been started, fill relalted info as initialization
		initAggregatorContext(ctx, k.memStore.agc, k, c)
	} else {
		// this is when a node restart and use the persistent state to refill cache, we don't need to commit these data again
		c.SkipCommit()
	}
	return k.memStore.agc
}

func (k Keeper) recacheAggregatorContext(ctx sdk.Context, agc *aggregator.AggregatorContext, c *cache.Cache) bool {
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

func initAggregatorContext(ctx sdk.Context, agc *aggregator.AggregatorContext, k *Keeper, c *cache.Cache) {
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

func (k *Keeper) ResetAggregatorContext() {
	k.memStore.agc = nil
}

func (k *Keeper) ResetCache() {
	k.memStore.cs = nil
}

func (k *Keeper) ResetAggregatorContextCheckTx() {
	k.memStore.agcCheckTx = nil
}

// setCommonParams save static fields in params in memory cache since these fields will not change during node running
// TODO: further when params is abled to be updated through tx/gov, this cache should be taken care if any is available to be changed
func setCommonParams(p *types.Params) {
	common.MaxNonce = p.MaxNonce
	common.ThresholdA = p.ThresholdA
	common.ThresholdB = p.ThresholdB
	common.MaxDetID = p.MaxDetId
	common.Mode = p.Mode
	common.MaxSizePrices = int(p.MaxSizePrices)
}

func (k *Keeper) ResetUpdatedFeederIDs() {
	if k.memStore.updatedFeederIDs != nil {
		k.memStore.updatedFeederIDs = nil
	}
}

func (k Keeper) GetUpdatedFeederIDs() []string {
	return k.memStore.updatedFeederIDs
}

func (k *Keeper) AppendUpdatedFeederIDs(id uint64) {
	k.memStore.updatedFeederIDs = append(k.memStore.updatedFeederIDs, strconv.FormatUint(id, 10))
}
