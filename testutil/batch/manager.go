package batch

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v16/crypto/hd"
	"github.com/evmos/evmos/v16/encoding"
	"github.com/imua-xyz/imuachain/app"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/imua-xyz/imuachain/precompiles/assets"
	"github.com/imua-xyz/imuachain/precompiles/delegation"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	AppName                   = "e2e-tool"
	DefaultAssetDecimal       = 6
	FaucetSKName              = "faucet-Sk"
	AssetNamePrefix           = "testAsset"
	StakerNamePrefix          = "testStaker"
	AVSNamePrefix             = "testAVS"
	DogfoodAVSName            = "dogfood"
	OperatorNamePrefix        = "testOperator"
	DefaultOperatorNamePrefix = "defaultOperator"
	DefaultNodeIndex          = 0
)

var (
	ImuaDecimalReduction     = new(big.Int).Exp(big.NewInt(10), big.NewInt(types.BaseDenomUnit), nil)
	logger                   = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	AssetsPrecompileAddr     = common.HexToAddress("0x0000000000000000000000000000000000000804")
	DelegationPrecompileAddr = common.HexToAddress("0x0000000000000000000000000000000000000805")
	AVSPrecompileAddr        = common.HexToAddress("0x0000000000000000000000000000000000000901")
	MaxUnbondingDuration     = uint64(10)

	DefaultOperatorCommission = stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
)

type Manager struct {
	ctx    context.Context
	config *TestToolConfig
	db     *gorm.DB

	DogfoodAddr        string
	FaucetSK           *ecdsa.PrivateKey
	Sequences          sync.Map
	KeyRing            keyring.Keyring
	NodeEVMHTTPClients []*ethclient.Client
	NodeEVMWSClients   []*ethclient.Client
	NodeClientCtx      []client.Context
	EthSigner          ethtypes.Signer
	WaitDuration       time.Duration
	WaitExpiration     time.Duration

	QueueSize atomic.Int32
	TxsQueue  chan interface{}
	Shutdown  chan bool
}

func NewManager(ctx context.Context, homePath string, config *TestToolConfig) (*Manager, error) {
	// open the sqlite db
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, xerrors.Errorf("can't open sqlite db, err:%w", err)
	}
	// SQLite waits for 600000 milliseconds (10 minute) when encountering a lock conflict.
	db.Exec("PRAGMA busy_timeout = 600000;")

	// get the private key for the virtual imua gateway address
	// most test transactions will be signed by this private key.
	sk, err := crypto.HexToECDSA(config.FaucetSk)
	if err != nil {
		return nil, xerrors.Errorf("invalid faucet Sk:%s, err:%w", config.FaucetSk, err)
	}

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	KeyRing, err := keyring.New(AppName, keyring.BackendTest, homePath, nil, encodingConfig.Codec, hd.EthSecp256k1Option())
	if err != nil {
		return nil, xerrors.Errorf("can't new a test key ring, err:%w", err)
	}
	_, err = KeyRing.Key(FaucetSKName)
	if err != nil {
		err = KeyRing.ImportPrivKeyHex(FaucetSKName, config.FaucetSk, ethsecp256k1.KeyType)
		if err != nil {
			return nil, xerrors.Errorf("failed to import the faucet private key '%s' into keyring: %w",
				FaucetSKName, err)
		}
	}

	manager := &Manager{
		ctx:                ctx,
		config:             config,
		db:                 db,
		FaucetSK:           sk,
		KeyRing:            KeyRing,
		DogfoodAddr:        avstypes.GenerateAVSAddress(avstypes.ChainIDWithoutRevision(config.ChainID)),
		NodeEVMHTTPClients: make([]*ethclient.Client, config.ChainValidatorNumber),
		NodeEVMWSClients:   make([]*ethclient.Client, config.ChainValidatorNumber),
		NodeClientCtx:      make([]client.Context, config.ChainValidatorNumber),
		TxsQueue:           make(chan interface{}, config.TxsQueueBufferSize),
		Shutdown:           make(chan bool),
	}

	// creat the evm clients
	// http clients
	for i, url := range config.NodesEVMRPCHTTP {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many http rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		logger.Info("http url", "url", url)
		rc, err := rpc.DialContext(manager.ctx, url)
		if err != nil {
			return nil, xerrors.Errorf("can't create the evm http rpc, err:%w, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		manager.NodeEVMHTTPClients[i] = c
	}
	// websocket clients
	for i, url := range config.NodesEVMRPCWebsocket {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many websocket rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		logger.Info("websocket url", "url", url)
		rc, err := rpc.DialContext(manager.ctx, url)
		if err != nil {
			return nil, xerrors.Errorf("can't create the evm websocket rpc, err:%w, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		manager.NodeEVMWSClients[i] = c
	}
	// creat client context for the nodes
	for i, url := range config.NodesRPC {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many node rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		logger.Info("node rpc url", "url", url)
		// Create a client context to connect to the Cosmos node
		clientCtx := client.Context{}.
			WithNodeURI(url).            // gRPC address of the Cosmos node
			WithChainID(config.ChainID). // Chain ID of the Cosmos blockchain
			WithKeyring(KeyRing).
			WithKeyringOptions(hd.EthSecp256k1Option()).
			WithCodec(encodingConfig.Codec).
			WithTxConfig(encodingConfig.TxConfig).
			WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
			WithAccountRetriever(authtypes.AccountRetriever{}).
			WithSkipConfirmation(true)

		client, err := client.NewClientFromNode(url)
		if err != nil {
			return nil, xerrors.Errorf("can't new client from node,url:%s,err:%w", url, err)
		}
		clientCtx = clientCtx.WithClient(client)
		manager.NodeClientCtx[i] = clientCtx
	}

	evmChainID, err := manager.NodeEVMHTTPClients[DefaultNodeIndex].ChainID(ctx)
	if err != nil {
		return nil, xerrors.Errorf("can't get the evm chainID, err:%w", err)
	}
	ethSigner := ethtypes.LatestSignerForChainID(evmChainID)
	manager.EthSigner = ethSigner
	manager.WaitDuration = time.Duration(config.SingleTxCheckInterval) * time.Second
	manager.WaitExpiration = time.Duration(config.TxWaitExpiration) * time.Second
	return manager, nil
}

func (m *Manager) Close() {
	for _, client := range m.NodeEVMHTTPClients {
		client.Close()
	}
	for _, client := range m.NodeEVMWSClients {
		client.Close()
	}
}

func (m *Manager) GetDB() *gorm.DB {
	return m.db
}

func (m *Manager) InitHelperRecord() error {
	helperRecord, err := LoadObjectByID[HelperRecord](m.GetDB(), SqliteDefaultStartID)
	logger.Info("InitHelperRecord load helper record", "err", err, "helperRecord", helperRecord)
	batchID := uint(0)
	if err == nil {
		// increase the batch id, because we use a new batch id to avoid check error
		// every time the test-tool is started
		batchID = helperRecord.CurrentBatchID + 1
	}
	logger.Info("InitHelperRecord the new test batch ID is:", "batchID", batchID)
	err = SaveObject[HelperRecord](m.GetDB(), HelperRecord{CurrentBatchID: batchID, ID: SqliteDefaultStartID})
	if err != nil {
		return xerrors.Errorf("can't init the helper record, err:%w", err)
	}
	return nil
}

func (m *Manager) CreateAssets() error {
	createNewAsset := func(id uint) (Asset, error) {
		addr, _ := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", AssetNamePrefix, id)
		metaInfo := fmt.Sprintf("the meta info of %s", name)
		oracleInfo := fmt.Sprintf(
			"%s,Ethereum,%d,10,0xB82381A3fBD3FaFA77B3a7bE693342618240067b",
			name, DefaultAssetDecimal)
		return Asset{
			Address:       addr,
			ClientChainID: m.config.DefaultClientChainID,
			Decimal:       DefaultAssetDecimal,
			Name:          name,
			OracleInfo:    oracleInfo,
			MetaInfo:      metaInfo,
		}, nil
	}
	// create a new Staker
	return CreateObjects(m.GetDB(), Asset{}, int64(m.config.AssetNumber), createNewAsset)
}

func (m *Manager) CreateStakers() error {
	createNewStaker := func(id uint) (Staker, error) {
		addr, privKey := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", StakerNamePrefix, id)
		// add the Sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), ethsecp256k1.KeyType)
		if err != nil {
			return Staker{}, xerrors.Errorf("failed to import private key for staker %s: %w", name, err)
		}
		return Staker{
			Name:    name,
			Address: addr,
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m.GetDB(), Staker{}, int64(m.config.StakerNumber), createNewStaker)
}

func (m *Manager) SaveDogfoodAVS() error {
	if m.GetDB().Migrator().HasTable(&AVS{}) {
		var avs AVS
		// Query the database for the AVS record with the specified address.
		err := m.GetDB().Where("address = ?", m.DogfoodAddr).First(&avs).Error
		// Check if the error is "record not found", indicating that the address does not exist in the database.
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// For other types of errors, return a detailed error message.
			return xerrors.Errorf("Failed to load AVS with address %s, err: %w", m.DogfoodAddr, err)
		} else if err == nil {
			// the dogfood AVS has been saved.
			logger.Info("SaveDogfoodAVS, already saved")
			return nil
		}
	}

	// save the dogfood avs to local db
	err := SaveObject[AVS](m.GetDB(), AVS{
		Name:    DogfoodAVSName,
		Address: m.DogfoodAddr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CreateAVS() error {
	createNewAVS := func(id uint) (AVS, error) {
		addr, privKey := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", AVSNamePrefix, id)
		// add the Sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), ethsecp256k1.KeyType)
		if err != nil {
			return AVS{}, xerrors.Errorf("failed to import private key for avs %s: %w", name, err)
		}
		return AVS{
			Name:    name,
			Address: strings.ToLower(addr.String()),
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m.GetDB(), AVS{}, int64(m.config.AVSNumber), createNewAVS)
}

func (m *Manager) SaveDefaultOperator() error {
	clientCtx := m.NodeClientCtx[DefaultNodeIndex]
	queryClient := operatortypes.NewQueryClient(clientCtx)
	req := &operatortypes.QueryAllOperatorsRequest{
		Pagination: &query.PageRequest{},
	}
	res, err := queryClient.QueryAllOperators(context.Background(), req)
	if err != nil {
		return err
	}
	for i, operatorAddr := range res.OperatorAccAddrs {
		if m.GetDB().Migrator().HasTable(&Operator{}) {
			var operator Operator
			// Query the database for the operator record with the specified address.
			err := m.GetDB().Where("address = ?", operatorAddr).First(&operator).Error
			// Check if the error is "record not found", indicating that the address does not exist in the database.
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				// For other types of errors, return a detailed error message.
				return xerrors.Errorf("Failed to load operator with address %s, err: %w", operatorAddr, err)
			} else if err == nil {
				logger.Info("SaveDefaultOperator, already saved", "operator", operatorAddr)
				continue
			}
		}

		// save the dogfood avs to local db
		err := SaveObject[Operator](m.GetDB(), Operator{
			Address: operatorAddr,
			Name:    fmt.Sprintf("%s%d", DefaultOperatorNamePrefix, i),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) CreateOperators() error {
	createNewOperator := func(id uint) (Operator, error) {
		addr, privKey := testutiltx.NewAccAddressAndKey()
		name := fmt.Sprintf("%s%d", OperatorNamePrefix, id)
		// add the Sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), ethsecp256k1.KeyType)
		if err != nil {
			return Operator{}, xerrors.Errorf("failed to import private key for operator %s: %w", name, err)
		}

		privVal := mock.NewPV()
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			return Operator{}, xerrors.Errorf("failed to get public key for operator %s: %w", name, err)
		}
		consensusKey := keytypes.NewWrappedConsKeyFromHex(hexutil.Encode(pubKey.Bytes()))
		return Operator{
			Name:            name,
			Address:         strings.ToLower(addr.String()),
			Sk:              privKey.Bytes(),
			ConsensusPubKey: consensusKey.ToJSON(),
			ConsensusSk:     privVal.PrivKey.String(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m.GetDB(), Operator{}, int64(m.config.OperatorNumber), createNewOperator)
}

func (m *Manager) Prepare() error {
	// save the dogfood AVS and the default operators to the local db
	if err := m.SaveDogfoodAVS(); err != nil {
		return xerrors.Errorf("Failed to save dogfood AVS,err:%w", err)
	}
	if err := m.SaveDefaultOperator(); err != nil {
		return xerrors.Errorf("Failed to save default operators,err:%w", err)
	}
	// create the assets, stakers, operators and AVSs for batch test
	if err := m.CreateAssets(); err != nil {
		return xerrors.Errorf("Failed to create assets,err:%w", err)
	}
	if err := m.CreateAVS(); err != nil {
		return xerrors.Errorf("Failed to create AVSs,err:%w", err)
	}
	if err := m.CreateStakers(); err != nil {
		return xerrors.Errorf("Failed to create stakers,err:%w", err)
	}
	if err := m.CreateOperators(); err != nil {
		return xerrors.Errorf("Failed to create operators,err:%w", err)
	}
	logger.Info("finish creating and saving test objects, next step: funding")
	// init the sequences
	if err := m.InitSequences(); err != nil {
		return xerrors.Errorf("Failed to init the sequences,err:%w", err)
	}
	// funding all test objects, EthSigner is faucet sk, tx type is cosmos
	if err := m.Funding(); err != nil {
		return xerrors.Errorf("Failed to fund the test objects,err:%w", err)
	}
	if err := m.FundingCheck(); err != nil {
		return xerrors.Errorf("Failed to check funding, err:%w", err)
	}
	logger.Info("finish funding test objects, next step: registration")
	// register the test objects, EthSigner is faucet sk, tx type is evm
	assets, err := m.RegisterAssets()
	if err != nil {
		return xerrors.Errorf("Failed to register assets,err:%w", err)
	}
	err = m.AssetsCheck(nil)
	if err != nil {
		return xerrors.Errorf("Failed to check assets,err:%w", err)
	}
	logger.Info("finish registering test assets, next step: operator registration")
	// register the operators, EthSigner is the operator sk, tx type is cosmos
	if err = m.RegisterOperators(); err != nil {
		return xerrors.Errorf("Failed to register operators,err:%w", err)
	}
	err = m.OperatorsCheck(nil)
	if err != nil {
		return xerrors.Errorf("Failed to check operators,err:%w", err)
	}
	logger.Info("finish registering test operators, next step: AVS registration")
	// EthSigner is the AVS sk, tx type is evm
	if err = m.RegisterAVSs(assets); err != nil {
		return xerrors.Errorf("Failed to register AVss,err:%w", err)
	}
	err = m.AVSsCheck(nil)
	if err != nil {
		return xerrors.Errorf("Failed to check AVSs,err:%w", err)
	}
	logger.Info("finish registering test AVSs, next step: add assets to dogfood")
	// add the test assets to the supported list of dogfood AVS, EthSigner is the AVS sk
	// tx type is cosmos
	if err = m.AddAssetsToDogfoodAVS(assets); err != nil {
		return xerrors.Errorf("Failed to add the test assets to the dofood AVS,err:%w", err)
	}
	_, isUpdate, err := m.DogfoodAssetsCheck(assets)
	if err != nil || isUpdate {
		return xerrors.Errorf("Failed to check the assets list of dogfood,isUpdate:%v,err:%w", isUpdate, err)
	}
	logger.Info("finish adding assets to dogfood, next step: opts the operators to AVSs")
	// opt all test operators to all test AVSs, EthSigner is the operator sk
	// tx type is cosmos
	if err = m.OptOperatorsIntoAVSs(); err != nil {
		return xerrors.Errorf("Failed to opt all operators to all AVSs,err:%w", err)
	}
	err = m.OperatorsOptInCheck(nil)
	if err != nil {
		return xerrors.Errorf("Failed to check the operator's opt-in status,err:%w", err)
	}
	logger.Info("finish the preparation for the batch test")
	return nil
}

func (m *Manager) EnqueueAndCheckTxsInBatch(msgType string) error {
	helperRecord, err := LoadObjectByID[HelperRecord](m.GetDB(), SqliteDefaultStartID)
	logger.Info("EnqueueAndCheckTxsInBatch load helper record", "err", err, "helperRecord", helperRecord)
	if err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: Failed to load the helper record", "err", err)
		return err
	}
	var enqueueTxsFunc func(batchID uint, msgType string) error
	var txsCheckFunc func(batchID uint, msgType string) error
	switch msgType {
	case assets.MethodDepositLST, assets.MethodWithdrawLST:
		enqueueTxsFunc = m.EnqueueDepositWithdrawLSTTxs
		txsCheckFunc = m.DepositWithdrawLSTCheck
	case delegation.MethodDelegate, delegation.MethodUndelegate:
		enqueueTxsFunc = m.EnqueueDelegationTxs
		txsCheckFunc = m.EvmDelegationCheck
	default:
		return xerrors.Errorf("EnqueueAndCheckTxsInBatch, invalid msgType:%s", msgType)
	}
	logger.Info("EnqueueAndCheckTxsInBatch: call enqueueTxsFunc", "msgType", msgType, "batchID", helperRecord.CurrentBatchID)
	if err := enqueueTxsFunc(helperRecord.CurrentBatchID, msgType); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: Failed to test transactions in batch", "msgType", msgType, "err", err)
		return err
	}
	// When configuring this interval, the durations of sending and on-chain processing should be taken into account.
	time.Sleep(time.Duration(m.config.BatchTxsCheckInterval) * time.Second)
	// Check if all test transactions have been dequeued; if not, wait for them to be dequeued.
	for {
		if m.QueueSize.Load() > 0 {
			time.Sleep(time.Duration(m.config.BatchTxsCheckInterval) * time.Second)
		} else {
			break
		}
	}
	logger.Info("EnqueueAndCheckTxsInBatch: call PrecompileTxOnChainCheck", "msgType", msgType, "batchID", helperRecord.CurrentBatchID)
	if err := m.PrecompileTxOnChainCheck(helperRecord.CurrentBatchID, msgType); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: Failed to check whether the txs are on chain", "err", err)
		return err
	}
	logger.Info("EnqueueAndCheckTxsInBatch: call txsCheckFunc", "msgType", msgType)
	if err := txsCheckFunc(helperRecord.CurrentBatchID, msgType); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: Failed to check the txs", "msgType", msgType, "err", err)
		return err
	}
	// increase the batch id if the msg type is withdrawal
	if msgType == assets.MethodWithdrawLST {
		helperRecord.CurrentBatchID++
		if err := SaveObject(m.GetDB(), helperRecord); err != nil {
			logger.Error("EnqueueAndCheckTxsInBatch: can't save the helper record")
			return err
		}
		logger.Info("EnqueueAndCheckTxsInBatch update the batchID", "batchID", helperRecord.CurrentBatchID)
	}
	return nil
}

func (m *Manager) WaitForShuttingDown() {
	// Set up channel to listen for OS interrupt signals (like Ctrl+C)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for an interrupt signal
	<-stopChan
	// Shutdown appManager gracefully
	fmt.Println("Shutting down...")
	close(m.Shutdown)
	m.Close()
}

func (m *Manager) ExecuteBatchTestForType(msgType string) error {
	if err := m.InitHelperRecord(); err != nil {
		return err
	}
	// send test transactions in batch
	go func() {
		// enqueue the txs and check them in batch
		if err := m.EnqueueAndCheckTxsInBatch(msgType); err != nil {
			close(m.Shutdown)
			return
		}
		logger.Info("ExecuteBatchTestForType: finish enqueuing and checking all txs")
	}()
	if err := m.DequeueAndSignSendTxs(); err != nil {
		logger.Error("ExecuteBatchTestForType, Failed to dequeue and sign send the txs,err:%w", err)
		return err
	}
	return nil
}

func (m *Manager) Start() error {
	if err := m.Prepare(); err != nil {
		return err
	}
	if err := m.InitHelperRecord(); err != nil {
		return err
	}
	eg, ctx := errgroup.WithContext(m.ctx)
	m.ctx = ctx
	// send test transactions in batch
	eg.Go(func() error {
		// deposit
		if err := m.EnqueueAndCheckTxsInBatch(assets.MethodDepositLST); err != nil {
			return xerrors.Errorf("deposit batch failed: %w", err)
		}
		// delegation
		if err := m.EnqueueAndCheckTxsInBatch(delegation.MethodDelegate); err != nil {
			return xerrors.Errorf("delegation batch failed: %w", err)
		}
		// undelegation
		if err := m.EnqueueAndCheckTxsInBatch(delegation.MethodUndelegate); err != nil {
			return xerrors.Errorf("undelegation batch failed: %w", err)
		}
		// withdrawal
		if err := m.EnqueueAndCheckTxsInBatch(assets.MethodWithdrawLST); err != nil {
			return xerrors.Errorf("withdrawal batch failed: %w", err)
		}
		logger.Info("Start: finish enqueuing and checking all withdrawal txs", "EachTestInterval", m.config.EachTestInterval)
		time.Sleep(time.Duration(m.config.EachTestInterval) * time.Second)
		if err := m.FundAndCheckStakers(); err != nil {
			return xerrors.Errorf("failed to fund and check the stakers %w", err)
		}
		return nil
	})

	// Dequeue the transactions and send them to the node.
	// This function will be blocked unless it receives the signal for shutting down.
	eg.Go(func() error {
		if err := m.DequeueAndSignSendTxs(); err != nil {
			return xerrors.Errorf("failed to dequeue and send transactions: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		close(m.Shutdown)
		logger.Error("start: error occurs, err:%v", err)
		return err
	}
	return nil
}
