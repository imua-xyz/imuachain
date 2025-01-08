// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package assets

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// AssetsABI is the input ABI used to generate the binding from.
const AssetsABI = "[{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"assetsAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stakerAddress\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"depositLST\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"latestAssetState\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"validatorPubkey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stakerAddress\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"depositNST\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"latestAssetState\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getClientChains\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"uint32[]\",\"name\":\"\",\"type\":\"uint32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"}],\"name\":\"isRegisteredClientChain\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isRegistered\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"uint8\",\"name\":\"addressLength\",\"type\":\"uint8\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"metaInfo\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"signatureType\",\"type\":\"string\"}],\"name\":\"registerOrUpdateClientChain\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"updated\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainId\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"token\",\"type\":\"bytes\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"metaData\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"oracleInfo\",\"type\":\"string\"}],\"name\":\"registerToken\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainId\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"token\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"metaData\",\"type\":\"string\"}],\"name\":\"updateToken\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"assetsAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"withdrawAddress\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"withdrawLST\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"latestAssetState\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"validatorPubkey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"withdrawAddress\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"withdrawNST\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"latestAssetState\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Assets is an auto generated Go binding around an Ethereum contract.
type Assets struct {
	AssetsCaller     // Read-only binding to the contract
	AssetsTransactor // Write-only binding to the contract
	AssetsFilterer   // Log filterer for contract events
}

// AssetsCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssetsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssetsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssetsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssetsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssetsSession struct {
	Contract     *Assets           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AssetsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssetsCallerSession struct {
	Contract *AssetsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// AssetsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssetsTransactorSession struct {
	Contract     *AssetsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AssetsRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssetsRaw struct {
	Contract *Assets // Generic contract binding to access the raw methods on
}

// AssetsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssetsCallerRaw struct {
	Contract *AssetsCaller // Generic read-only contract binding to access the raw methods on
}

// AssetsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssetsTransactorRaw struct {
	Contract *AssetsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssets creates a new instance of Assets, bound to a specific deployed contract.
func NewAssets(address common.Address, backend bind.ContractBackend) (*Assets, error) {
	contract, err := bindAssets(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Assets{AssetsCaller: AssetsCaller{contract: contract}, AssetsTransactor: AssetsTransactor{contract: contract}, AssetsFilterer: AssetsFilterer{contract: contract}}, nil
}

// NewAssetsCaller creates a new read-only instance of Assets, bound to a specific deployed contract.
func NewAssetsCaller(address common.Address, caller bind.ContractCaller) (*AssetsCaller, error) {
	contract, err := bindAssets(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssetsCaller{contract: contract}, nil
}

// NewAssetsTransactor creates a new write-only instance of Assets, bound to a specific deployed contract.
func NewAssetsTransactor(address common.Address, transactor bind.ContractTransactor) (*AssetsTransactor, error) {
	contract, err := bindAssets(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssetsTransactor{contract: contract}, nil
}

// NewAssetsFilterer creates a new log filterer instance of Assets, bound to a specific deployed contract.
func NewAssetsFilterer(address common.Address, filterer bind.ContractFilterer) (*AssetsFilterer, error) {
	contract, err := bindAssets(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssetsFilterer{contract: contract}, nil
}

// bindAssets binds a generic wrapper to an already deployed contract.
func bindAssets(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AssetsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Assets *AssetsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Assets.Contract.AssetsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Assets *AssetsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Assets.Contract.AssetsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Assets *AssetsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Assets.Contract.AssetsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Assets *AssetsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Assets.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Assets *AssetsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Assets.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Assets *AssetsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Assets.Contract.contract.Transact(opts, method, params...)
}

// GetClientChains is a free data retrieval call binding the contract method 0x41a3745b.
//
// Solidity: function getClientChains() view returns(bool, uint32[])
func (_Assets *AssetsCaller) GetClientChains(opts *bind.CallOpts) (bool, []uint32, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "getClientChains")

	if err != nil {
		return *new(bool), *new([]uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([]uint32)).(*[]uint32)

	return out0, out1, err

}

// GetClientChains is a free data retrieval call binding the contract method 0x41a3745b.
//
// Solidity: function getClientChains() view returns(bool, uint32[])
func (_Assets *AssetsSession) GetClientChains() (bool, []uint32, error) {
	return _Assets.Contract.GetClientChains(&_Assets.CallOpts)
}

// GetClientChains is a free data retrieval call binding the contract method 0x41a3745b.
//
// Solidity: function getClientChains() view returns(bool, uint32[])
func (_Assets *AssetsCallerSession) GetClientChains() (bool, []uint32, error) {
	return _Assets.Contract.GetClientChains(&_Assets.CallOpts)
}

// IsRegisteredClientChain is a free data retrieval call binding the contract method 0x6b67d7f7.
//
// Solidity: function isRegisteredClientChain(uint32 clientChainID) view returns(bool success, bool isRegistered)
func (_Assets *AssetsCaller) IsRegisteredClientChain(opts *bind.CallOpts, clientChainID uint32) (struct {
	Success      bool
	IsRegistered bool
}, error) {
	var out []interface{}
	err := _Assets.contract.Call(opts, &out, "isRegisteredClientChain", clientChainID)

	outstruct := new(struct {
		Success      bool
		IsRegistered bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Success = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.IsRegistered = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// IsRegisteredClientChain is a free data retrieval call binding the contract method 0x6b67d7f7.
//
// Solidity: function isRegisteredClientChain(uint32 clientChainID) view returns(bool success, bool isRegistered)
func (_Assets *AssetsSession) IsRegisteredClientChain(clientChainID uint32) (struct {
	Success      bool
	IsRegistered bool
}, error) {
	return _Assets.Contract.IsRegisteredClientChain(&_Assets.CallOpts, clientChainID)
}

// IsRegisteredClientChain is a free data retrieval call binding the contract method 0x6b67d7f7.
//
// Solidity: function isRegisteredClientChain(uint32 clientChainID) view returns(bool success, bool isRegistered)
func (_Assets *AssetsCallerSession) IsRegisteredClientChain(clientChainID uint32) (struct {
	Success      bool
	IsRegistered bool
}, error) {
	return _Assets.Contract.IsRegisteredClientChain(&_Assets.CallOpts, clientChainID)
}

// DepositLST is a paid mutator transaction binding the contract method 0x497b2a74.
//
// Solidity: function depositLST(uint32 clientChainID, bytes assetsAddress, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactor) DepositLST(opts *bind.TransactOpts, clientChainID uint32, assetsAddress []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "depositLST", clientChainID, assetsAddress, stakerAddress, opAmount)
}

// DepositLST is a paid mutator transaction binding the contract method 0x497b2a74.
//
// Solidity: function depositLST(uint32 clientChainID, bytes assetsAddress, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsSession) DepositLST(clientChainID uint32, assetsAddress []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DepositLST(&_Assets.TransactOpts, clientChainID, assetsAddress, stakerAddress, opAmount)
}

// DepositLST is a paid mutator transaction binding the contract method 0x497b2a74.
//
// Solidity: function depositLST(uint32 clientChainID, bytes assetsAddress, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactorSession) DepositLST(clientChainID uint32, assetsAddress []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DepositLST(&_Assets.TransactOpts, clientChainID, assetsAddress, stakerAddress, opAmount)
}

// DepositNST is a paid mutator transaction binding the contract method 0x447956e0.
//
// Solidity: function depositNST(uint32 clientChainID, bytes validatorPubkey, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactor) DepositNST(opts *bind.TransactOpts, clientChainID uint32, validatorPubkey []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "depositNST", clientChainID, validatorPubkey, stakerAddress, opAmount)
}

// DepositNST is a paid mutator transaction binding the contract method 0x447956e0.
//
// Solidity: function depositNST(uint32 clientChainID, bytes validatorPubkey, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsSession) DepositNST(clientChainID uint32, validatorPubkey []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DepositNST(&_Assets.TransactOpts, clientChainID, validatorPubkey, stakerAddress, opAmount)
}

// DepositNST is a paid mutator transaction binding the contract method 0x447956e0.
//
// Solidity: function depositNST(uint32 clientChainID, bytes validatorPubkey, bytes stakerAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactorSession) DepositNST(clientChainID uint32, validatorPubkey []byte, stakerAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.DepositNST(&_Assets.TransactOpts, clientChainID, validatorPubkey, stakerAddress, opAmount)
}

// RegisterOrUpdateClientChain is a paid mutator transaction binding the contract method 0x1b315b52.
//
// Solidity: function registerOrUpdateClientChain(uint32 clientChainID, uint8 addressLength, string name, string metaInfo, string signatureType) returns(bool success, bool updated)
func (_Assets *AssetsTransactor) RegisterOrUpdateClientChain(opts *bind.TransactOpts, clientChainID uint32, addressLength uint8, name string, metaInfo string, signatureType string) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "registerOrUpdateClientChain", clientChainID, addressLength, name, metaInfo, signatureType)
}

// RegisterOrUpdateClientChain is a paid mutator transaction binding the contract method 0x1b315b52.
//
// Solidity: function registerOrUpdateClientChain(uint32 clientChainID, uint8 addressLength, string name, string metaInfo, string signatureType) returns(bool success, bool updated)
func (_Assets *AssetsSession) RegisterOrUpdateClientChain(clientChainID uint32, addressLength uint8, name string, metaInfo string, signatureType string) (*types.Transaction, error) {
	return _Assets.Contract.RegisterOrUpdateClientChain(&_Assets.TransactOpts, clientChainID, addressLength, name, metaInfo, signatureType)
}

// RegisterOrUpdateClientChain is a paid mutator transaction binding the contract method 0x1b315b52.
//
// Solidity: function registerOrUpdateClientChain(uint32 clientChainID, uint8 addressLength, string name, string metaInfo, string signatureType) returns(bool success, bool updated)
func (_Assets *AssetsTransactorSession) RegisterOrUpdateClientChain(clientChainID uint32, addressLength uint8, name string, metaInfo string, signatureType string) (*types.Transaction, error) {
	return _Assets.Contract.RegisterOrUpdateClientChain(&_Assets.TransactOpts, clientChainID, addressLength, name, metaInfo, signatureType)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x3a3e7f00.
//
// Solidity: function registerToken(uint32 clientChainId, bytes token, uint8 decimals, string name, string metaData, string oracleInfo) returns(bool success)
func (_Assets *AssetsTransactor) RegisterToken(opts *bind.TransactOpts, clientChainId uint32, token []byte, decimals uint8, name string, metaData string, oracleInfo string) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "registerToken", clientChainId, token, decimals, name, metaData, oracleInfo)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x3a3e7f00.
//
// Solidity: function registerToken(uint32 clientChainId, bytes token, uint8 decimals, string name, string metaData, string oracleInfo) returns(bool success)
func (_Assets *AssetsSession) RegisterToken(clientChainId uint32, token []byte, decimals uint8, name string, metaData string, oracleInfo string) (*types.Transaction, error) {
	return _Assets.Contract.RegisterToken(&_Assets.TransactOpts, clientChainId, token, decimals, name, metaData, oracleInfo)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x3a3e7f00.
//
// Solidity: function registerToken(uint32 clientChainId, bytes token, uint8 decimals, string name, string metaData, string oracleInfo) returns(bool success)
func (_Assets *AssetsTransactorSession) RegisterToken(clientChainId uint32, token []byte, decimals uint8, name string, metaData string, oracleInfo string) (*types.Transaction, error) {
	return _Assets.Contract.RegisterToken(&_Assets.TransactOpts, clientChainId, token, decimals, name, metaData, oracleInfo)
}

// UpdateToken is a paid mutator transaction binding the contract method 0xc7a919c7.
//
// Solidity: function updateToken(uint32 clientChainId, bytes token, string metaData) returns(bool success)
func (_Assets *AssetsTransactor) UpdateToken(opts *bind.TransactOpts, clientChainId uint32, token []byte, metaData string) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "updateToken", clientChainId, token, metaData)
}

// UpdateToken is a paid mutator transaction binding the contract method 0xc7a919c7.
//
// Solidity: function updateToken(uint32 clientChainId, bytes token, string metaData) returns(bool success)
func (_Assets *AssetsSession) UpdateToken(clientChainId uint32, token []byte, metaData string) (*types.Transaction, error) {
	return _Assets.Contract.UpdateToken(&_Assets.TransactOpts, clientChainId, token, metaData)
}

// UpdateToken is a paid mutator transaction binding the contract method 0xc7a919c7.
//
// Solidity: function updateToken(uint32 clientChainId, bytes token, string metaData) returns(bool success)
func (_Assets *AssetsTransactorSession) UpdateToken(clientChainId uint32, token []byte, metaData string) (*types.Transaction, error) {
	return _Assets.Contract.UpdateToken(&_Assets.TransactOpts, clientChainId, token, metaData)
}

// WithdrawLST is a paid mutator transaction binding the contract method 0xa900f232.
//
// Solidity: function withdrawLST(uint32 clientChainID, bytes assetsAddress, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactor) WithdrawLST(opts *bind.TransactOpts, clientChainID uint32, assetsAddress []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "withdrawLST", clientChainID, assetsAddress, withdrawAddress, opAmount)
}

// WithdrawLST is a paid mutator transaction binding the contract method 0xa900f232.
//
// Solidity: function withdrawLST(uint32 clientChainID, bytes assetsAddress, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsSession) WithdrawLST(clientChainID uint32, assetsAddress []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.WithdrawLST(&_Assets.TransactOpts, clientChainID, assetsAddress, withdrawAddress, opAmount)
}

// WithdrawLST is a paid mutator transaction binding the contract method 0xa900f232.
//
// Solidity: function withdrawLST(uint32 clientChainID, bytes assetsAddress, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactorSession) WithdrawLST(clientChainID uint32, assetsAddress []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.WithdrawLST(&_Assets.TransactOpts, clientChainID, assetsAddress, withdrawAddress, opAmount)
}

// WithdrawNST is a paid mutator transaction binding the contract method 0xf9238476.
//
// Solidity: function withdrawNST(uint32 clientChainID, bytes validatorPubkey, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactor) WithdrawNST(opts *bind.TransactOpts, clientChainID uint32, validatorPubkey []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.contract.Transact(opts, "withdrawNST", clientChainID, validatorPubkey, withdrawAddress, opAmount)
}

// WithdrawNST is a paid mutator transaction binding the contract method 0xf9238476.
//
// Solidity: function withdrawNST(uint32 clientChainID, bytes validatorPubkey, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsSession) WithdrawNST(clientChainID uint32, validatorPubkey []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.WithdrawNST(&_Assets.TransactOpts, clientChainID, validatorPubkey, withdrawAddress, opAmount)
}

// WithdrawNST is a paid mutator transaction binding the contract method 0xf9238476.
//
// Solidity: function withdrawNST(uint32 clientChainID, bytes validatorPubkey, bytes withdrawAddress, uint256 opAmount) returns(bool success, uint256 latestAssetState)
func (_Assets *AssetsTransactorSession) WithdrawNST(clientChainID uint32, validatorPubkey []byte, withdrawAddress []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Assets.Contract.WithdrawNST(&_Assets.TransactOpts, clientChainID, validatorPubkey, withdrawAddress, opAmount)
}
