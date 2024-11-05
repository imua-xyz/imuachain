// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package delegation

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

// DelegationABI is the input ABI used to generate the binding from.
const DelegationABI = "[{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"staker\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"operator\",\"type\":\"bytes\"}],\"name\":\"associateOperatorWithStaker\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"lzNonce\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"assetsAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stakerAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"operatorAddr\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"delegate\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"staker\",\"type\":\"bytes\"}],\"name\":\"dissociateOperatorFromStaker\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"clientChainID\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"lzNonce\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"assetsAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"stakerAddress\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"operatorAddr\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"opAmount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Delegation is an auto generated Go binding around an Ethereum contract.
type Delegation struct {
	DelegationCaller     // Read-only binding to the contract
	DelegationTransactor // Write-only binding to the contract
	DelegationFilterer   // Log filterer for contract events
}

// DelegationCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegationSession struct {
	Contract     *Delegation       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DelegationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegationCallerSession struct {
	Contract *DelegationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// DelegationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegationTransactorSession struct {
	Contract     *DelegationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// DelegationRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegationRaw struct {
	Contract *Delegation // Generic contract binding to access the raw methods on
}

// DelegationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegationCallerRaw struct {
	Contract *DelegationCaller // Generic read-only contract binding to access the raw methods on
}

// DelegationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegationTransactorRaw struct {
	Contract *DelegationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegation creates a new instance of Delegation, bound to a specific deployed contract.
func NewDelegation(address common.Address, backend bind.ContractBackend) (*Delegation, error) {
	contract, err := bindDelegation(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Delegation{DelegationCaller: DelegationCaller{contract: contract}, DelegationTransactor: DelegationTransactor{contract: contract}, DelegationFilterer: DelegationFilterer{contract: contract}}, nil
}

// NewDelegationCaller creates a new read-only instance of Delegation, bound to a specific deployed contract.
func NewDelegationCaller(address common.Address, caller bind.ContractCaller) (*DelegationCaller, error) {
	contract, err := bindDelegation(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegationCaller{contract: contract}, nil
}

// NewDelegationTransactor creates a new write-only instance of Delegation, bound to a specific deployed contract.
func NewDelegationTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegationTransactor, error) {
	contract, err := bindDelegation(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegationTransactor{contract: contract}, nil
}

// NewDelegationFilterer creates a new log filterer instance of Delegation, bound to a specific deployed contract.
func NewDelegationFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegationFilterer, error) {
	contract, err := bindDelegation(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegationFilterer{contract: contract}, nil
}

// bindDelegation binds a generic wrapper to an already deployed contract.
func bindDelegation(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DelegationABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Delegation *DelegationRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Delegation.Contract.DelegationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Delegation *DelegationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Delegation.Contract.DelegationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Delegation *DelegationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Delegation.Contract.DelegationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Delegation *DelegationCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Delegation.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Delegation *DelegationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Delegation.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Delegation *DelegationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Delegation.Contract.contract.Transact(opts, method, params...)
}

// AssociateOperatorWithStaker is a paid mutator transaction binding the contract method 0xf221f9e7.
//
// Solidity: function associateOperatorWithStaker(uint32 clientChainID, bytes staker, bytes operator) returns(bool success)
func (_Delegation *DelegationTransactor) AssociateOperatorWithStaker(opts *bind.TransactOpts, clientChainID uint32, staker []byte, operator []byte) (*types.Transaction, error) {
	return _Delegation.contract.Transact(opts, "associateOperatorWithStaker", clientChainID, staker, operator)
}

// AssociateOperatorWithStaker is a paid mutator transaction binding the contract method 0xf221f9e7.
//
// Solidity: function associateOperatorWithStaker(uint32 clientChainID, bytes staker, bytes operator) returns(bool success)
func (_Delegation *DelegationSession) AssociateOperatorWithStaker(clientChainID uint32, staker []byte, operator []byte) (*types.Transaction, error) {
	return _Delegation.Contract.AssociateOperatorWithStaker(&_Delegation.TransactOpts, clientChainID, staker, operator)
}

// AssociateOperatorWithStaker is a paid mutator transaction binding the contract method 0xf221f9e7.
//
// Solidity: function associateOperatorWithStaker(uint32 clientChainID, bytes staker, bytes operator) returns(bool success)
func (_Delegation *DelegationTransactorSession) AssociateOperatorWithStaker(clientChainID uint32, staker []byte, operator []byte) (*types.Transaction, error) {
	return _Delegation.Contract.AssociateOperatorWithStaker(&_Delegation.TransactOpts, clientChainID, staker, operator)
}

// Delegate is a paid mutator transaction binding the contract method 0x870c5b03.
//
// Solidity: function delegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationTransactor) Delegate(opts *bind.TransactOpts, clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.contract.Transact(opts, "delegate", clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}

// Delegate is a paid mutator transaction binding the contract method 0x870c5b03.
//
// Solidity: function delegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationSession) Delegate(clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.Contract.Delegate(&_Delegation.TransactOpts, clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}

// Delegate is a paid mutator transaction binding the contract method 0x870c5b03.
//
// Solidity: function delegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationTransactorSession) Delegate(clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.Contract.Delegate(&_Delegation.TransactOpts, clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}

// DissociateOperatorFromStaker is a paid mutator transaction binding the contract method 0x1a004d5a.
//
// Solidity: function dissociateOperatorFromStaker(uint32 clientChainID, bytes staker) returns(bool success)
func (_Delegation *DelegationTransactor) DissociateOperatorFromStaker(opts *bind.TransactOpts, clientChainID uint32, staker []byte) (*types.Transaction, error) {
	return _Delegation.contract.Transact(opts, "dissociateOperatorFromStaker", clientChainID, staker)
}

// DissociateOperatorFromStaker is a paid mutator transaction binding the contract method 0x1a004d5a.
//
// Solidity: function dissociateOperatorFromStaker(uint32 clientChainID, bytes staker) returns(bool success)
func (_Delegation *DelegationSession) DissociateOperatorFromStaker(clientChainID uint32, staker []byte) (*types.Transaction, error) {
	return _Delegation.Contract.DissociateOperatorFromStaker(&_Delegation.TransactOpts, clientChainID, staker)
}

// DissociateOperatorFromStaker is a paid mutator transaction binding the contract method 0x1a004d5a.
//
// Solidity: function dissociateOperatorFromStaker(uint32 clientChainID, bytes staker) returns(bool success)
func (_Delegation *DelegationTransactorSession) DissociateOperatorFromStaker(clientChainID uint32, staker []byte) (*types.Transaction, error) {
	return _Delegation.Contract.DissociateOperatorFromStaker(&_Delegation.TransactOpts, clientChainID, staker)
}

// Undelegate is a paid mutator transaction binding the contract method 0xdfea719f.
//
// Solidity: function undelegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationTransactor) Undelegate(opts *bind.TransactOpts, clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.contract.Transact(opts, "undelegate", clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}

// Undelegate is a paid mutator transaction binding the contract method 0xdfea719f.
//
// Solidity: function undelegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationSession) Undelegate(clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.Contract.Undelegate(&_Delegation.TransactOpts, clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}

// Undelegate is a paid mutator transaction binding the contract method 0xdfea719f.
//
// Solidity: function undelegate(uint32 clientChainID, uint64 lzNonce, bytes assetsAddress, bytes stakerAddress, bytes operatorAddr, uint256 opAmount) returns(bool success)
func (_Delegation *DelegationTransactorSession) Undelegate(clientChainID uint32, lzNonce uint64, assetsAddress []byte, stakerAddress []byte, operatorAddr []byte, opAmount *big.Int) (*types.Transaction, error) {
	return _Delegation.Contract.Undelegate(&_Delegation.TransactOpts, clientChainID, lzNonce, assetsAddress, stakerAddress, operatorAddr, opAmount)
}
