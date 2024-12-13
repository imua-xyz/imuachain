package assets

import "math/big"

type TokenInfo struct {
	Name          string
	Symbol        string
	ClientChainID uint32
	TokenID       []byte
	Decimals      uint8
	TotalStaked   *big.Int
}

func NewEmptyTokenInfo() TokenInfo {
	return TokenInfo{
		Name:          "",
		Symbol:        "",
		ClientChainID: 0,
		TokenID:       []byte{},
		Decimals:      0,
		TotalStaked:   big.NewInt(0),
	}
}
