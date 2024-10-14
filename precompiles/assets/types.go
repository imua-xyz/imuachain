package assets

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/ethereum/go-ethereum/common/hexutil"

	sdkmath "cosmossdk.io/math"
	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

// oracleInfo: '[tokenName],[chainName],[tokenDecimal](,[interval],[contract](,[ChainDesc:{...}],[TokenDesc:{...}]))'
var (
	tokenDescMatcher = regexp.MustCompile(`TokenDesc:{(.+?)}`)
	chainDescMatcher = regexp.MustCompile(`ChainDesc:{(.+?)}`)
)

func (p Precompile) DepositWithdrawParams(ctx sdk.Context, method *abi.Method, args []interface{}) (*assetskeeper.DepositWithdrawParams, error) {
	inputsLen := len(p.ABI.Methods[method.Name].Inputs)
	if len(args) != inputsLen {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	depositWithdrawParams := &assetskeeper.DepositWithdrawParams{}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}
	depositWithdrawParams.ClientChainLzID = uint64(clientChainID)

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, depositWithdrawParams.ClientChainLzID)
	if err != nil {
		return nil, err
	}
	clientChainAddrLength := info.AddressLength

	switch method.Name {
	// deposit and withdraw transactions
	case MethodDepositLST, MethodWithdrawLST:
		// the length of client chain address inputted by caller is 32, so we need to check the length and remove the padding according to the actual length.
		assetAddr, ok := args[1].([]byte)
		if !ok || len(assetAddr) == 0 {
			return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
		}
		// #nosec G115
		if uint32(len(assetAddr)) < clientChainAddrLength {
			return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(assetAddr), clientChainAddrLength)
		}
		depositWithdrawParams.AssetsAddress = assetAddr[:clientChainAddrLength]
		if method.Name == MethodDepositLST {
			depositWithdrawParams.Action = assetstypes.DepositLST
		} else {
			depositWithdrawParams.Action = assetstypes.WithdrawLST
		}
	case MethodDepositNST, MethodWithdrawNST:
		validatorPubkey, ok := args[1].([]byte)
		if !ok || len(validatorPubkey) == 0 {
			return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
		}
		// generate the virtual address for native restaking asset
		depositWithdrawParams.AssetsAddress = assetstypes.GenerateNSTAddr(clientChainAddrLength)
		// todo: add a check for the validator pubkey
		depositWithdrawParams.ValidatorPubkey = validatorPubkey
		if method.Name == MethodDepositNST {
			depositWithdrawParams.Action = assetstypes.DepositNST
		} else {
			depositWithdrawParams.Action = assetstypes.WithdrawNST
		}
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	stakerAddr, ok := args[2].([]byte)
	if !ok || len(stakerAddr) == 0 {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 2, "[]byte", args[2])
	}
	// #nosec G115
	if uint32(len(stakerAddr)) < clientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(stakerAddr), clientChainAddrLength)
	}
	depositWithdrawParams.StakerAddress = stakerAddr[:clientChainAddrLength]

	opAmount, ok := args[3].(*big.Int)
	if !ok || opAmount == nil || !(opAmount.Cmp(big.NewInt(0)) == 1) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 3, "*big.Int", args[3])
	}
	depositWithdrawParams.OpAmount = sdkmath.NewIntFromBigInt(opAmount)

	return depositWithdrawParams, nil
}

func (p Precompile) ClientChainInfoFromInputs(_ sdk.Context, args []interface{}) (*assetstypes.ClientChainInfo, error) {
	inputsLen := len(p.ABI.Methods[MethodRegisterOrUpdateClientChain].Inputs)
	if len(args) != inputsLen {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	clientChain := assetstypes.ClientChainInfo{}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}
	clientChain.LayerZeroChainID = uint64(clientChainID)

	addressLength, ok := args[1].(uint8)
	if !ok || addressLength == 0 {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "uint8", args[1])
	}
	if addressLength < assetstypes.MinClientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, addressLength, assetstypes.MinClientChainAddrLength)
	}
	// #nosec G115
	clientChain.AddressLength = uint32(addressLength)

	name, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 2, "string", args[2])
	}
	if name == "" || len(name) > assetstypes.MaxChainTokenNameLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidNameLength, name, len(name), assetstypes.MaxChainTokenNameLength)
	}
	clientChain.Name = name

	metaInfo, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 3, "string", args[2])
	}
	if metaInfo == "" || len(metaInfo) > assetstypes.MaxChainTokenMetaInfoLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidMetaInfoLength, metaInfo, len(metaInfo), assetstypes.MaxChainTokenMetaInfoLength)
	}
	clientChain.MetaInfo = metaInfo

	signatureType, ok := args[4].(string)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 4, "string", args[4])
	}
	clientChain.SignatureType = signatureType

	return &clientChain, nil
}

func (p Precompile) TokenFromInputs(ctx sdk.Context, args []interface{}) (assetstypes.AssetInfo, oracletypes.OracleInfo, error) {
	inputsLen := len(p.ABI.Methods[MethodRegisterToken].Inputs)
	if len(args) != inputsLen {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	asset := assetstypes.AssetInfo{}
	oracleInfo := oracletypes.OracleInfo{}

	clientChainID, ok := args[0].(uint32)
	if !ok {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}
	asset.LayerZeroChainID = uint64(clientChainID)
	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, asset.LayerZeroChainID)
	if err != nil {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, err
	}
	clientChainAddrLength := info.AddressLength

	assetAddr, ok := args[1].([]byte)
	if !ok || assetAddr == nil {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
	}
	// #nosec G115
	if uint32(len(assetAddr)) < clientChainAddrLength {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(assetAddr), clientChainAddrLength)
	}
	asset.Address = hexutil.Encode(assetAddr[:clientChainAddrLength])

	decimal, ok := args[2].(uint8)
	if !ok {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 2, "uint8", args[2])
	}
	// #nosec G115
	asset.Decimals = uint32(decimal)

	name, ok := args[3].(string)
	if !ok {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 3, "string", args[3])
	}
	if name == "" || len(name) > assetstypes.MaxChainTokenNameLength {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrInvalidNameLength, name, len(name), assetstypes.MaxChainTokenNameLength)
	}
	asset.Name = name

	metaInfo, ok := args[4].(string)
	if !ok {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 4, "string", args[4])
	}
	if metaInfo == "" || len(metaInfo) > assetstypes.MaxChainTokenMetaInfoLength {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrInvalidMetaInfoLength, metaInfo, len(metaInfo), assetstypes.MaxChainTokenMetaInfoLength)
	}
	asset.MetaInfo = metaInfo

	oracleInfoStr, ok := args[5].(string)
	if !ok {
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, fmt.Errorf(exocmn.ErrContractInputParaOrType, 5, "string", args[5])
	}
	parsed := strings.Split(oracleInfoStr, ",")
	l := len(parsed)
	switch {
	case l > 5:
		joined := strings.Join(parsed[5:], "")
		tokenDesc := tokenDescMatcher.FindStringSubmatch(joined)
		chainDesc := chainDescMatcher.FindStringSubmatch(joined)
		if len(tokenDesc) == 2 {
			oracleInfo.Token.Desc = tokenDesc[1]
		}
		if len(chainDesc) == 2 {
			oracleInfo.Chain.Desc = chainDesc[1]
		}
		fallthrough
	case l >= 5:
		oracleInfo.Token.Contract = parsed[4]
		fallthrough
	case l >= 4:
		oracleInfo.Feeder.Interval = parsed[3]
		fallthrough
	case l >= 3:
		oracleInfo.Token.Name = parsed[0]
		oracleInfo.Chain.Name = parsed[1]
		oracleInfo.Token.Decimal = parsed[2]
	default:
		return assetstypes.AssetInfo{}, oracletypes.OracleInfo{}, errors.New(exocmn.ErrInvalidOracleInfo)
	}

	return asset, oracleInfo, nil
}

func (p Precompile) UpdateTokenFromInputs(
	ctx sdk.Context, args []interface{},
) (clientChainID uint32, hexAssetAddr string, metadata string, err error) {
	inputsLen := len(p.ABI.Methods[MethodUpdateToken].Inputs)
	if len(args) != inputsLen {
		return 0, "", "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}

	clientChainID, ok := args[0].(uint32)
	if !ok {
		return 0, "", "", fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return 0, "", "", err
	}
	clientChainAddrLength := info.AddressLength
	assetAddr, ok := args[1].([]byte)
	if !ok || assetAddr == nil {
		return 0, "", "", fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
	}
	// #nosec G115
	if uint32(len(assetAddr)) < clientChainAddrLength {
		return 0, "", "", fmt.Errorf(exocmn.ErrInvalidAddrLength, len(assetAddr), clientChainAddrLength)
	}
	hexAssetAddr = hexutil.Encode(assetAddr[:clientChainAddrLength])

	metadata, ok = args[2].(string)
	if !ok {
		return 0, "", "", fmt.Errorf(exocmn.ErrContractInputParaOrType, 3, "string", args[3])
	}
	if len(metadata) > assetstypes.MaxChainTokenMetaInfoLength || len(metadata) == 0 {
		return 0, "", "", fmt.Errorf(exocmn.ErrInvalidMetaInfoLength, metadata, len(metadata), assetstypes.MaxChainTokenMetaInfoLength)
	}

	return clientChainID, hexAssetAddr, metadata, nil
}

func (p Precompile) ClientChainIDFromInputs(_ sdk.Context, args []interface{}) (uint32, error) {
	inputsLen := len(p.ABI.Methods[MethodIsRegisteredClientChain].Inputs)
	if len(args) != inputsLen {
		return 0, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return 0, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}
	return clientChainID, nil
}
