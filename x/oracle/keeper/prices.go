package keeper

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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
	accPrice, ok := k.GetAccumulatedPrice(ctx, tokenID)
	if ok {
		val.AccumulatedPrice = &accPrice
	} else {
		k.Logger(ctx).Error("GetPrices failed to get accumulated price", "tokenID", tokenID)
	}
	twap, ok := k.GetTWAP(ctx, tokenID)
	if ok {
		val.Twap = &twap
	} else {
		k.Logger(ctx).Error("GetPrices failed to get TWAP", "tokenID", tokenID)
	}
	var i uint64
	maxSizePrices := k.FeederManager.GetMaxSizePricesFromCache()
	// #nosec G115
	if nextRoundID <= uint64(maxSizePrices) {
		i = 0
		val.PriceList = make([]*types.PriceTimeRound, 0, nextRoundID)
	} else {
		// #nosec G11
		i = nextRoundID - uint64(maxSizePrices)
		val.PriceList = make([]*types.PriceTimeRound, 0, maxSizePrices)
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
	return val, found
}

// GetSpecifiedAssetsPrice returns the TWAP(time weighted average price) for a specific assetID
func (k Keeper) GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (types.Price, error) {
	v := sdkmath.NewInt(types.DefaultPriceValue)
	decimal := uint8(types.DefaultPriceDecimal)
	// get params from cache if exists
	p := k.GetParamsFromCache()
	tokenID := p.GetTokenIDFromAssetID(assetID)
	if tokenID == 0 {
		// we always return 0 value for assetID that does not exist in oracle to avoid panic
		return types.Price{
			Value:   v,
			Decimal: decimal,
		}, types.ErrGetPriceAssetNotFound.Wrapf("assetID does not exist in oracle %s", assetID)
	}
	// #nosec G115
	// price, found := k.GetPriceTRLatest(ctx, uint64(tokenID))
	price, found := k.GetTWAP(ctx, uint64(tokenID))
	if found {
		v, _ = sdkmath.NewIntFromString(price.Price)
		decimal = uint8(price.Decimal) // #nosec G115
	} else {
		// #nosec G115
		latestPrice, found := k.GetPriceTRLatest(ctx, uint64(tokenID))
		if found {
			v, _ = sdkmath.NewIntFromString(latestPrice.Price)
			decimal = uint8(latestPrice.Decimal) // #nosec G115
		}
	}
	// for tokens really have 0 price, it should be removed from assets support, not here to provide zero price
	if v.IsNil() || v.LTE(sdkmath.ZeroInt()) {
		v = sdkmath.NewInt(types.DefaultPriceValue)
		decimal = types.DefaultPriceDecimal
	}
	return types.Price{
		Value:   v,
		Decimal: decimal,
	}, nil
}

// return TWAP(time weighted average price)s for assets
func (k Keeper) GetMultipleAssetsPrices(ctx sdk.Context, assets map[string]struct{}) (prices map[string]types.Price, err error) {
	// get params from cache if exists
	p := k.GetParamsFromCache()
	prices = make(map[string]types.Price)
	info := ""
	for assetID := range assets {
		tokenID := p.GetTokenIDFromAssetID(assetID)
		if tokenID == 0 {
			info = info + assetID + " "
			prices[assetID] = types.Price{
				Value:   sdkmath.NewInt(types.DefaultPriceValue),
				Decimal: types.DefaultPriceDecimal,
			}
			continue
			// break
		}
		// #nosec G115
		price, found := k.GetTWAP(ctx, uint64(tokenID))
		var v sdkmath.Int
		var decimal uint8
		if found {
			v, _ = sdkmath.NewIntFromString(price.Price)
			decimal = uint8(price.Decimal) // #nosec G115
		} else {
			// #nosec G115
			latestPrice, found := k.GetPriceTRLatest(ctx, uint64(tokenID))
			if found {
				v, _ = sdkmath.NewIntFromString(latestPrice.Price)
				decimal = uint8(latestPrice.Decimal) // #nosec G115
			}
		}
		if v.IsNil() || v.LTE(sdkmath.ZeroInt()) {
			info = info + assetID + " "
			v = sdkmath.NewInt(types.DefaultPriceValue)
			decimal = types.DefaultPriceDecimal
		}
		prices[assetID] = types.Price{
			Value:   v,
			Decimal: decimal,
		}
	}
	if len(info) > 0 {
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
	//	var err error
	prevTokenID := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		tokenID, _, nextRoundID, err := parseKey(iterator.Key())
		if err != nil {
			k.Logger(ctx).Error("GetAllPrices failed to parse key", "key", iterator.Key(), "err", err)
			return []types.Prices{}
		}
		if prevTokenID == 0 {
			prevTokenID = tokenID
			price.TokenID = tokenID
		} else if prevTokenID != tokenID && price.TokenID > 0 {
			list = append(list, price)
			prevTokenID = tokenID
			price = types.Prices{TokenID: tokenID}
		}
		if nextRoundID {
			price.NextRoundID, err = types.BytesToUint64(iterator.Value())
			if err != nil {
				k.Logger(ctx).Error("GetAllPrices failed to parse nextRoundID", "nextRoundID", iterator.Value(), "err", err)
				return []types.Prices{}
			}
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

// AppendPriceTR appends a new round of price for a specific token and returns false if the roundID does not match.
// The price round is always stored and the next round ID advanced when the roundID matches, regardless of the
// value of accumulate. When accumulate is true, this also updates the accumulated price/TWAP state for the token;
// when accumulate is false, no TWAP/accumulated-price updates are performed.
func (k Keeper) AppendPriceTR(ctx sdk.Context, tokenID uint64, priceTR types.PriceTimeRound, accumulate bool) bool {
	nextRoundID := k.GetNextRoundID(ctx, tokenID)
	logger := k.Logger(ctx)
	// This should not happen
	if nextRoundID != priceTR.RoundID {
		logger.Error("roundID not match", "nextRoundID", nextRoundID, "priceTR.RoundID", priceTR.RoundID)
		return false
	}
	store := k.getPriceTRStore(ctx, tokenID)
	b := k.cdc.MustMarshal(&priceTR)
	store.Set(types.PricesRoundKey(nextRoundID), b)

	p := *k.GetParamsFromCache()
	// #nosec G115  // maxSizePrices is not negative
	if nextRoundID >= uint64(p.MaxSizePrices) {
		expiredRoundID := nextRoundID - uint64(p.MaxSizePrices)
		store.Delete(types.PricesRoundKey(expiredRoundID))
	}
	k.IncreaseNextRoundID(ctx, tokenID)

	// accumulate the price for TWAP calculation
	if accumulate {
		if err := k.accumulatePriceTR(ctx, tokenID, priceTR); err != nil {
			// this case should not happen, we just log an error here
			logger.Error("accumulatePriceTR failed", "tokenID", tokenID, "priceTR", priceTR, "error", err)
		}
	}
	return true
}

// GrowRoundID increases the roundID using the previously stored price.
// If accumulate is true, the carried-forward price is appended in a way that
// updates TWAP/accumulated-price state (via AppendPriceTR); if false, the
// round is advanced without updating TWAP/accumulated-price state.
func (k Keeper) GrowRoundID(ctx sdk.Context, tokenID, nextRoundID uint64, accumulate bool) (price string, roundID uint64) {
	storedNextRoundID := k.GetNextRoundID(ctx, tokenID)
	pTR, _ := k.GetPriceTRLatest(ctx, tokenID)
	price = pTR.Price
	if nextRoundID < storedNextRoundID {
		roundID = storedNextRoundID - 1
		// we don't append new price if storedNextRoundID is larger than expected
		return
	}
	if nextRoundID > storedNextRoundID {
		// if storedNextRoundID is too old, we just set it with input params
		store := k.getPriceTRStore(ctx, tokenID)
		b := types.Uint64Bytes(nextRoundID)
		store.Set(types.PricesNextRoundIDKey, b)
	}
	roundID = nextRoundID
	pTR.RoundID = nextRoundID
	k.AppendPriceTR(ctx, tokenID, pTR, accumulate)
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
	var nextRoundID uint64
	var err error
	if nextRoundIDB == nil {
		nextRoundID = 1
	} else {
		nextRoundID, err = types.BytesToUint64(nextRoundIDB)
		// This should not happen
		if err != nil {
			// This should not happen
			k.Logger(ctx).Error("GetPriceTRLatest failed to parse nextRoundID", "nextRoundIDB", nextRoundIDB, "err", err)
			return
		}
		if nextRoundID < 1 {
			nextRoundID = 1
		}
	}

	b := store.Get(types.PricesRoundKey(nextRoundID - 1))
	if b != nil {
		k.cdc.MustUnmarshal(b, &price)
		if len(price.Price) > 0 {
			// price=="0" is also a valid price means the price is zero
			found = true
		}
	}
	return
}

// GetNextRoundID gets the next round id of a token
// the minimum returned value is 1
func (k Keeper) GetNextRoundID(ctx sdk.Context, tokenID uint64) (nextRoundID uint64) {
	var err error
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundIDB := store.Get(types.PricesNextRoundIDKey)
	if nextRoundIDB != nil {
		nextRoundID, err = types.BytesToUint64(nextRoundIDB)
		if err != nil {
			// this should not happen, we simplely log it here
			k.Logger(ctx).Error("GetNextRoundID failed to parse nextRoundID", "error", err)
		}
	}
	return max(1, nextRoundID)
}

// IncreaseNextRoundID increases nextRoundID persisted by 1 of a token
func (k Keeper) IncreaseNextRoundID(ctx sdk.Context, tokenID uint64) uint64 {
	store := k.getPriceTRStore(ctx, tokenID)
	nextRoundID := k.GetNextRoundID(ctx, tokenID)
	b := types.Uint64Bytes(nextRoundID + 1)
	store.Set(types.PricesNextRoundIDKey, b)
	return nextRoundID
}

func (k Keeper) accumulatePriceTR(ctx sdk.Context, tokenID uint64, priceTR types.PriceTimeRound) error {
	if priceTR.RoundID <= 1 {
		// if the roundID is 0,1 we don't accumulate it and return no error
		return nil
	}

	prevRoundID := priceTR.RoundID - 1
	prevPrice, found := k.GetPriceTRRoundID(ctx, tokenID, prevRoundID)
	if !found {
		return types.ErrAccumulatePrice.Wrapf("roundID %d not found for tokenID %d", prevRoundID, tokenID)
	}
	priceUnChanged := prevPrice.Equal(priceTR)

	store := ctx.KVStore(k.storeKey)
	key := types.PricesAccumulatedKey(tokenID)
	bz := store.Get(key)

	if priceUnChanged {
		if bz == nil {
			store.Set(key, k.cdc.MustMarshal(&types.PriceAcc{
				StartRoundID: prevRoundID,
				LastRoundID:  prevRoundID - 1,
				Price:        "0",
			}))
		}
		return nil
	}
	var accPrice types.PriceAcc
	if bz == nil {
		accPrice.StartRoundID = prevRoundID
		accPrice.LastRoundID = prevRoundID - 1
		accPrice.Price = "0"
	} else {
		k.cdc.MustUnmarshal(bz, &accPrice)
	}
	// prevPrice has already been accumulated, just skip
	if prevRoundID <= accPrice.LastRoundID {
		return nil
	}
	accPrice, err := accPrice.AccumulatePriceTR(prevPrice)
	if err != nil {
		return types.ErrAccumulatePrice.Wrapf("failed to accumulate price, err: %v, tokenID: %d, prevPrice: %v, priceTR: %v, accPrice: %v", err, tokenID, prevPrice, priceTR, accPrice)
	}
	store.Set(key, k.cdc.MustMarshal(&accPrice))
	return nil
}

func (k Keeper) ResetAccumulatedPrice(ctx sdk.Context, tokenID uint64) {
	store := ctx.KVStore(k.storeKey)
	key := types.PricesAccumulatedKey(tokenID)
	latestPrice, found := k.GetPriceTRLatest(ctx, tokenID)
	if !found || latestPrice.RoundID == 0 {
		return
	}
	accPrice := types.PriceAcc{
		StartRoundID: latestPrice.RoundID,
		LastRoundID:  latestPrice.RoundID - 1,
		Price:        "0",
		Decimal:      latestPrice.Decimal,
	}
	store.Set(key, k.cdc.MustMarshal(&accPrice))
}

// GetTWAP retrieves the time-weighted average price for the given tokenID from the store.
// NOTE: Non-numeric token prices (e.g. special tokens) will have no TWAP entry; callers should filter these before querying.
func (k Keeper) GetTWAP(ctx sdk.Context, tokenID uint64) (types.PriceEpoch, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.PricesTWAPKey(tokenID)
	bz := store.Get(key)
	if bz == nil {
		return types.PriceEpoch{}, false
	}
	var twap types.PriceEpoch
	k.cdc.MustUnmarshal(bz, &twap)
	return twap, true
}

func (k Keeper) GetAccumulatedPrice(ctx sdk.Context, tokenID uint64) (types.PriceAcc, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.PricesAccumulatedKey(tokenID)
	bz := store.Get(key)
	if bz == nil {
		return types.PriceAcc{}, false
	}
	var accPrice types.PriceAcc
	k.cdc.MustUnmarshal(bz, &accPrice)
	return accPrice, true
}

// CalculateTWAP calculates the TWAP for a specific tokenID within an epoch
func (k Keeper) CalculateTWAP(ctx sdk.Context, tokenID uint64, epochNumber int64) (types.PriceEpoch, error) {
	latestPrice, found := k.GetPriceTRLatest(ctx, tokenID)
	if !found {
		return types.PriceEpoch{}, types.ErrAccumulatePrice.Wrapf("latest price not found for tokenID %d", tokenID)
	}

	var twap types.PriceEpoch
	twapLatestPrice := types.PriceEpoch{
		Epoch:   epochNumber,
		RoundId: latestPrice.RoundID,
		Price:   latestPrice.Price,
		Decimal: latestPrice.Decimal,
	}
	keyTWAP := types.PricesTWAPKey(tokenID)
	store := ctx.KVStore(k.storeKey)
	key := types.PricesAccumulatedKey(tokenID)
	bz := store.Get(key)
	if bz == nil {
		store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
		return twapLatestPrice, types.ErrTWAPFallback.Wrapf("CalculateTWAP failed due to no accumulatedPrice found, just set twap to the latest price, tokenID:%d", tokenID)
	}
	if latestPrice.RoundID <= 1 {
		store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
		return twapLatestPrice, nil
	}

	// var accPrice types.PriceTimeRound
	var accPrice types.PriceAcc
	k.cdc.MustUnmarshal(bz, &accPrice)
	bz = store.Get(keyTWAP)
	var rounds uint64
	if bz == nil {
		if epochNumber == 0 {
			store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
			return twapLatestPrice, nil
		}
		twap.Epoch = epochNumber
		twap.Price = "0"
		twap.RoundId = accPrice.StartRoundID
	} else {
		k.cdc.MustUnmarshal(bz, &twap)
	}
	if latestPrice.RoundID > accPrice.LastRoundID+1 {
		prevPrice, ok := k.GetPriceTRRoundID(ctx, tokenID, latestPrice.RoundID-1)
		if !ok {
			store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
			return twapLatestPrice, types.ErrTWAPFallback.Wrapf("CalculateTWAP failed to get previous price for latestPrice.RoundID-1(%d), tokenID:%d, set twap to the latest price", latestPrice.RoundID-1, tokenID)
		}
		// if the latest price roundID is larger than accumulated price roundID, we need to add the latest price to the accumulated price
		var err error
		accPrice, err = accPrice.AccumulatePriceTR(prevPrice)
		if err != nil {
			store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
			return twapLatestPrice, types.ErrTWAPFallback.Wrapf("CalculateTWAP failed to accumulate price, err:%v, tokenID:%d, set twap to the latest price", err, tokenID)
		}
	}
	accPriceInt, ok := new(big.Int).SetString(accPrice.Price, 10)
	if !ok {
		store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
		return twapLatestPrice, types.ErrTWAPFallback.Wrapf("CalculateTWAP failed to parse accumulated price, accPrice:%s, tokenID:%d, set twap to the latest price", accPrice.Price, tokenID)
	}
	rounds = latestPrice.RoundID - accPrice.StartRoundID
	if rounds == 0 {
		store.Set(keyTWAP, k.cdc.MustMarshal(&twapLatestPrice))
		return twapLatestPrice, types.ErrTWAPFallback.Wrapf("CalculateTWAP failed, rounds is 0, latestPrice.RoundID:%d, twap.RoundId:%d, tokenID:%d, set twap to the latest price", latestPrice.RoundID, twap.RoundId, tokenID)
	}
	accPriceInt.Div(accPriceInt, new(big.Int).SetUint64(rounds))
	twap.Price = accPriceInt.String()
	twap.Decimal = accPrice.Decimal
	twap.Epoch = epochNumber
	twap.RoundId = latestPrice.RoundID
	store.Set(keyTWAP, k.cdc.MustMarshal(&twap))
	return twap, nil
}

func (k Keeper) getPriceTRStore(ctx sdk.Context, tokenID uint64) prefix.Store {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PricesKeyPrefix))
	return prefix.NewStore(store, types.PricesKey(tokenID))
}

func parseKey(key []byte) (tokenID uint64, roundID uint64, nextRoundID bool, err error) {
	if len(key) < 17 {
		err = types.ErrGetPriceRoundNotFound.Wrapf("key length is too short")
		return
	}
	tokenID, _ = types.BytesToUint64(key[:8])
	if len(key) == 21 {
		nextRoundID = true
		return
	}
	roundID, _ = types.BytesToUint64(key[9:17])
	return
}
