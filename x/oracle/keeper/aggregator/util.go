package aggregator

import "math/big"

func copyBigInt(i *big.Int) *big.Int {
	if i == nil {
		return nil
	}
	return big.NewInt(0).Set(i)
}
