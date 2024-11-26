package batch

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ExocoreNetwork/exocore/app"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v16/crypto/hd"
	"github.com/evmos/evmos/v16/encoding"

	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types/query"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	"github.com/ExocoreNetwork/exocore/types"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
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
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	AppName                   = "e2e-tool"
	DefaultAssetDecimal       = 6
	DefaultTestSKName         = "default-test-Sk"
	AssetNamePrefix           = "testAsset"
	StakerNamePrefix          = "testStaker"
	AVSNamePrefix             = "testAVS"
	DogfoodAVSName            = "dogfood"
	OperatorNamePrefix        = "testOperator"
	DefaultOperatorNamePrefix = "defaultOperator"
	DefaultNodeIndex          = 0
)

var (
	ExoDecimalReduction  = new(big.Int).Exp(big.NewInt(10), big.NewInt(types.BaseDenomUnit), nil)
	logger               = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	AssetsPrecompileAddr = common.HexToAddress("0x0000000000000000000000000000000000000804")
	AVSPrecompileAddr    = common.HexToAddress("0x0000000000000000000000000000000000000901")
	AllEpochs            = []string{
		epochstypes.MinuteEpochID,
		epochstypes.HourEpochID,
		epochstypes.DayEpochID,
		epochstypes.WeekEpochID,
	}
	MaxUnbondingDuration = uint64(10)

	DefaultOperatorCommission = stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
)

type Manager struct {
	ctx    context.Context
	config *TestToolConfig
	db     *gorm.DB
	lock   sync.Mutex

	DogfoodAddr              string
	FaucetSK                 *ecdsa.PrivateKey
	KeyRing                  keyring.Keyring
	NodeEVMHTTPClients       []*ethclient.Client
	NodeEVMWSClients         []*ethclient.Client
	NodeClientCtx            []client.Context
	DefaultEvmTxRequirements *BasicEvmTxRequirements

	TxsQueue chan interface{}
	Shutdown chan bool
}

func NewManager(ctx context.Context, homePath string, config *TestToolConfig) (*Manager, error) {
	// open the sqlite db
	dsn := "file:" + filepath.Join(homePath, SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, xerrors.Errorf("can't open sqlite db, err:%s", err)
	}

	// get the private key for the virtual exocore gateway address
	// most test transactions will be signed by this private key.
	sk, err := crypto.HexToECDSA(config.FaucetSk)
	if err != nil {
		return nil, xerrors.Errorf("invalid faucet Sk:%s, err:%s", config.FaucetSk, err)
	}

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	KeyRing, err := keyring.New(AppName, keyring.BackendTest, homePath, nil, encodingConfig.Codec, hd.EthSecp256k1Option())
	if err != nil {
		return nil, xerrors.Errorf("can't new a test key ring, err:%s", err)
	}
	_, err = KeyRing.Key(DefaultTestSKName)
	if err != nil {
		err = KeyRing.ImportPrivKeyHex(DefaultTestSKName, config.FaucetSk, ethsecp256k1.KeyType)
		if err != nil {
			return nil, xerrors.Errorf("can't import the faucet private key, err:%s", err)
		}
	}

	manager := &Manager{
		ctx:                ctx,
		config:             config,
		db:                 db,
		FaucetSK:           sk,
		KeyRing:            KeyRing,
		DogfoodAddr:        avstypes.GenerateAVSAddr(avstypes.ChainIDWithoutRevision(config.ChainID)),
		NodeEVMHTTPClients: make([]*ethclient.Client, config.ChainValidatorNumber),
		NodeEVMWSClients:   make([]*ethclient.Client, config.ChainValidatorNumber),
		NodeClientCtx:      make([]client.Context, config.ChainValidatorNumber),
		TxsQueue:           make(chan interface{}, config.TxsQueueBufferSize),
		Shutdown:           make(chan bool),
	}
	if err != nil {
		err = SaveObject[HelperRecord](manager, HelperRecord{CurrentBatchID: 0})
		if err != nil {
			return nil, xerrors.Errorf("can't init the helper record, err:%s", err)
		}
	}
	// creat the evm clients
	// http clients
	for i, url := range config.NodesEVMRPCHTTP {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many http rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		fmt.Println("http url", url)
		rc, err := rpc.DialContext(context.Background(), url)
		if err != nil {
			return nil, xerrors.Errorf("can't creat the evm http rpc, err:%s, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		fmt.Println("the evm http rpc is:", c)
		manager.NodeEVMHTTPClients[i] = c

		fmt.Println("the node evm http clients is:", len(manager.NodeEVMHTTPClients), manager.NodeEVMHTTPClients)
		evmChainID, err := manager.NodeEVMHTTPClients[DefaultNodeIndex].ChainID(ctx)
		if err != nil {
			return nil, xerrors.Errorf("can't get the evm chainID, err:%s", err)
		}
		fmt.Println("http the evm chain id is:", evmChainID)
	}
	// websocket clients
	for i, url := range config.NodesEVMRPCWebsocket {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many websocket rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		fmt.Println("websocket url", url)
		rc, err := rpc.DialContext(context.Background(), url)
		if err != nil {
			return nil, xerrors.Errorf("can't creat the evm websocket rpc, err:%s, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		fmt.Println("the evm ws rpc is:", c)
		manager.NodeEVMWSClients[i] = c

		evmChainID, err := manager.NodeEVMWSClients[DefaultNodeIndex].ChainID(ctx)
		if err != nil {
			return nil, xerrors.Errorf("can't get the evm chainID, err:%s", err)
		}
		fmt.Println("ws the evm chain id is:", evmChainID)
	}
	// creat client context for the nodes
	for i, url := range config.NodesRPC {
		if i >= config.ChainValidatorNumber {
			return nil, xerrors.Errorf("too many node rpc,index:%d,nodeNumber:%d", i, config.ChainValidatorNumber)
		}
		fmt.Println("node rpc url", url)
		// Create a client context to connect to the Cosmos node
		clientCtx := client.Context{}.
			WithNodeURI(url).            // gRPC address of the Cosmos node
			WithChainID(config.ChainID). // Chain ID of the Cosmos blockchain
			WithKeyring(KeyRing).
			WithKeyringOptions(hd.EthSecp256k1Option())
		fmt.Println("the node client ctx:", clientCtx)

		client, err := client.NewClientFromNode(url)
		if err != nil {
			return nil, xerrors.Errorf("can't new client from node,url:%s,err:%s", url, err)
		}
		clientCtx = clientCtx.WithClient(client)
		manager.NodeClientCtx[i] = clientCtx
	}

	evmChainID, err := manager.NodeEVMHTTPClients[DefaultNodeIndex].ChainID(ctx)
	if err != nil {
		return nil, xerrors.Errorf("can't get the evm chainID, err:%s", err)
	}
	ethSigner := ethtypes.LatestSignerForChainID(evmChainID)
	manager.DefaultEvmTxRequirements = &BasicEvmTxRequirements{
		ctx:          manager.ctx,
		sk:           manager.FaucetSK,
		caller:       crypto.PubkeyToAddress(sk.PublicKey),
		signer:       ethSigner,
		ethC:         manager.NodeEVMHTTPClients[DefaultNodeIndex],
		WaitDuration: time.Duration(config.SingleTxCheckInterval),
	}
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
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.db
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
	fmt.Println("m.config", m.config, m.config.AsssetNumber)
	return CreateObjects(m, Asset{}, int64(m.config.AsssetNumber), createNewAsset)
}

func (m *Manager) CreateStakers() error {
	createNewStaker := func(id uint) (Staker, error) {
		addr, privKey := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", StakerNamePrefix, id)
		// add the Sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), ethsecp256k1.KeyType)
		if err != nil {
			return Staker{}, err
		}
		return Staker{
			Name:    name,
			Address: addr,
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, Staker{}, int64(m.config.StakerNumber), createNewStaker)
}

func (m *Manager) SaveDogfoodAVS() error {
	if m.GetDB().Migrator().HasTable(&AVS{}) {
		var avs AVS
		// Query the database for the AVS record with the specified address.
		err := m.GetDB().Where("address = ?", m.DogfoodAddr).First(&avs).Error
		// Check if the error is "record not found", indicating that the address does not exist in the database.
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// For other types of errors, return a detailed error message.
			return xerrors.Errorf("Failed to load AVS with address %s, err: %s", m.DogfoodAddr, err)
		} else if err == nil {
			// the dogfood AVS has been saved.
			logger.Info("SaveDogfoodAVS, already saved")
			return nil
		}
	}

	// save the dogfood avs to local db
	err := SaveObject[AVS](m, AVS{
		Name:    DogfoodAVSName,
		Address: m.DogfoodAddr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) AddAssetsToDogfoodAVS(allAssetsID []string) error {
	clientCtx := m.NodeClientCtx[DefaultNodeIndex]
	queryClient := dogfoodtypes.NewQueryClient(clientCtx)

	res, err := queryClient.Params(m.ctx, &dogfoodtypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	// Create a set (using a map) to keep track of existing AssetIDs in res.Params.AssetIDs
	existingAssets := make(map[string]struct{})

	// Populate the map with the current AssetIDs
	for _, id := range res.Params.AssetIDs {
		existingAssets[id] = struct{}{}
	}

	// Iterate through allAssetsID to find any missing IDs
	isUpdateParam := false
	for _, id := range allAssetsID {
		if _, exists := existingAssets[id]; !exists {
			// If the ID is missing, add it to the existing AssetIDs
			res.Params.AssetIDs = append(res.Params.AssetIDs, id)
			isUpdateParam = true
		}
	}
	if isUpdateParam {
		record, err := m.KeyRing.Key(DefaultTestSKName)
		if err != nil {
			return err
		}
		// Retrieve the address from the Record
		address, err := record.GetAddress()
		if err != nil {
			return err
		}
		msg := &dogfoodtypes.MsgUpdateParams{
			Authority: address.String(),
			Params:    res.Params,
		}
		err = SignAndSendMultiMsgs(clientCtx, DefaultTestSKName, flags.BroadcastSync, msg)
		if err != nil {
			return err
		}
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
			return AVS{}, err
		}
		return AVS{
			Name:    name,
			Address: strings.ToLower(addr.String()),
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, AVS{}, int64(m.config.AVSNumber), createNewAVS)
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
				return xerrors.Errorf("Failed to load operator with address %s, err: %s", operatorAddr, err)
			} else if err == nil {
				logger.Info("SaveDefaultOperator, already saved", "operator", operatorAddr)
				continue
			}
		}

		// save the dogfood avs to local db
		err := SaveObject[Operator](m, Operator{
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
			return Operator{}, err
		}

		privVal := mock.NewPV()
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			return Operator{}, err
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
	return CreateObjects(m, Operator{}, int64(m.config.OperatorNumber), createNewOperator)
}

func (m *Manager) Prepare() error {
	// save the dogfood AVS and the default operators to the local db
	if err := m.SaveDogfoodAVS(); err != nil {
		return xerrors.Errorf("failed to save dogfood AVS,err:%s", err)
	}
	if err := m.SaveDefaultOperator(); err != nil {
		return xerrors.Errorf("failed to save default operators,err:%s", err)
	}
	// create the assets, stakers, operators and AVSs for batch test
	if err := m.CreateAssets(); err != nil {
		return xerrors.Errorf("failed to create assets,err:%s", err)
	}
	if err := m.CreateAVS(); err != nil {
		return xerrors.Errorf("failed to create AVSs,err:%s", err)
	}
	if err := m.CreateStakers(); err != nil {
		return xerrors.Errorf("failed to create stakers,err:%s", err)
	}
	if err := m.CreateOperators(); err != nil {
		return xerrors.Errorf("failed to create operators,err:%s", err)
	}
	logger.Info("finish creating and saving test objects, next step: funding")
	// funding all test objects
	if err := m.Funding(); err != nil {
		return xerrors.Errorf("failed to fund the test objects,err:%s", err)
	}
	logger.Info("finish funding test objects, next step: registration")
	// register the test objects
	assets, err := m.RegisterAssets()
	if err != nil {
		return xerrors.Errorf("failed to register assets,err:%s", err)
	}
	if err = m.RegisterOperators(); err != nil {
		return xerrors.Errorf("failed to register operators,err:%s", err)
	}
	if err = m.RegisterAVSs(assets); err != nil {
		return xerrors.Errorf("failed to register AVss,err:%s", err)
	}
	// add the test assets to the supported list of dogfood AVS
	if err = m.AddAssetsToDogfoodAVS(assets); err != nil {
		return xerrors.Errorf("failed to add the test assets to the dofood AVS,err:%s", err)
	}
	// opt all test operators to all test AVSs
	if err = m.OptOperatorsIntoAVSs(); err != nil {
		return xerrors.Errorf("failed to opt all operators to all AVSs,err:%s", err)
	}
	logger.Info("finish the preparation for the batch test")
	return nil
}

func (m *Manager) EnqueueAndCheckTxsInBatch(msgType string) error {
	helperRecord, err := LoadObjectByID[HelperRecord](m, SqliteDefaultStartID)
	if err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: failed to load the helper record", "err", err)
		return err
	}
	var enqueueTxsFunc func(msgType string) error
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
	if err := enqueueTxsFunc(msgType); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: failed to test transactions in batch", "msgType", msgType, "err", err)
		return err
	}
	// When configuring this interval, the durations of sending and on-chain processing should be taken into account.
	time.Sleep(time.Duration(m.config.BatchTxsCheckInterval) * time.Second)
	if err := m.TxOnChainCheck(helperRecord.CurrentBatchID, assets.MethodDepositLST); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: failed to check whether the txs are on chain", "err", err)
		return err
	}
	if err := txsCheckFunc(helperRecord.CurrentBatchID, assets.MethodDepositLST); err != nil {
		logger.Error("EnqueueAndCheckTxsInBatch: failed to check the txs", "msgType", msgType, "err", err)
		return err
	}
	// increase the batch id if the msg type is withdrawal
	if msgType == assets.MethodWithdrawLST {
		helperRecord.CurrentBatchID++
		if err := SaveObject(m, helperRecord); err != nil {
			logger.Error("EnqueueAndCheckTxsInBatch: can't save the helper record")
			return err
		}
	}
	return nil
}

func (m *Manager) Start() error {
	if err := m.Prepare(); err != nil {
		return err
	}
	// send test transactions in batch
	go func() {
		// deposit
		if err := m.EnqueueAndCheckTxsInBatch(assets.MethodDepositLST); err != nil {
			return
		}
		// delegation
		if err := m.EnqueueAndCheckTxsInBatch(delegation.MethodDelegate); err != nil {
			return
		}
		// undelegation
		if err := m.EnqueueAndCheckTxsInBatch(delegation.MethodUndelegate); err != nil {
			return
		}
		// withdrawal
		if err := m.EnqueueAndCheckTxsInBatch(assets.MethodWithdrawLST); err != nil {
			return
		}
		time.Sleep(time.Duration(m.config.EachTestInterval) * time.Second)
	}()
	// Dequeue the transactions and send them to the node.
	// This function will be blocked unless it receives the signal for shutting down.
	if err := m.DequeueAndSignSendTxs(); err != nil {
		logger.Error("DequeueAndSignSendTxs, err:%s", err)
		return err
	}
	return nil
}
