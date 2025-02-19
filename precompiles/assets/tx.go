package assets

import (
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
	MethodRegisterOrUpdateClientChain = "registerOrUpdateClientChain"
	MethodRegisterToken               = "registerToken"
	MethodUpdateToken                 = "updateToken"
	MethodUpdateAuthorizedGateways    = "updateAuthorizedGateways"
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
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	// parse the depositTo input params
	depositWithdrawParams, err := p.DepositWithdrawParams(ctx, method, args)
	if err != nil {
		return nil, err
	}

	// call assets keeper to perform the deposit or withdraw action
	finalDepositAmount, err := p.assetsKeeper.PerformDepositOrWithdraw(ctx, depositWithdrawParams)
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
			hexutil.Encode(depositWithdrawParams.ValidatorID),
			opAmount)
		if err != nil {
			return nil, err
		}
	}

	// return the latest asset state of staker
	return method.Outputs.Pack(true, finalDepositAmount.BigInt())
}

func (p Precompile) RegisterOrUpdateClientChain(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be exoCore LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	clientChainInfo, err := p.ClientChainInfoFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}
	updated, err := p.assetsKeeper.SetClientChainInfo(ctx, clientChainInfo)
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
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
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
		AssetBasicInfo:     *asset,
		StakingTotalAmount: sdkmath.ZeroInt(),
	}

	if err := p.assetsKeeper.RegisterNewTokenAndSetTokenFeeder(ctx, oInfo); err != nil {
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
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
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

// UpdateAuthorizedGateways updates the authorized gateways for the assets module.
// For mainnet, if the authority of the assets module is the governance module, this method would not work.
// So it is mainly used for testing purposes.
func (p Precompile) UpdateAuthorizedGateways(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	ta := NewTypedArgs(args)
	gateways, err := ta.GetRequiredEVMAddressSlice(0)
	if err != nil {
		return nil, err
	}

	gatewaysStr := make([]string, len(gateways))
	for i, gateway := range gateways {
		gatewaysStr[i] = gateway.Hex()
	}

	_, err = p.assetsKeeper.UpdateParams(ctx, &assetstypes.MsgUpdateParams{
		Authority: contract.CallerAddress.String(),
		Params:    assetstypes.Params{Gateways: gatewaysStr},
	})
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
