package utils

import sdkmath "cosmossdk.io/math"

// CalculateUSDValue assetUSDValue = (assetAmount*price)/(10^(asset.decimal+priceDecimal))
func CalculateUSDValue(assetAmount sdkmath.Int, price sdkmath.Int, assetDecimal uint32, priceDecimal uint8) sdkmath.LegacyDec {
	assetValue := assetAmount.Mul(price)
	assetValueDec := sdkmath.LegacyNewDecFromBigInt(assetValue.BigInt())
	// #nosec G115
	divisor := sdkmath.NewIntWithDecimal(1, int(assetDecimal)+int(priceDecimal))
	return assetValueDec.QuoInt(divisor)
}

// CalculateRewardUSDValue rewardUSDValue = (rewardAmountDec*price)/(10^(stakingDecimal+priceDecimal-rewardDenominationExponent))
func CalculateRewardUSDValue(rewardAmountDec sdkmath.LegacyDec, denominationExponent, stakingDecimal uint32, price sdkmath.Int, priceDecimal uint8) sdkmath.LegacyDec {
	assetValueDec := rewardAmountDec.MulInt(price)
	// #nosec G115
	divisor := sdkmath.NewIntWithDecimal(1, int(stakingDecimal)+int(priceDecimal)-int(denominationExponent))
	return assetValueDec.QuoInt(divisor)
}
