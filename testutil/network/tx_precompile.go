package network

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type precompile string

const (
	privateKey = "D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE" // gitleaks:allow
	gasLimit   = uint64(500000)

	ASSETS     precompile = "assets"
	DELEGATION precompile = "delegation"
)

var (
	sk, _           = crypto.HexToECDSA(privateKey) // gitleaks:allow
	callAddr        = crypto.PubkeyToAddress(sk.PublicKey)
	precompilePaths = map[precompile]string{
		ASSETS:     "../../precompiles/assets/abi.json",
		DELEGATION: "../../precompiles/delegation/abi.json",
	}
	abis                = make(map[precompile]abi.ABI)
	precompileAddresses = map[precompile]common.Address{
		ASSETS:     common.HexToAddress("0x0000000000000000000000000000000000000804"),
		DELEGATION: common.HexToAddress("0x0000000000000000000000000000000000000805"),
	}
)

// init loads and parse precompile abis
func init() {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("Failed to get current file path")
		panic("Failed to get current file path")
	}
	basePath := filepath.Dir(currentFile)
	var err error
	for precompileName, precompilePath := range precompilePaths {
		p := filepath.Join(basePath, precompilePath)
		abis[precompileName], err = parseABI(p)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse abi from path:%s\r\n", p))
		}
	}
}

// SendPrecompileTx wraps the function to send a transaction to precompile contract with gateway address
// which defined in the test genesis state
func (n Network) SendPrecompileTx(preCompileName precompile, methodName string, args ...interface{}) error {
	ctx := context.Background()

	ethC := n.Validators[0].JSONRPCClient

	chainID, err := ethC.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chainID, error:%w", err)
	}

	precompileAddr := precompileAddresses[preCompileName]
	precompileABI := abis[preCompileName]

	data, err := precompileABI.Pack(methodName, args...)
	if err != nil {
		return fmt.Errorf("failed to pack message for %s, error:%w", methodName, err)
	}

	nonce, err := ethC.NonceAt(ctx, callAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to get nonce for address %s: %w", callAddr.Hex(), err)
	}

	gasPrice, err := ethC.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get suggested gas price: %w", err)
	}

	retTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &precompileAddr,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signer := types.LatestSignerForChainID(chainID)
	signTx, err := types.SignTx(retTx, signer, sk)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	fmt.Println("the txID is:", signTx.Hash().String())
	msg := ethereum.CallMsg{
		From: callAddr,
		To:   retTx.To(),
		Data: retTx.Data(),
	}
	_, err = ethC.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call contract:%s, error:%w", preCompileName, err)
	}

	err = ethC.SendTransaction(ctx, signTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction, error:%w", err)
	}

	return nil
}

// parseABI parses abi from file
func parseABI(abiPath string) (abi.ABI, error) {
	f, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, err
	}
	defer f.Close()
	return abi.JSON(f)
}

// PaddingAddressTo32 pads the 20-length address to 32 bytes with 0s
func PaddingAddressTo32(address common.Address) []byte {
	ret := make([]byte, 32)
	copy(ret, address[:])
	return ret
}
