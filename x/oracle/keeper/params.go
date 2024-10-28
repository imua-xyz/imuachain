package keeper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/cache"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	startAfterBlocks = 10
	defaultInterval  = 30
)

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	// TODO: validation check
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)
}

func (k Keeper) RegisterNewTokenAndSetTokenFeeder(ctx sdk.Context, oInfo *types.OracleInfo) error {
	p := k.GetParams(ctx)
	if p.GetTokenIDFromAssetID(oInfo.AssetID) > 0 {
		return fmt.Errorf("assetID exists:%s", oInfo.AssetID)
	}
	chainID := uint64(0)
	for id, c := range p.Chains {
		if c.Name == oInfo.Chain.Name {
			chainID = uint64(id)
			break
		}
	}
	if chainID == 0 {
		// add new chain
		p.Chains = append(p.Chains, &types.Chain{
			Name: oInfo.Chain.Name,
			Desc: oInfo.Chain.Desc,
		})
		chainID = uint64(len(p.Chains) - 1)
	}
	decimalInt, err := strconv.ParseInt(oInfo.Token.Decimal, 10, 32)
	if err != nil {
		return err
	}
	intervalInt := uint64(0)
	if len(oInfo.Feeder.Interval) > 0 {
		intervalInt, err = strconv.ParseUint(oInfo.Feeder.Interval, 10, 64)
		if err != nil {
			return err
		}
	}
	if intervalInt == 0 {
		intervalInt = defaultInterval
	}

	for _, t := range p.Tokens {
		// token exists, bind assetID for this token
		// it's possible for  one price bonded with multiple assetID, like ETHUSDT from sepolia/mainnet
		if t.Name == oInfo.Token.Name && t.ChainID == chainID {
			t.AssetID = strings.Join([]string{t.AssetID, oInfo.AssetID}, ",")
			k.SetParams(ctx, p)
			if !ctx.IsCheckTx() {
				_ = GetAggregatorContext(ctx, k)
				cs.AddCache(cache.ItemP(p))
			}
			// there should have been existing tokenFeeder running(currently we register tokens from assets-module and with infinite endBlock)
			return nil
		}
	}

	// add a new token
	p.Tokens = append(p.Tokens, &types.Token{
		Name:            oInfo.Token.Name,
		ChainID:         chainID,
		ContractAddress: oInfo.Token.Contract,
		Decimal:         int32(decimalInt), // #nosec G115
		Active:          true,
		AssetID:         oInfo.AssetID,
	})

	// set a tokenFeeder for the new token
	p.TokenFeeders = append(p.TokenFeeders, &types.TokenFeeder{
		TokenID: uint64(len(p.Tokens) - 1),
		// we only support rule_1 for v1
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: uint64(ctx.BlockHeight() + startAfterBlocks),
		Interval:       intervalInt,
		// we don't end feeders for v1
		EndBlock: 0,
	})

	k.SetParams(ctx, p)
	// skip cache update if this is not deliverTx
	// for normal cosmostx, checkTx will skip actual message exucution and do anteHandler only, but from ethc.callContract the message will be executed without anteHandler check as checkTx mode.
	if !ctx.IsCheckTx() {
		_ = GetAggregatorContext(ctx, k)
		cs.AddCache(cache.ItemP(p))
	}
	return nil
}
