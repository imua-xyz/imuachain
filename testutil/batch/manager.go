package batch

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	"github.com/ExocoreNetwork/exocore/types"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
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
	AppName             = "e2e-tool"
	DefaultAssetDecimal = 6
	DefaultTestSKName   = "default-test-sk"
	AssetNamePrefix     = "testAsset"
	StakerNamePrefix    = "testStaker"
	AVSNamePrefix       = "testAVS"
	OperatorNamePrefix  = "testOperator"
	DefaultNodeIndex    = 0
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
	config *EndToEndConfig
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

func NewManager(ctx context.Context, config EndToEndConfig) (*Manager, error) {
	// open the sqlite db
	dsn := "file:" + config.DBPathOrURL + "?cache=shared&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, xerrors.Errorf("can't open sqlite db, err:%s", err)
	}

	// get the private key for the virtual exocore gateway address
	// most test transactions will be signed by this private key.
	sk, err := crypto.HexToECDSA(config.FaucetSk)
	if err != nil {
		return nil, xerrors.Errorf("invalid faucet sk, err:%s", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	KeyRing, err := keyring.New(AppName, keyring.BackendTest, config.KeyRingDir, nil, cdc)
	if err != nil {
		return nil, xerrors.Errorf("can't new a test key ring, err:%s", err)
	}
	err = KeyRing.ImportPrivKeyHex(DefaultTestSKName, config.FaucetSk, secp256k1.KeyType)
	if err != nil {
		return nil, xerrors.Errorf("can't import the faucet private key, err:%s", err)
	}
	manager := &Manager{
		ctx:                ctx,
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
	for _, url := range config.NodesEVMRPCHTTP {
		rc, err := rpc.DialContext(context.Background(), url)
		if err != nil {
			return nil, xerrors.Errorf("can't creat the evm http rpc, err:%s, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		manager.NodeEVMHTTPClients = append(manager.NodeEVMHTTPClients, c)
	}
	// websocket clients
	for _, url := range config.NodesEVMRPCWebsocket {
		rc, err := rpc.DialContext(context.Background(), url)
		if err != nil {
			return nil, xerrors.Errorf("can't creat the evm websocket rpc, err:%s, url:%s", err, url)
		}
		c := ethclient.NewClient(rc)
		manager.NodeEVMWSClients = append(manager.NodeEVMWSClients, c)
	}
	// creat client context for the nodes
	for _, url := range config.NodesRPC {
		// Create a client context to connect to the Cosmos node
		clientCtx := client.Context{}.
			WithNodeURI(url).            // gRPC address of the Cosmos node
			WithChainID(config.ChainID). // Chain ID of the Cosmos blockchain
			WithKeyring(KeyRing)
		manager.NodeClientCtx = append(manager.NodeClientCtx, clientCtx)
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
		WaitDuration: time.Duration(config.TxCheckInterval),
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
	createNewAsset := func(id uint) (*Asset, error) {
		addr, _ := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", AssetNamePrefix, id)
		metaInfo := fmt.Sprintf("the meta info of %s", name)
		oracleInfo := fmt.Sprintf(
			"%s,Ethereum,%d,10,0xB82381A3fBD3FaFA77B3a7bE693342618240067b",
			name, DefaultAssetDecimal)
		return &Asset{
			Address:       addr,
			ClientChainID: m.config.DefaultClientChainID,
			Decimal:       DefaultAssetDecimal,
			Name:          name,
			OracleInfo:    oracleInfo,
			MetaInfo:      metaInfo,
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, &Asset{}, int64(m.config.AsssetNumber), createNewAsset)
}

func (m *Manager) CreateStakers() error {
	createNewStaker := func(id uint) (*Staker, error) {
		addr, privKey := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", StakerNamePrefix, id)
		// add the sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), secp256k1.KeyType)
		if err != nil {
			return nil, err
		}
		return &Staker{
			Address: addr,
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, &Staker{}, int64(m.config.StakerNumber), createNewStaker)
}

func (m *Manager) fetchAndSaveDogfoodAVS() error {
	dogfoodAvs, err := LoadObjectByID[AVS](m, SqliteDefaultStartID)
	if err != nil || dogfoodAvs. {
		return err
	}

	queryClient := avstypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	req := &avstypes.QueryAVSInfoReq{
		AVSAddress: m.DogfoodAddr,
	}
	res, err := queryClient.QueryAVSInfo(context.Background(), req)
	if err != nil {
		return err
	}
	// save the dogfood avs to local db
	err = SaveObject[AVS](m, AVS{})
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CreateAVS() error {
	createNewAVS := func(id uint) (*AVS, error) {
		addr, privKey := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", AVSNamePrefix, id)
		// add the sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), secp256k1.KeyType)
		if err != nil {
			return nil, err
		}
		return &AVS{
			Address: addr,
			Sk:      privKey.Bytes(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, &AVS{}, int64(m.config.AVSNumber), createNewAVS)
}

func (m *Manager) CreateOperators() error {
	createNewOperator := func(id uint) (*Operator, error) {
		addr, privKey := testutiltx.NewAccAddressAndKey()
		name := fmt.Sprintf("%s%d", OperatorNamePrefix, id)
		// add the sk to key ring
		err := m.KeyRing.ImportPrivKeyHex(name, common.Bytes2Hex(privKey.Key), secp256k1.KeyType)
		if err != nil {
			return nil, err
		}

		privVal := mock.NewPV()
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			return nil, err
		}
		consensusKey := keytypes.NewWrappedConsKeyFromHex(hexutil.Encode(pubKey.Bytes()))
		return &Operator{
			Name:            name,
			Address:         strings.ToLower(addr.String()),
			Sk:              privKey.Bytes(),
			ConsensusPubKey: consensusKey.ToJSON(),
			ConsensusSk:     privVal.PrivKey.String(),
		}, nil
	}
	// create a new Staker
	return CreateObjects(m, &Operator{}, int64(m.config.OperatorNumber), createNewOperator)
}

func (m *Manager) Start() error {

	return nil
}
