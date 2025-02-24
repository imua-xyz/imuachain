package assets

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// oracleInfo: '[tokenName],[chainName],[tokenDecimal](,[interval],[contract](,[ChainDesc:{...}],[TokenDesc:{...}]))'
var (
	tokenDescMatcher = regexp.MustCompile(`TokenDesc:{(.+?)}`)
	chainDescMatcher = regexp.MustCompile(`ChainDesc:{(.+?)}`)
)

func (p Precompile) DepositWithdrawParams(ctx sdk.Context, method *abi.Method, args []interface{}) (*assetskeeper.DepositWithdrawParams, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[method.Name].Inputs)); err != nil {
		return nil, err
	}

	// Get client chain info first for address length validation
	clientChainID, err := ta.GetPositiveUint32(0)
	if err != nil {
		return nil, err
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, err
	}

	stakerAddr, err := ta.GetRequiredBytesPrefix(2, info.AddressLength)
	if err != nil {
		return nil, err
	}

	opAmount, err := ta.GetPositiveBigInt(3)
	if err != nil {
		return nil, err
	}

	params := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: uint64(clientChainID),
		StakerAddress:   stakerAddr,
		OpAmount:        sdkmath.NewIntFromBigInt(opAmount),
	}

	switch method.Name {
	case MethodDepositLST, MethodWithdrawLST:
		assetAddr, err := ta.GetRequiredBytesPrefix(1, info.AddressLength)
		if err != nil {
			return nil, err
		}

		params.AssetsAddress = assetAddr
		if method.Name == MethodDepositLST {
			params.Action = assetstypes.DepositLST
		} else {
			params.Action = assetstypes.WithdrawLST
		}
	case MethodDepositNST, MethodWithdrawNST:
		params.ValidatorID, err = ta.GetRequiredBytes(1) // In NST case, the second parameter is validatorID
		if err != nil {
			return nil, err
		}

		params.AssetsAddress = assetstypes.GenerateNSTAddr(info.AddressLength)
		if method.Name == MethodDepositNST {
			params.Action = assetstypes.DepositNST
		} else {
			params.Action = assetstypes.WithdrawNST
		}
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return params, nil
}

func (p Precompile) ClientChainInfoFromInputs(_ sdk.Context, args []interface{}) (*assetstypes.ClientChainInfo, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodRegisterOrUpdateClientChain].Inputs)); err != nil {
		return nil, err
	}

	clientChainID, err := ta.GetPositiveUint32(0)
	if err != nil {
		return nil, err
	}

	addressLength, err := ta.GetPositiveUint8(1)
	if err != nil {
		return nil, err
	}
	if addressLength < assetstypes.MinClientChainAddrLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidAddrLength, addressLength, assetstypes.MinClientChainAddrLength)
	}

	name, err := ta.GetRequiredString(2)
	if err != nil {
		return nil, err
	}
	if len(name) > assetstypes.MaxChainTokenNameLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidNameLength, name, len(name), assetstypes.MaxChainTokenNameLength)
	}

	metaInfo, err := ta.GetRequiredString(3)
	if err != nil {
		return nil, err
	}
	if metaInfo == "" || len(metaInfo) > assetstypes.MaxChainTokenMetaInfoLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidMetaInfoLength, metaInfo, len(metaInfo), assetstypes.MaxChainTokenMetaInfoLength)
	}

	signatureType, err := ta.GetString(4)
	if err != nil {
		return nil, err
	}

	return &assetstypes.ClientChainInfo{
		LayerZeroChainID: uint64(clientChainID),
		AddressLength:    uint32(addressLength),
		Name:             name,
		MetaInfo:         metaInfo,
		SignatureType:    signatureType,
	}, nil
}

func (p Precompile) TokenFromInputs(ctx sdk.Context, args []interface{}) (*assetstypes.AssetInfo, *oracletypes.OracleInfo, error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodRegisterToken].Inputs)); err != nil {
		return nil, nil, err
	}

	clientChainID, err := ta.GetPositiveUint32(0) // Must not be zero
	if err != nil {
		return nil, nil, err
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, nil, err
	}

	assetAddr, err := ta.GetRequiredHexAddress(1, info.AddressLength) // Must not be empty and must match length
	if err != nil {
		return nil, nil, err
	}

	decimal, err := ta.GetUint8(2) // No specific non-zero check needed
	if err != nil {
		return nil, nil, err
	}

	name, err := ta.GetRequiredString(3) // Must not be empty
	if err != nil {
		return nil, nil, err
	}
	if len(name) > assetstypes.MaxChainTokenNameLength {
		return nil, nil, fmt.Errorf(imuacmn.ErrInvalidNameLength, name, len(name), assetstypes.MaxChainTokenNameLength)
	}

	metaInfo, err := ta.GetRequiredString(4) // Must not be empty
	if err != nil {
		return nil, nil, err
	}
	if len(metaInfo) > assetstypes.MaxChainTokenMetaInfoLength {
		return nil, nil, fmt.Errorf(imuacmn.ErrInvalidMetaInfoLength, metaInfo, len(metaInfo), assetstypes.MaxChainTokenMetaInfoLength)
	}

	oracleInfoStr, err := ta.GetRequiredString(5) // Must not be empty
	if err != nil {
		return nil, nil, err
	}

	// Assign values to asset
	asset := &assetstypes.AssetInfo{
		LayerZeroChainID: uint64(clientChainID),
		Address:          assetAddr,
		Decimals:         uint32(decimal),
		Name:             name,
		MetaInfo:         metaInfo,
	}

	// Parse oracleInfoStr
	oracleInfo := &oracletypes.OracleInfo{}
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
		return nil, nil, errors.New(imuacmn.ErrInvalidOracleInfo)
	}

	return asset, oracleInfo, nil
}

func (p Precompile) UpdateTokenFromInputs(ctx sdk.Context, args []interface{}) (clientChainID uint32, hexAssetAddr string, metadata string, err error) {
	ta := NewTypedArgs(args)
	if err := ta.RequireLen(len(p.ABI.Methods[MethodUpdateToken].Inputs)); err != nil {
		return 0, "", "", err
	}

	clientChainID, err = ta.GetPositiveUint32(0)
	if err != nil {
		return 0, "", "", err
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return 0, "", "", err
	}

	hexAssetAddr, err = ta.GetRequiredHexAddress(1, info.AddressLength)
	if err != nil {
		return 0, "", "", err
	}

	metadata, err = ta.GetRequiredString(2)
	if err != nil {
		return 0, "", "", err
	}
	if len(metadata) > assetstypes.MaxChainTokenMetaInfoLength {
		return 0, "", "", fmt.Errorf(imuacmn.ErrInvalidMetaInfoLength, metadata, len(metadata), assetstypes.MaxChainTokenMetaInfoLength)
	}

	return clientChainID, hexAssetAddr, metadata, nil
}
