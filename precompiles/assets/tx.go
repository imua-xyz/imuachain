package assets

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/ethereum/go-ethereum/common/hexutil"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	MethodDepositLST                  = "depositLST"
	MethodDepositNST                  = "depositNST"
	MethodWithdrawLST                 = "withdrawLST"
	MethodWithdrawNST                 = "withdrawNST"
	MethodGetClientChains             = "getClientChains"
	MethodRegisterOrUpdateClientChain = "registerOrUpdateClientChain"
	MethodRegisterToken               = "registerToken"
	MethodUpdateToken                 = "updateToken"
	MethodIsRegisteredClientChain     = "isRegisteredClientChain"
)

// DepositOrWithdraw deposit and withdraw the client chain assets for the staker,
// that will change the state in assets module.
func (p Precompile) DepositOrWithdraw(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be exoCore LzApp contract
	err := p.assetsKeeper.CheckExocoreGatewayAddr(ctx, contract.CallerAddress)
	if err != nil {
		return nil, fmt.Errorf(exocmn.ErrContractCaller, err.Error())
	}
	// parse the depositTo input params
	depositWithdrawParams, err := p.DepositWithdrawParams(ctx, method, args)
	if err != nil {
		return nil, err
	}

	// call assets keeper to perform the deposit or withdraw action
	err = p.assetsKeeper.PerformDepositOrWithdraw(ctx, depositWithdrawParams)
	if err != nil {
		return nil, err
	}

	// call oracle to update the validator info of staker for native asset restaking
	if depositWithdrawParams.Action == assetstypes.DepositNST ||
		depositWithdrawParams.Action == assetstypes.WithdrawNST {
		opAmount := depositWithdrawParams.OpAmount
		if depositWithdrawParams.Action == assetstypes.WithdrawNST {
			opAmount = opAmount.Neg()
		}
		_, assetID := assetstypes.GetStakerIDAndAssetID(depositWithdrawParams.ClientChainLzID,
			depositWithdrawParams.StakerAddress, depositWithdrawParams.AssetsAddress)
		err = p.assetsKeeper.UpdateNSTValidatorListForStaker(ctx, assetID,
			hexutil.Encode(depositWithdrawParams.StakerAddress),
			hexutil.Encode(depositWithdrawParams.ValidatorPubkey),
			opAmount)
		if err != nil {
			return nil, err
		}
	}

	// get the latest asset state of staker to return.
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(depositWithdrawParams.ClientChainLzID, depositWithdrawParams.StakerAddress, depositWithdrawParams.AssetsAddress)
	info, err := p.assetsKeeper.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, info.TotalDepositAmount.BigInt())
}

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
		return method.Outputs.Pack(false, nil)
	}
	ids, err := p.assetsKeeper.GetAllClientChainID(ctx)
	if err != nil {
		ctx.Logger().Error(
			"GetClientChains",
			"err", err,
		)
		return method.Outputs.Pack(false, nil)
	}
	return method.Outputs.Pack(true, ids)
}

func (p Precompile) RegisterOrUpdateClientChain(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be exoCore LzApp contract
	err := p.assetsKeeper.CheckExocoreGatewayAddr(ctx, contract.CallerAddress)
	if err != nil {
		return nil, fmt.Errorf(exocmn.ErrContractCaller, err.Error())
	}

	clientChainInfo, err := p.ClientChainInfoFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}
	updated := p.assetsKeeper.ClientChainExists(ctx, clientChainInfo.LayerZeroChainID)
	err = p.assetsKeeper.SetClientChainInfo(ctx, clientChainInfo)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, updated)
}

func (p Precompile) RegisterToken(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// the caller must be the ExocoreGateway contract
	err := p.assetsKeeper.CheckExocoreGatewayAddr(ctx, contract.CallerAddress)
	if err != nil {
		return nil, fmt.Errorf(exocmn.ErrContractCaller, err.Error())
	}

	// parse inputs
	asset, oInfo, err := p.TokenFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(asset.LayerZeroChainID, "", asset.Address)
	oInfo.AssetID = assetID

	if p.assetsKeeper.IsStakingAsset(ctx, assetID) {
		return nil, fmt.Errorf("asset %s already exists", assetID)
	}

	stakingAsset := &assetstypes.StakingAssetInfo{
		AssetBasicInfo:     asset,
		StakingTotalAmount: sdkmath.NewInt(0),
	}

	if err := p.assetsKeeper.RegisterNewTokenAndSetTokenFeeder(ctx, &oInfo); err != nil {
		return nil, err
	}

	// this is where the magic happens
	if err := p.assetsKeeper.SetStakingAssetInfo(ctx, stakingAsset); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) UpdateToken(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// the caller must be the ExocoreGateway contract
	err := p.assetsKeeper.CheckExocoreGatewayAddr(ctx, contract.CallerAddress)
	if err != nil {
		return nil, fmt.Errorf(exocmn.ErrContractCaller, err.Error())
	}

	// parse inputs
	clientChainID, hexAssetAddr, metadata, err := p.UpdateTokenFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(uint64(clientChainID), "", hexAssetAddr)
	// this verifies the existence of the asset and returns an error if it doesn't exist
	if err := p.assetsKeeper.UpdateStakingAssetMetaInfo(ctx, assetID, metadata); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) IsRegisteredClientChain(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	clientChainID, err := p.ClientChainIDFromInputs(ctx, args)
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
