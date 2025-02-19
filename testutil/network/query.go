package network

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
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
