//go:build skiptests

package batch

import (
	"context"
	"fmt"
	"testing"

	"github.com/ExocoreNetwork/exocore/app"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	cryptohd "github.com/evmos/evmos/v16/crypto/hd"
	"github.com/evmos/evmos/v16/encoding"
	"github.com/stretchr/testify/assert"
)

func Test_RPC(t *testing.T) {
	ctx := context.Background()
	rc, err := rpc.DialContext(ctx, "http://127.0.0.1:8545")
	assert.NoError(t, err)
	c := ethclient.NewClient(rc)
	chainID, err := c.ChainID(ctx)
	assert.NoError(t, err)
	fmt.Println("the chain ID is:", chainID)

	rc, err = rpc.DialContext(ctx, "ws://127.0.0.1:8546")
	assert.NoError(t, err)
	c = ethclient.NewClient(rc)
	chainID, err = c.ChainID(ctx)
	assert.NoError(t, err)
	fmt.Println("the chain ID is:", chainID)
}

func Test_QueryBalance(t *testing.T) {
	appManager, err := NewManager(context.Background(), "/home/timmy/tests/test-tool", &DefaultTestToolConfig)
	assert.NoError(t, err)
	clientCtx := appManager.NodeClientCtx[DefaultNodeIndex]
	keyRecord, err := clientCtx.Keyring.Key(DefaultTestSKName)
	assert.NoError(t, err)
	fromAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the from address is:", fromAddr.String())

	fromAddr, err = sdktypes.AccAddressFromBech32("exo18cggcpvwspnd5c6ny8wrqxpffj5zmhklprtnph")
	assert.NoError(t, err)
	balances, err := appManager.QueryBalance(fromAddr)
	assert.NoError(t, err)
	fmt.Println(balances)
}

func Test_FaucetSk(t *testing.T) {
	appManager, err := NewManager(context.Background(), "/home/timmy/tests/test-tool", &DefaultTestToolConfig)
	assert.NoError(t, err)

	keyRing := appManager.KeyRing
	keyRecord, err := keyRing.Key(DefaultTestSKName)
	assert.NoError(t, err)
	keyRecordAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the key record address is:", keyRecordAddr.String())
	faucetEvmAddr := crypto.PubkeyToAddress(appManager.FaucetSK.PublicKey)
	fmt.Println("the faucet evm address is:", faucetEvmAddr.String())
	faucetExoAddr := sdktypes.AccAddress(faucetEvmAddr.Bytes())
	fmt.Println("the faucet exo address is:", faucetExoAddr.String())
	assert.Equal(t, faucetExoAddr, keyRecordAddr)
}

func Test_ImportPrivKeyHex(t *testing.T) {
	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	KeyRing := keyring.NewInMemory(encodingConfig.Codec, cryptohd.EthSecp256k1Option())
	signAlgoList1, signAlgoList2 := KeyRing.SupportedAlgorithms()
	fmt.Println("keyRing supported algothrim", signAlgoList1, signAlgoList2)
	// add the Sk to key ring
	keyName := "testKey"

	sk, err := crypto.HexToECDSA("D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE")
	assert.NoError(t, err)
	evmAddr := crypto.PubkeyToAddress(sk.PublicKey)
	fmt.Println("the evm address is:", evmAddr.String())
	exoAddr := sdktypes.AccAddress(evmAddr.Bytes())
	fmt.Println("the exo address is:", exoAddr.String())

	err = KeyRing.ImportPrivKeyHex(keyName, "D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE", ethsecp256k1.KeyType)
	assert.NoError(t, err)
	keyRecord, err := KeyRing.Key(keyName)
	assert.NoError(t, err)
	outPut, err := keyring.MkAccKeyOutput(keyRecord)
	fmt.Println(outPut)

	keyRecordAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the key record address is:", keyRecordAddr.String())
	assert.Equal(t, exoAddr.String(), keyRecordAddr.String())
}

func Test_Start(t *testing.T) {
	appManager, err := NewManager(context.Background(), "/home/timmy/tests/test-tool", &DefaultTestToolConfig)
	assert.NoError(t, err)
	err = appManager.Start()
	assert.NoError(t, err)
}
