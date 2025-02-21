package batch

import (
	"crypto/ecdsa"
	"math/big"
	"strings"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	Queued               = iota // The tx has been stored in the local db but hasn't been sent to the node.
	Pending                     // Pending status
	OnChainButFailed            // On-chain but Failed
	OnChainAndSuccessful        // On-chain and successful
)

const (
	WaitToCheck = iota
	Successful
	Failed
)

const (
	SqliteDBFileName = "sqlite.db"
	ConfigFileName   = "test-tool-config.toml"
)

type TestToolConfig struct {
	// the numbers of staker, opeartor and AVS for e2e tests
	StakerNumber   int `mapstructure:"staker-number" toml:"staker-number"`
	OperatorNumber int `mapstructure:"operator-number" toml:"operator-number"`
	AVSNumber      int `mapstructure:"avs-number" toml:"avs-number"`
	AssetNumber    int `mapstructure:"asset-number" toml:"asset-number"`

	// the private key of faucet
	// and the IMUA amount that should be sent to the test addresses
	FaucetSk           string `mapstructure:"faucet-Sk" toml:"faucet-Sk"`
	StakerImuaAmount   int64  `mapstructure:"staker-imua-amount" toml:"staker-imua-amount"`
	OperatorImuaAmount int64  `mapstructure:"operator-imua-amount" toml:"operator-imua-amount"`
	AVSImuaAmount      int64  `mapstructure:"avs-imua-amount" toml:"avs-imua-amount"`

	// parameters for the testnet
	ChainValidatorNumber int    `mapstructure:"chain-validator-number" toml:"chain-validator-number"`
	ChainID              string `mapstructure:"chain-id" toml:"chain-id"`
	DefaultClientChainID uint32 `mapstructure:"default-client-chain-id" toml:"default-client-chain-id"`

	// the RPCs of the Imuachain chain nodes
	NodesRPC             []string `mapstructure:"nodes-rpc" toml:"nodes-rpc"`
	NodesEVMRPCHTTP      []string `mapstructure:"nodes-evm-rpc-http" toml:"nodes-evm-rpc-http"`
	NodesEVMRPCWebsocket []string `mapstructure:"nodes-evm-rpc-websocket" toml:"nodes-evm-rpc-websocket"`

	// the config of test strategy
	// the process of single test for each staker:
	// 1. deposit -> wait for transaction check
	// 2. delegation -> wait for transaction check
	// 3. undelegation -> wait for transaction check(only check whether the transaction is on chain)
	// 4. withdraw -> wait for transaction check
	// TxNumberPerSec indicates the number of transactions sent to Imuachain chain per second.
	// Currently, since we are using the `NoOpMempool`, it does not support sending transactions
	// with different nonces from the same sender at a very high rate. Therefore, it is recommended
	// to set this parameter to 1 for now. In the future,consider modifying this parameter configuration
	// to support slower transaction sending rates.
	TxNumberPerSec    int `mapstructure:"tx-number-per-second" toml:"tx-number-per-second"`
	TxChecksPerSecond int `mapstructure:"tx-checks-per-second" toml:"tx-checks-per-second"`
	// EachTestInterval indicates the interval of single test. The staker will start another test
	// After this interval.
	EachTestInterval int64 `mapstructure:"each-test-interval" toml:"each-test-interval"`
	// configure this parameter to test the voting power and reward distribution, the delegations will soon be
	// undelegated in the next batch test.
	IntervalAfterDelegations int64 `mapstructure:"interval-after-delegations" toml:"interval-after-delegations"`
	// SingleTxCheckInterval is an interval waiting to check the transaction.
	SingleTxCheckInterval int64 `mapstructure:"single-tx-check-interval" toml:"single-tx-check-interval"`
	// TxWaitExpiration defines the timeout period for checking a transaction.
	TxWaitExpiration int64 `mapstructure:"tx-wait-expiration" toml:"tx-wait-expiration"`
	// BatchTxsCheckInterval is an interval waiting to check the batch transactions.
	BatchTxsCheckInterval int64 `mapstructure:"batch-txs-check-interval" toml:"batch-txs-check-interval"`

	// AddrNumberInMultiSend is the number of address included in a multisend message, in order to
	// prevent the message from becoming too large and causing the transaction to exceed the packing limit
	AddrNumberInMultiSend int `mapstructure:"addr-number-in-multi-send" toml:"addr-number-in-multi-send"`
	// it's used to indicate the channel size
	TxsQueueBufferSize int `mapstructure:"tx-queue-buffer-size" toml:"tx-queue-buffer-size"`
}

// DefaultTestToolConfig is a default config for test by a local node
var DefaultTestToolConfig = TestToolConfig{
	StakerNumber:   5,
	OperatorNumber: 3,
	AVSNumber:      2,
	AssetNumber:    5,
	// this private key is from the local_node.sh
	FaucetSk:                 "D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE",
	StakerImuaAmount:         100,
	OperatorImuaAmount:       10,
	AVSImuaAmount:            10,
	ChainValidatorNumber:     1,
	ChainID:                  "imuachainlocalnet_232-1",
	DefaultClientChainID:     101,
	NodesRPC:                 []string{"http://127.0.0.1:26657"},
	NodesEVMRPCHTTP:          []string{"http://127.0.0.1:8545"},
	NodesEVMRPCWebsocket:     []string{"ws://127.0.0.1:8546"},
	TxNumberPerSec:           1,
	TxChecksPerSecond:        10,
	EachTestInterval:         10 * 60,     // 10 minutes
	IntervalAfterDelegations: 3 * 60 * 60, // 3 hours
	SingleTxCheckInterval:    6,           // 6 seconds
	TxWaitExpiration:         60,          // 1 minutes
	BatchTxsCheckInterval:    2 * 60,      // 2 minutes
	AddrNumberInMultiSend:    10,
	TxsQueueBufferSize:       100,
}

type HelperRecord struct {
	ID             uint `gorm:"primaryKey"` // primary key
	CurrentBatchID uint
}

type Asset struct {
	ID            uint           `gorm:"primaryKey"`                   // primary key
	Address       common.Address `gorm:"column:address;type:char(42)"` // eth address
	ClientChainID uint32
	Decimal       uint8
	Name          string `gorm:"type:varchar(50)"`
	MetaInfo      string `gorm:"type:varchar(200)"`
	OracleInfo    string
}

type Staker struct {
	ID           uint `gorm:"primaryKey"` // primary key
	Name         string
	Address      common.Address `gorm:"column:address;type:char(42)"` // eth address
	Sk           []byte         // private key
	Transactions []Transaction  `gorm:"foreignKey:StakerID"`
	// AvailableNonce int64
}

type Transaction struct {
	ID                 uint `gorm:"primaryKey"`
	StakerID           uint `gorm:"index"`
	AssetID            uint
	OperatorID         uint
	Type               string
	IsCosmosTx         bool
	TxHash             string
	OpAmount           string
	Nonce              uint64
	Status             int
	CheckResult        int
	SendHeight         uint64
	OnChainHeight      uint64
	SendTime           string
	OnChainTime        string
	TestBatchID        uint
	ExpectedCheckValue string
	ActualCheckValue   string
}

type Operator struct {
	ID              uint `gorm:"primaryKey"` // primary key
	Name            string
	Address         string // cosmos address
	Sk              []byte // private key
	ConsensusPubKey string
	ConsensusSk     string
}

type AVS struct {
	ID      uint `gorm:"primaryKey"` // primary key
	Name    string
	Address string // eth address
	Sk      []byte // private key
}

type AddressForFunding interface {
	AccAddress() sdktypes.AccAddress
	EvmAddress() common.Address
	ShouldFund() bool
	ObjectName() string
}

func (a AVS) AccAddress() sdktypes.AccAddress {
	return common.HexToAddress(a.Address).Bytes()
}

func (a AVS) EvmAddress() common.Address {
	return common.HexToAddress(a.Address)
}

func (a AVS) ShouldFund() bool {
	return !a.IsDogfood()
}

func (a AVS) IsDogfood() bool {
	return a.Name == DogfoodAVSName
}

func (a AVS) ObjectName() string {
	return a.Name
}

func (o Operator) AccAddress() sdktypes.AccAddress {
	accAddr, _ := sdktypes.AccAddressFromBech32(o.Address)
	return accAddr
}

func (o Operator) EvmAddress() common.Address {
	accAddr, _ := sdktypes.AccAddressFromBech32(o.Address)
	return common.BytesToAddress(accAddr[:common.AddressLength])
}

func (o Operator) ShouldFund() bool {
	return !o.IsDefaultOperator()
}

func (o Operator) IsDefaultOperator() bool {
	return strings.HasPrefix(o.Name, DefaultOperatorNamePrefix)
}

func (o Operator) ObjectName() string {
	return o.Name
}

func (s Staker) AccAddress() sdktypes.AccAddress {
	return s.Address.Bytes()
}

func (s Staker) EvmAddress() common.Address {
	return s.Address
}

func (s Staker) ShouldFund() bool {
	return true
}

func (s Staker) ObjectName() string {
	return s.Name
}

type EvmTxInQueue struct {
	Sk               *ecdsa.PrivateKey
	From             common.Address
	UseExternalNonce bool
	Nonce            uint64
	ToAddr           *common.Address
	Value            *big.Int
	Data             []byte
	TxRecord         *Transaction
}

type CosmosMsgInQueue struct{}
