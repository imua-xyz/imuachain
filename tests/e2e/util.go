package e2e

import (
	"context"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/testutil/network"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/evmos/evmos/v16/crypto/hd"
)

// func (s *E2ETestSuite) queryNativeCoinBalance(address sdk.AccAddress, n *network.Network) (*banktypes.QueryBalanceResponse, error) {
func QueryNativeCoinBalance(address sdk.AccAddress, n *network.Network) (*banktypes.QueryBalanceResponse, error) {
	return n.QueryBank().Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: address.String(),
		// Denom:   s.network.Config.NativeDenom,
		Denom: n.Config.NativeDenom,
	})
}

// func (s *E2ETestSuite) newNativeCoin(amount sdkmath.Int, n *network.Network) sdk.Coin {
func NewNativeCoin(amount sdkmath.Int, n *network.Network) sdk.Coin {
	// return sdk.NewCoin(s.network.Config.NativeDenom, amount)
	return sdk.NewCoin(n.Config.NativeDenom, amount)
}

func GenerateAccAddress(kr keyring.Keyring, name string) (sdk.AccAddress, error) {
	// generate a new account with ethsecp256k1
	r, _, err := kr.NewMnemonic(name, keyring.English, sdk.GetConfig().GetFullBIP44Path(), "", hd.EthSecp256k1)
	if err != nil {
		return nil, err
	}
	addr, _ := r.GetAddress()
	return addr, nil
}
