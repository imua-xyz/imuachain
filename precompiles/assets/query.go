package assets

import (
	"errors"
	"math"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	MethodGetClientChains         = "getClientChains"
	MethodIsRegisteredClientChain = "isRegisteredClientChain"
	MethodIsAuthorizedGateway     = "isAuthorizedGateway"
	MethodGetTokenInfo            = "getTokenInfo"
	MethodGetStakerBalanceByToken = "getStakerBalanceByToken"
)

func (p Precompile) GetClientChains(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) > 0 {
		ctx.Logger().Error(
			"GetClientChains",
			"err", errors.New("no input is required"),
		)
		return method.Outputs.Pack(false, []uint32{})
	}
	ids, err := p.assetsKeeper.GetAllClientChainID(ctx)
	if err != nil {
		ctx.Logger().Error(
			"GetClientChains",
			"err", err,
		)
		return method.Outputs.Pack(false, []uint32{})
	}
	return method.Outputs.Pack(true, ids)
}

func (p Precompile) IsRegisteredClientChain(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodIsRegisteredClientChain].Inputs)); err != nil {
		return nil, err
	}
	clientChainID, err := ta.GetUint32(0)
	if err != nil {
		return nil, err
	}
	if clientChainID == 0 {
		// explicitly return false for client chain ID 0 to prevent `setPeer` calls
		return method.Outputs.Pack(true, false)
	}
	exists := p.assetsKeeper.ClientChainExists(ctx, uint64(clientChainID))
	return method.Outputs.Pack(true, exists)
}

func (p Precompile) IsAuthorizedGateway(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodIsAuthorizedGateway].Inputs)); err != nil {
		return nil, err
	}
	gateway, err := ta.GetEVMAddress(0)
	if err != nil {
		return nil, err
	}
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, gateway)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, authorized)
}

func (p Precompile) GetTokenInfo(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodGetTokenInfo].Inputs)); err != nil {
		return nil, err
	}
	clientChainID, err := ta.GetUint32(0)
	if err != nil {
		return nil, err
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, err
	}

	assetAddress, err := ta.GetRequiredBytesPrefix(1, info.AddressLength)
	if err != nil {
		return nil, err
	}
	_, assetID := assetstype.GetStakerIDAndAssetID(uint64(clientChainID), nil, assetAddress)
	tokenInfo, err := p.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if tokenInfo.AssetBasicInfo.Decimals > math.MaxUint8 {
		return nil, errors.New("decimals exceed max uint8")
	}

	// Pack the values into the struct
	result := TokenInfo{
		Name:          tokenInfo.AssetBasicInfo.Name,
		Symbol:        tokenInfo.AssetBasicInfo.Symbol,
		ClientChainID: clientChainID,
		TokenID:       assetAddress,
		Decimals:      uint8(tokenInfo.AssetBasicInfo.Decimals),
		TotalStaked:   tokenInfo.StakingTotalAmount.BigInt(),
	}

	return method.Outputs.Pack(true, result)
}

func (p Precompile) GetStakerBalanceByToken(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodGetStakerBalanceByToken].Inputs)); err != nil {
		return nil, err
	}

	clientChainID, err := ta.GetUint32(0)
	if err != nil {
		return nil, err
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, err
	}

	stakerAddress, err := ta.GetRequiredBytesPrefix(1, info.AddressLength)
	if err != nil {
		return nil, err
	}

	assetAddress, err := ta.GetRequiredBytesPrefix(2, info.AddressLength)
	if err != nil {
		return nil, err
	}

	stakerID, assetID := assetstype.GetStakerIDAndAssetID(uint64(clientChainID), stakerAddress, assetAddress)

	balance, err := p.assetsKeeper.GetStakerBalanceByAsset(ctx, stakerID, assetID)
	if err != nil {
		return nil, err
	}

	result := StakerBalance{
		ClientChainID:      clientChainID,
		StakerAddress:      stakerAddress,
		TokenID:            assetAddress,
		Balance:            balance.Balance,
		Withdrawable:       balance.Withdrawable,
		Delegated:          balance.Delegated,
		PendingUndelegated: balance.PendingUndelegated,
		TotalDeposited:     balance.TotalDeposited,
	}

	return method.Outputs.Pack(true, result)
}
