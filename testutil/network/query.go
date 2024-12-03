package network

import (
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
)

func (n *Network) QueryOracle() oracletypes.QueryClient {
	return oracletypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QueryBank() banktypes.QueryClient {
	return banktypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QueryAssets() assetstypes.QueryClient {
	return assetstypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QueryOperator() operatortypes.QueryClient {
	return operatortypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QueryAVS() avstypes.QueryClient {
	return avstypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QueryDogfood() dogfoodtypes.QueryClient {
	return dogfoodtypes.NewQueryClient(n.Validators[0].ClientCtx)
}

func (n *Network) QuerySlashing() slashingtypes.QueryClient {
	return slashingtypes.NewQueryClient(n.Validators[0].ClientCtx)
}
