//go:build skiptests

// Used to debug the test tool. Standard unit tests should launch a running node
// through the startInProcess function, similar to the tests in the network directory.
package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/precompiles/assets"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	cryptohd "github.com/evmos/evmos/v16/crypto/hd"
	"github.com/evmos/evmos/v16/encoding"
	"github.com/imua-xyz/imuachain/app"
	"github.com/stretchr/testify/assert"
)

func newTestAppManager(t *testing.T) *Manager {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	configPath := filepath.Join(homeDir, "tests/test-tool")
	appManager, err := NewManager(context.Background(), configPath, &DefaultTestToolConfig)
	assert.NoError(t, err)
	return appManager
}

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

func Test_QueryAllBalance(t *testing.T) {
	appManager := newTestAppManager(t)
	assert.NoError(t, err)
	clientCtx := appManager.NodeClientCtx[DefaultNodeIndex]
	keyRecord, err := clientCtx.Keyring.Key(FaucetSKName)
	assert.NoError(t, err)
	fromAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the from address is:", fromAddr.String())

	fromAddr, err = sdktypes.AccAddressFromBech32("im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj")
	assert.NoError(t, err)
	balances, err := appManager.QueryAllBalance(fromAddr)
	assert.NoError(t, err)
	fmt.Println(balances)
}

func Test_FaucetSk(t *testing.T) {
	appManager := newTestAppManager(t)
	assert.NoError(t, err)

	keyRing := appManager.KeyRing
	keyRecord, err := keyRing.Key(FaucetSKName)
	assert.NoError(t, err)
	keyRecordAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the key record address is:", keyRecordAddr.String())
	faucetEvmAddr := crypto.PubkeyToAddress(appManager.FaucetSK.PublicKey)
	fmt.Println("the faucet evm address is:", faucetEvmAddr.String())
	faucetImAddr := sdktypes.AccAddress(faucetEvmAddr.Bytes())
	fmt.Println("the faucet im address is:", faucetImAddr.String())
	assert.Equal(t, faucetImAddr, keyRecordAddr)
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
	imAddr := sdktypes.AccAddress(evmAddr.Bytes())
	fmt.Println("the im address is:", imAddr.String())

	err = KeyRing.ImportPrivKeyHex(keyName, "D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE", ethsecp256k1.KeyType)
	assert.NoError(t, err)
	keyRecord, err := KeyRing.Key(keyName)
	assert.NoError(t, err)
	outPut, err := keyring.MkAccKeyOutput(keyRecord)
	fmt.Println(outPut)

	keyRecordAddr, err := keyRecord.GetAddress()
	assert.NoError(t, err)
	fmt.Println("the key record address is:", keyRecordAddr.String())
	assert.Equal(t, imAddr.String(), keyRecordAddr.String())
}

func Test_queryEvmTx(t *testing.T) {
	appManager := newTestAppManager(t)
	assert.NoError(t, err)
	txID := common.HexToHash("0x1c48a4a180d68ee52bb4e3d4fc479f5eee99d9a83b770a20850fca9f38043b8f")
	evmHTTPClient := appManager.NodeEVMHTTPClients[DefaultNodeIndex]
	receipt, err := evmHTTPClient.TransactionReceipt(appManager.ctx, txID)
	assert.NoError(t, err)
	receiptBytes, err := json.MarshalIndent(receipt, " ", " ")
	assert.NoError(t, err)
	fmt.Println(string(receiptBytes))
}

func Test_LoadAllAVSs(t *testing.T) {
	// open the sqlite db
	homePath := "/home/timmy/tests/test-tool"
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	opFunc := func(id uint, objectNumber int64, object AVS) error {
		objectBytes, err := json.MarshalIndent(&object, " ", " ")
		assert.NoError(t, err)
		fmt.Println(string(objectBytes))
		fmt.Println("the evm and cosmos address is:", object.EvmAddress(), object.AccAddress().String())
		return nil
	}
	err = IterateObjects(db, AVS{}, opFunc)
	assert.NoError(t, err)
}

func Test_LoadTxRecords(t *testing.T) {
	// open the sqlite db
	homePath := "/home/timmy/tests/test-tool"
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	txDbIDs, _, err := GetTxIDsByBatchTypeAndStatus(db, 0, assets.MethodDepositLST, Pending)
	assert.NoError(t, err)
	for _, txDbID := range txDbIDs {
		txRecord, err := LoadObjectByID[Transaction](db, txDbID)
		assert.NoError(t, err)
		txRecordBytes, err := json.MarshalIndent(&txRecord, " ", " ")
		assert.NoError(t, err)
		fmt.Println(string(txRecordBytes))
	}
}

func Test_LoadAllTxRecords(t *testing.T) {
	// open the sqlite db
	homePath := "/home/timmy/tests/test-tool"
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	opFunc := func(id uint, objectNumber int64, object Transaction) error {
		objectBytes, err := json.MarshalIndent(&object, " ", " ")
		assert.NoError(t, err)
		fmt.Println(string(objectBytes))
		return nil
	}
	err = IterateObjects(db, Transaction{}, opFunc)
	assert.NoError(t, err)
}

func Test_LoadAllStakers(t *testing.T) {
	// open the sqlite db
	homePath := "/home/timmy/tests/test-tool"
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	opFunc := func(id uint, objectNumber int64, object Staker) error {
		objectBytes, err := json.MarshalIndent(&object, " ", " ")
		assert.NoError(t, err)
		fmt.Println(string(objectBytes))
		return nil
	}
	err = IterateObjects(db, Staker{}, opFunc)
	assert.NoError(t, err)
}

func Test_LoadAllAssets(t *testing.T) {
	// open the sqlite db
	homePath := "/home/timmy/tests/test-tool"
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	opFunc := func(id uint, objectNumber int64, object Asset) error {
		objectBytes, err := json.MarshalIndent(&object, " ", " ")
		assert.NoError(t, err)
		fmt.Println(string(objectBytes))
		return nil
	}
	err = IterateObjects(db, Asset{}, opFunc)
	assert.NoError(t, err)
}

func Test_Start(t *testing.T) {
	appManager := newTestAppManager(t)
	assert.NoError(t, err)
	err = appManager.Start()
	assert.NoError(t, err)
}
