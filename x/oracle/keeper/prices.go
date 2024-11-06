package keeper

import (
	"encoding/binary"

	sdkmath "cosmossdk.io/math"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetPrices set a specific prices in the store from its index
func (k Keeper) SetPrices(ctx sdk.Context, prices types.Prices) {
	store := k.getPriceTRStore(ctx, prices.TokenID)
	for _, v := range prices.PriceList {
		b := k.cdc.MustMarshal(v)
		store.Set(types.PricesRoundKey(v.RoundID), b)
	}
	store.Set(types.PricesNextRoundIDKey, types.Uint64Bytes(prices.NextRoundID))
}

// GetPrices returns a prices from its index
func (k Keeper) GetPrices(
	ctx sdk.Context,
	tokenID uint64,
) (val types.Prices, found bool) {
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundID := k.GetNextRoundID(ctx, tokenID)

	val.TokenID = tokenID
	val.NextRoundID = nextRoundID
	var i uint64
	if nextRoundID <= uint64(common.MaxSizePrices) {
		i = 1
		val.PriceList = make([]*types.PriceTimeRound, 0, nextRoundID)
	} else {
		i = nextRoundID - uint64(common.MaxSizePrices)
		val.PriceList = make([]*types.PriceTimeRound, 0, common.MaxSizePrices)
	}
	for ; i < nextRoundID; i++ {
		b := store.Get(types.PricesRoundKey(i))
		v := &types.PriceTimeRound{}
		if b != nil {
			k.cdc.MustUnmarshal(b, v)
			found = true
		}
		val.PriceList = append(val.PriceList, v)
	}
	return
}

// return latest price for one specified price
func (k Keeper) GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (types.Price, error) {
	// for native token exo, we temporarily use default price
	if assetID == assetstypes.ExocoreAssetID {
		return types.Price{
			Value:   sdkmath.NewInt(types.DefaultPriceValue),
			Decimal: types.DefaultPriceDecimal,
		}, nil
	}

	var p types.Params
	// get params from cache if exists
	if agc != nil {
		p = agc.GetParams()
	} else {
		p = k.GetParams(ctx)
	}
	tokenID := p.GetTokenIDFromAssetID(assetID)
	if tokenID == 0 {
		return types.Price{}, types.ErrGetPriceAssetNotFound.Wrapf("assetID does not exist in oracle %s", assetID)
	}
	price, found := k.GetPriceTRLatest(ctx, uint64(tokenID))
	if !found {
		return types.Price{
			Value:   sdkmath.NewInt(types.DefaultPriceValue),
			Decimal: types.DefaultPriceDecimal,
		}, types.ErrGetPriceRoundNotFound.Wrapf("no valid price for assetID=%s", assetID)
	}
	v, _ := sdkmath.NewIntFromString(price.Price)
	// for tokens really have 0 price, it should be removed from assets support, not here to provide zero price
	if v.IsNil() || v.LTE(sdkmath.ZeroInt()) {
		return types.Price{
			Value:   sdkmath.NewInt(types.DefaultPriceValue),
			Decimal: types.DefaultPriceDecimal,
		}, types.ErrGetPriceRoundNotFound.Wrapf("no valid price for assetID=%s", assetID)
	}
	return types.Price{
		Value:   v,
		Decimal: uint8(price.Decimal), // #nosec G115
	}, nil
}

// return latest price for assets
func (k Keeper) GetMultipleAssetsPrices(ctx sdk.Context, assets map[string]interface{}) (prices map[string]types.Price, err error) {
	var p types.Params
	// get params from cache if exists
	if agc != nil {
		p = agc.GetParams()
	} else {
		p = k.GetParams(ctx)
	}
	// ret := make(map[string]types.Price)
	prices = make(map[string]types.Price)
	info := ""
	for assetID := range assets {
		// for native token exo, we temporarily use default price
		if assetID == assetstypes.ExocoreAssetID {
			prices[assetID] = types.Price{
				Value:   sdkmath.NewInt(types.DefaultPriceValue),
				Decimal: types.DefaultPriceDecimal,
			}
			continue
		}
		tokenID := p.GetTokenIDFromAssetID(assetID)
		if tokenID == 0 {
			err = types.ErrGetPriceAssetNotFound.Wrapf("assetID does not exist in oracle %s", assetID)
			prices = nil
			break
		}
		price, found := k.GetPriceTRLatest(ctx, uint64(tokenID))
		if !found {
			info = info + assetID + " "
			prices[assetID] = types.Price{
				Value:   sdkmath.NewInt(types.DefaultPriceValue),
				Decimal: types.DefaultPriceDecimal,
			}
		} else {
			v, _ := sdkmath.NewIntFromString(price.Price)
			// for tokens really have 0 price, it should be removed from assets support, not here to provide zero price
			if v.IsNil() || v.LTE(sdkmath.ZeroInt()) {
				info = info + assetID + " "
				prices[assetID] = types.Price{
					Value:   sdkmath.NewInt(types.DefaultPriceValue),
					Decimal: types.DefaultPriceDecimal,
				}
				continue
			}
			prices[assetID] = types.Price{
				Value:   v,
				Decimal: uint8(price.Decimal), // #nosec G115
			}
		}
	}
	if err == nil && len(info) > 0 {
		err = types.ErrGetPriceRoundNotFound.Wrapf("no valid price for assetIDs=%s", info)
	}
	return prices, err
}

// RemovePrices removes a prices from the store
func (k Keeper) RemovePrices(
	ctx sdk.Context,
	tokenID uint64,
) {
	store := k.getPriceTRStore(ctx, tokenID)
	//	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}

// GetAllPrices returns all prices
func (k Keeper) GetAllPrices(ctx sdk.Context) (list []types.Prices) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PricesKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	var price types.Prices
	prevTokenID := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		tokenID, _, nextRoundID := parseKey(iterator.Key())
		if prevTokenID == 0 {
			prevTokenID = tokenID
			price.TokenID = tokenID
		} else if prevTokenID != tokenID && price.TokenID > 0 {
			list = append(list, price)
			prevTokenID = tokenID
			price = types.Prices{TokenID: tokenID}
		}
		if nextRoundID {
			price.NextRoundID = binary.BigEndian.Uint64(iterator.Value())
		} else {
			var val types.PriceTimeRound
			k.cdc.MustUnmarshal(iterator.Value(), &val)
			price.PriceList = append(price.PriceList, &val)
		}
	}
	if price.TokenID > 0 {
		list = append(list, price)
	}
	return list
}

// AppenPriceTR append a new round of price for specific token, return false if the roundID not match
func (k Keeper) AppendPriceTR(ctx sdk.Context, tokenID uint64, priceTR types.PriceTimeRound) bool {
	nextRoundID := k.GetNextRoundID(ctx, tokenID)
	// This should not happen
	if nextRoundID != priceTR.RoundID {
		return false
	}
	store := k.getPriceTRStore(ctx, tokenID)
	b := k.cdc.MustMarshal(&priceTR)
	store.Set(types.PricesRoundKey(nextRoundID), b)
	if expiredRoundID := nextRoundID - agc.GetParamsMaxSizePrices(); expiredRoundID > 0 {
		store.Delete(types.PricesRoundKey(expiredRoundID))
	}
	roundID := k.IncreaseNextRoundID(ctx, tokenID)

	// update for native tokens
	// TODO: set hooks as a genral approach
	var p types.Params
	// get params from cache if exists
	if agc != nil {
		p = agc.GetParams()
	} else {
		p = k.GetParams(ctx)
	}
	assetIDs := p.GetAssetIDsFromTokenID(tokenID)
	for _, assetID := range assetIDs {
		if assetstypes.IsNST(assetID) {
			if err := k.UpdateNSTByBalanceChange(ctx, assetID, []byte(priceTR.Price), roundID); err != nil {
				// we just report this error in log to notify validators
				k.Logger(ctx).Error(types.ErrUpdateNativeTokenVirtualPriceFail.Error(), "error", err)
			}
		}
	}

	return true
}

// GrowRoundID Increases roundID with the previous price
func (k Keeper) GrowRoundID(ctx sdk.Context, tokenID uint64) (price string, roundID uint64) {
	if pTR, ok := k.GetPriceTRLatest(ctx, tokenID); ok {
		pTR.RoundID++
		k.AppendPriceTR(ctx, tokenID, pTR)
		price = pTR.Price
		roundID = pTR.RoundID
	} else {
		nextRoundID := k.GetNextRoundID(ctx, tokenID)
		k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
			RoundID: nextRoundID,
		})
		price = ""
		roundID = nextRoundID
	}
	return
}

// GetPriceTRoundID gets the price of the specific roundID of a specific token, return format as PriceTimeRound
func (k Keeper) GetPriceTRRoundID(ctx sdk.Context, tokenID uint64, roundID uint64) (price types.PriceTimeRound, found bool) {
	store := k.getPriceTRStore(ctx, tokenID)

	b := store.Get(types.PricesRoundKey(roundID))
	if b == nil {
		return
	}

	k.cdc.MustUnmarshal(b, &price)
	found = true
	return
}

// GetPriceTRLatest gets the latest price of the specific tokenID, return format as PriceTimeRound
func (k Keeper) GetPriceTRLatest(ctx sdk.Context, tokenID uint64) (price types.PriceTimeRound, found bool) {
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundIDB := store.Get(types.PricesNextRoundIDKey)
	if nextRoundIDB == nil {
		return
	}
	nextRoundID := binary.BigEndian.Uint64(nextRoundIDB)
	// this token has no valid round yet
	if nextRoundID <= 1 {
		return
	}
	b := store.Get(types.PricesRoundKey(nextRoundID - 1))
	if b != nil {
		// should always be true
		k.cdc.MustUnmarshal(b, &price)
		found = true
	}
	return
}

// GetNextRoundID gets the next round id of a token
func (k Keeper) GetNextRoundID(ctx sdk.Context, tokenID uint64) (nextRoundID uint64) {
	nextRoundID = 1
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundIDB := store.Get(types.PricesNextRoundIDKey)
	if nextRoundIDB != nil {
		if nextRoundID = binary.BigEndian.Uint64(nextRoundIDB); nextRoundID == 0 {
			nextRoundID = 1
		}
	}
	return
}

// IncreaseNextRoundID increases nextRoundID persisted by 1 of a token
func (k Keeper) IncreaseNextRoundID(ctx sdk.Context, tokenID uint64) uint64 {
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundID := k.GetNextRoundID(ctx, tokenID)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, nextRoundID+1)
	store.Set(types.PricesNextRoundIDKey, b)
	return nextRoundID
}

func (k Keeper) getPriceTRStore(ctx sdk.Context, tokenID uint64) prefix.Store {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PricesKeyPrefix))
	return prefix.NewStore(store, types.PricesKey(tokenID))
}

func parseKey(key []byte) (tokenID uint64, roundID uint64, nextRoundID bool) {
	tokenID = binary.BigEndian.Uint64(key[:8])
	if len(key) == 21 {
		nextRoundID = true
		return
	}
	roundID = binary.BigEndian.Uint64(key[9:17])
	return
}
