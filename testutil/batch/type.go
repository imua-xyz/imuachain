package batch

import (
	"crypto/ecdsa"
	"math/big"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	Queued               = iota // The tx has been stored in the local db but hasn't been sent to the node.
	Pending                     // Pending status
	OnChainButFailed            // On-chain but failed
	OnChainAndSuccessful        // On-chain and successful
)

const (
	WaitToCheck = iota
	Successful
	failed
)

type EndToEndConfig struct {
	// the numbers of staker, opeartor and AVS for e2e tests
	StakerNumber   int `mapstructure:"staker-number"`
	OperatorNumber int `mapstructure:"operator-number"`
	AVSNumber      int `mapstructure:"avs-number"`
	AsssetNumber   int `mapstructure:"asset-number"`

	// the private key of faucet
	// and the Exo amount that should be sent to the test addresses
	FaucetSk          string `mapstructure:"faucet-sk"`
	StakerExoAmount   int64  `mapstructure:"staker-exo-amount"`
	OperatorExoAmount int64  `mapstructure:"operator-exo-amount"`
	AVSExoAmount      int64  `mapstructure:"avs-exo-amount"`

	// parameters for the testnet
	ChainValidatorNumber int    `mapstructure:"chain-validator-number"`
	ChainID              string `mapstructure:"chain-id"`
	DefaultClientChainID uint64 `mapstructure:"default-client-chain-id"`
	BlockDuration        int64  `mapstructure:"block-duration"`

	// the file path or URL of local DB, and the RPCs of the Exocore chain nodes
	KeyRingDir           string   `mapstructure:"key-ring-dir"`
	DBPathOrURL          string   `mapstructure:"db-path-or-url"`
	NodesRPC             []string `mapstructure:"nodes-rpc"`
	NodesEVMRPCHTTP      []string `mapstructure:"nodes-evm-rpc-http"`
	NodesEVMRPCWebsocket []string `mapstructure:"nodes-evm-rpc-websocket"`

	// the config of test strategy
	// the process of single test for each staker:
	// 1. deposit -> wait for transaction check
	// 2. delegation -> wait for transaction check
	// 3. undelegation -> wait for transaction check(only check whether the transaction is on chain)
	// 4. withdraw -> wait for transaction check
	// TxNumberPerSec indicates the number of transactions sent to Exocore chain per second.
	TxNumberPerSec    int `mapstructure:"tx-number-per-second"`
	TxChecksPerSecond int `mapstructure:"tx-checks-per-second"`
	// EachTestInterval indicates the interval of single test. The staker will start another test
	// After this interval.
	EachTestInterval int64 `mapstructure:"each-test-interval"`
	// TxCheckInterval is an interval waiting to check the transaction.
	TxCheckInterval int64 `mapstructure:"tx-check-interval"`
	// AddrNumberInMultiSend is the number of address included in a multisend message, in order to
	// prevent the message from becoming too large and causing the transaction to exceed the packing limit
	AddrNumberInMultiSend int `mapstructure:"addr-number-in-multi-send"`
	// it's used to indicate the channel size
	TxsQueueBufferSize int `mapstructure:"tx-queue-buffer-size"`
}

type HelperRecord struct {
	ID             uint `gorm:"primaryKey"` // primary key
	CurrentBatchID uint
}

type Asset struct {
	ID            uint           `gorm:"primaryKey"`                   // primary key
	Address       common.Address `gorm:"column:address;type:char(42)"` // eth address
	ClientChainID uint64
	Decimal       uint8
	Name          string `gorm:"type:varchar(50)"`
	MetaInfo      string `gorm:"type:varchar(200)"`
	OracleInfo    string
}

type Staker struct {
	ID           uint           `gorm:"primaryKey"`                   // primary key
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
	ID      uint           `gorm:"primaryKey"`                   // primary key
	Address common.Address `gorm:"column:address;type:char(42)"` // eth address
	Sk      []byte         // private key
}

type Addressable interface {
	AccAddress() sdktypes.AccAddress
	EvmAddress() common.Address
}

func (a AVS) AccAddress() sdktypes.AccAddress {
	return a.Address.Bytes()
}

func (a AVS) EvmAddress() common.Address {
	return a.Address
}

func (o Operator) AccAddress() sdktypes.AccAddress {
	accAddr, _ := sdktypes.AccAddressFromBech32(o.Address)
	return accAddr
}

func (o Operator) EvmAddress() common.Address {
	accAddr, _ := sdktypes.AccAddressFromBech32(o.Address)
	return common.BytesToAddress(accAddr[:common.AddressLength])
}

func (s Staker) AccAddress() sdktypes.AccAddress {
	return s.Address.Bytes()
}

func (s Staker) EvmAddress() common.Address {
	return s.Address
}

type EvmTxInQueue struct {
	sk               *ecdsa.PrivateKey
	From             common.Address
	UseExternalNonce bool
	Nonce            uint64
	ToAddr           *common.Address
	Value            *big.Int
	Data             []byte
	TxRecordID       uint
}

type CosmosMsgInQueue struct{}
