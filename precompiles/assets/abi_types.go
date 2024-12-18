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

type StakerBalance struct {
	ClientChainID      uint32
	StakerAddress      []byte
	TokenID            []byte
	Balance            *big.Int
	Withdrawable       *big.Int
	Delegated          *big.Int
	PendingUndelegated *big.Int
	TotalDeposited     *big.Int
}

func NewEmptyStakerBalance() StakerBalance {
	return StakerBalance{
		ClientChainID:      0,
		StakerAddress:      []byte{},
		TokenID:            []byte{},
		Balance:            big.NewInt(0),
		Withdrawable:       big.NewInt(0),
		Delegated:          big.NewInt(0),
		PendingUndelegated: big.NewInt(0),
		TotalDeposited:     big.NewInt(0),
	}
}
