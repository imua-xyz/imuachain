// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package avs

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

// AvsABI is the input ABI used to generate the binding from.
const AvsABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"}],\"name\":\"AVSDeregistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"}],\"name\":\"AVSRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"}],\"name\":\"AVSUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"taskHash\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"taskID\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"taskResponseHash\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"operatorAddress\",\"type\":\"string\"}],\"name\":\"ChallengeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"}],\"name\":\"OperatorJoined\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"}],\"name\":\"OperatorLeft\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"PublicKeyRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"taskContractAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"taskId\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"hash\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"taskResponsePeriod\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"taskChallengePeriod\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"thresholdPercentage\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"taskStatisticalPeriod\",\"type\":\"uint64\"}],\"name\":\"TaskCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"taskContractAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"taskId\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sender\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"taskResponse\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"blsSignature\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"phase\",\"type\":\"uint8\"}],\"name\":\"TaskSubmittedByOperator\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"taskHash\",\"type\":\"bytes\"},{\"internalType\":\"uint64\",\"name\":\"taskID\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"taskResponseHash\",\"type\":\"bytes\"},{\"internalType\":\"string\",\"name\":\"operatorAddress\",\"type\":\"string\"}],\"name\":\"challenge\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"hash\",\"type\":\"bytes\"},{\"internalType\":\"uint64\",\"name\":\"taskResponsePeriod\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"taskChallengePeriod\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"thresholdPercentage\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"taskStatisticalPeriod\",\"type\":\"uint64\"}],\"name\":\"createTask\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"taskID\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"}],\"name\":\"deregisterAVS\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"deregisterOperatorFromAVS\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"}],\"name\":\"getAVSEpochIdentifier\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"epochIdentifier\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"}],\"name\":\"getAVSUSDValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"epochIdentifier\",\"type\":\"string\"}],\"name\":\"getCurrentEpoch\",\"outputs\":[{\"internalType\":\"int64\",\"name\":\"currentEpoch\",\"type\":\"int64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"avsAddr\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"operatorAddr\",\"type\":\"string\"}],\"name\":\"getOperatorOptedUSDValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"avsAddress\",\"type\":\"address\"}],\"name\":\"getOptInOperators\",\"outputs\":[{\"internalType\":\"string[]\",\"name\":\"operators\",\"type\":\"string[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"operator\",\"type\":\"string\"}],\"name\":\"getRegisteredPubkey\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"taskAddr\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"taskID\",\"type\":\"uint64\"}],\"name\":\"getTaskInfo\",\"outputs\":[{\"internalType\":\"uint64[]\",\"name\":\"info\",\"type\":\"uint64[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operatorAddr\",\"type\":\"address\"}],\"name\":\"isOperator\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"taskID\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"taskResponse\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"blsSignature\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"taskContractAddress\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"phase\",\"type\":\"uint8\"}],\"name\":\"operatorSubmitTask\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"minStakeAmount\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"taskAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"slashAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rewardAddr\",\"type\":\"address\"},{\"internalType\":\"string[]\",\"name\":\"avsOwnerAddress\",\"type\":\"string[]\"},{\"internalType\":\"string[]\",\"name\":\"assetIds\",\"type\":\"string[]\"},{\"internalType\":\"uint64\",\"name\":\"avsUnbondingPeriod\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minSelfDelegation\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"epochIdentifier\",\"type\":\"string\"},{\"internalType\":\"uint64[]\",\"name\":\"params\",\"type\":\"uint64[]\"}],\"name\":\"registerAVS\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"pubKey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"pubkeyRegistrationSignature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"pubkeyRegistrationMessageHash\",\"type\":\"bytes\"}],\"name\":\"registerBLSPublicKey\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"registerOperatorToAVS\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"avsName\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"minStakeAmount\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"taskAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"slashAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rewardAddr\",\"type\":\"address\"},{\"internalType\":\"string[]\",\"name\":\"avsOwnerAddress\",\"type\":\"string[]\"},{\"internalType\":\"string[]\",\"name\":\"assetIds\",\"type\":\"string[]\"},{\"internalType\":\"uint64\",\"name\":\"avsUnbondingPeriod\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minSelfDelegation\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"epochIdentifier\",\"type\":\"string\"},{\"internalType\":\"uint64[]\",\"name\":\"params\",\"type\":\"uint64[]\"}],\"name\":\"updateAVS\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// Avs is an auto generated Go binding around an Ethereum contract.
type Avs struct {
	AvsCaller     // Read-only binding to the contract
	AvsTransactor // Write-only binding to the contract
	AvsFilterer   // Log filterer for contract events
}

// AvsCaller is an auto generated read-only Go binding around an Ethereum contract.
type AvsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AvsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AvsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AvsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AvsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AvsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AvsSession struct {
	Contract     *Avs              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AvsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AvsCallerSession struct {
	Contract *AvsCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// AvsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AvsTransactorSession struct {
	Contract     *AvsTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AvsRaw is an auto generated low-level Go binding around an Ethereum contract.
type AvsRaw struct {
	Contract *Avs // Generic contract binding to access the raw methods on
}

// AvsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AvsCallerRaw struct {
	Contract *AvsCaller // Generic read-only contract binding to access the raw methods on
}

// AvsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AvsTransactorRaw struct {
	Contract *AvsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAvs creates a new instance of Avs, bound to a specific deployed contract.
func NewAvs(address common.Address, backend bind.ContractBackend) (*Avs, error) {
	contract, err := bindAvs(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Avs{AvsCaller: AvsCaller{contract: contract}, AvsTransactor: AvsTransactor{contract: contract}, AvsFilterer: AvsFilterer{contract: contract}}, nil
}

// NewAvsCaller creates a new read-only instance of Avs, bound to a specific deployed contract.
func NewAvsCaller(address common.Address, caller bind.ContractCaller) (*AvsCaller, error) {
	contract, err := bindAvs(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AvsCaller{contract: contract}, nil
}

// NewAvsTransactor creates a new write-only instance of Avs, bound to a specific deployed contract.
func NewAvsTransactor(address common.Address, transactor bind.ContractTransactor) (*AvsTransactor, error) {
	contract, err := bindAvs(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AvsTransactor{contract: contract}, nil
}

// NewAvsFilterer creates a new log filterer instance of Avs, bound to a specific deployed contract.
func NewAvsFilterer(address common.Address, filterer bind.ContractFilterer) (*AvsFilterer, error) {
	contract, err := bindAvs(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AvsFilterer{contract: contract}, nil
}

// bindAvs binds a generic wrapper to an already deployed contract.
func bindAvs(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AvsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Avs *AvsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Avs.Contract.AvsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Avs *AvsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Avs.Contract.AvsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Avs *AvsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Avs.Contract.AvsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Avs *AvsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Avs.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Avs *AvsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Avs.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Avs *AvsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Avs.Contract.contract.Transact(opts, method, params...)
}

// GetAVSEpochIdentifier is a free data retrieval call binding the contract method 0xe0938414.
//
// Solidity: function getAVSEpochIdentifier(address avsAddr) view returns(string epochIdentifier)
func (_Avs *AvsCaller) GetAVSEpochIdentifier(opts *bind.CallOpts, avsAddr common.Address) (string, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getAVSEpochIdentifier", avsAddr)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetAVSEpochIdentifier is a free data retrieval call binding the contract method 0xe0938414.
//
// Solidity: function getAVSEpochIdentifier(address avsAddr) view returns(string epochIdentifier)
func (_Avs *AvsSession) GetAVSEpochIdentifier(avsAddr common.Address) (string, error) {
	return _Avs.Contract.GetAVSEpochIdentifier(&_Avs.CallOpts, avsAddr)
}

// GetAVSEpochIdentifier is a free data retrieval call binding the contract method 0xe0938414.
//
// Solidity: function getAVSEpochIdentifier(address avsAddr) view returns(string epochIdentifier)
func (_Avs *AvsCallerSession) GetAVSEpochIdentifier(avsAddr common.Address) (string, error) {
	return _Avs.Contract.GetAVSEpochIdentifier(&_Avs.CallOpts, avsAddr)
}

// GetAVSUSDValue is a free data retrieval call binding the contract method 0xdcf61b2c.
//
// Solidity: function getAVSUSDValue(address avsAddr) view returns(uint256 amount)
func (_Avs *AvsCaller) GetAVSUSDValue(opts *bind.CallOpts, avsAddr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getAVSUSDValue", avsAddr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAVSUSDValue is a free data retrieval call binding the contract method 0xdcf61b2c.
//
// Solidity: function getAVSUSDValue(address avsAddr) view returns(uint256 amount)
func (_Avs *AvsSession) GetAVSUSDValue(avsAddr common.Address) (*big.Int, error) {
	return _Avs.Contract.GetAVSUSDValue(&_Avs.CallOpts, avsAddr)
}

// GetAVSUSDValue is a free data retrieval call binding the contract method 0xdcf61b2c.
//
// Solidity: function getAVSUSDValue(address avsAddr) view returns(uint256 amount)
func (_Avs *AvsCallerSession) GetAVSUSDValue(avsAddr common.Address) (*big.Int, error) {
	return _Avs.Contract.GetAVSUSDValue(&_Avs.CallOpts, avsAddr)
}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0x992907fb.
//
// Solidity: function getCurrentEpoch(string epochIdentifier) view returns(int64 currentEpoch)
func (_Avs *AvsCaller) GetCurrentEpoch(opts *bind.CallOpts, epochIdentifier string) (int64, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getCurrentEpoch", epochIdentifier)

	if err != nil {
		return *new(int64), err
	}

	out0 := *abi.ConvertType(out[0], new(int64)).(*int64)

	return out0, err

}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0x992907fb.
//
// Solidity: function getCurrentEpoch(string epochIdentifier) view returns(int64 currentEpoch)
func (_Avs *AvsSession) GetCurrentEpoch(epochIdentifier string) (int64, error) {
	return _Avs.Contract.GetCurrentEpoch(&_Avs.CallOpts, epochIdentifier)
}

// GetCurrentEpoch is a free data retrieval call binding the contract method 0x992907fb.
//
// Solidity: function getCurrentEpoch(string epochIdentifier) view returns(int64 currentEpoch)
func (_Avs *AvsCallerSession) GetCurrentEpoch(epochIdentifier string) (int64, error) {
	return _Avs.Contract.GetCurrentEpoch(&_Avs.CallOpts, epochIdentifier)
}

// GetOperatorOptedUSDValue is a free data retrieval call binding the contract method 0x2d9d6a20.
//
// Solidity: function getOperatorOptedUSDValue(address avsAddr, string operatorAddr) view returns(uint256 amount)
func (_Avs *AvsCaller) GetOperatorOptedUSDValue(opts *bind.CallOpts, avsAddr common.Address, operatorAddr string) (*big.Int, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getOperatorOptedUSDValue", avsAddr, operatorAddr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetOperatorOptedUSDValue is a free data retrieval call binding the contract method 0x2d9d6a20.
//
// Solidity: function getOperatorOptedUSDValue(address avsAddr, string operatorAddr) view returns(uint256 amount)
func (_Avs *AvsSession) GetOperatorOptedUSDValue(avsAddr common.Address, operatorAddr string) (*big.Int, error) {
	return _Avs.Contract.GetOperatorOptedUSDValue(&_Avs.CallOpts, avsAddr, operatorAddr)
}

// GetOperatorOptedUSDValue is a free data retrieval call binding the contract method 0x2d9d6a20.
//
// Solidity: function getOperatorOptedUSDValue(address avsAddr, string operatorAddr) view returns(uint256 amount)
func (_Avs *AvsCallerSession) GetOperatorOptedUSDValue(avsAddr common.Address, operatorAddr string) (*big.Int, error) {
	return _Avs.Contract.GetOperatorOptedUSDValue(&_Avs.CallOpts, avsAddr, operatorAddr)
}

// GetOptInOperators is a free data retrieval call binding the contract method 0x1d4c8007.
//
// Solidity: function getOptInOperators(address avsAddress) view returns(string[] operators)
func (_Avs *AvsCaller) GetOptInOperators(opts *bind.CallOpts, avsAddress common.Address) ([]string, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getOptInOperators", avsAddress)

	if err != nil {
		return *new([]string), err
	}

	out0 := *abi.ConvertType(out[0], new([]string)).(*[]string)

	return out0, err

}

// GetOptInOperators is a free data retrieval call binding the contract method 0x1d4c8007.
//
// Solidity: function getOptInOperators(address avsAddress) view returns(string[] operators)
func (_Avs *AvsSession) GetOptInOperators(avsAddress common.Address) ([]string, error) {
	return _Avs.Contract.GetOptInOperators(&_Avs.CallOpts, avsAddress)
}

// GetOptInOperators is a free data retrieval call binding the contract method 0x1d4c8007.
//
// Solidity: function getOptInOperators(address avsAddress) view returns(string[] operators)
func (_Avs *AvsCallerSession) GetOptInOperators(avsAddress common.Address) ([]string, error) {
	return _Avs.Contract.GetOptInOperators(&_Avs.CallOpts, avsAddress)
}

// GetRegisteredPubkey is a free data retrieval call binding the contract method 0x55b42cbe.
//
// Solidity: function getRegisteredPubkey(string operator) view returns(bytes pubkey)
func (_Avs *AvsCaller) GetRegisteredPubkey(opts *bind.CallOpts, operator string) ([]byte, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getRegisteredPubkey", operator)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetRegisteredPubkey is a free data retrieval call binding the contract method 0x55b42cbe.
//
// Solidity: function getRegisteredPubkey(string operator) view returns(bytes pubkey)
func (_Avs *AvsSession) GetRegisteredPubkey(operator string) ([]byte, error) {
	return _Avs.Contract.GetRegisteredPubkey(&_Avs.CallOpts, operator)
}

// GetRegisteredPubkey is a free data retrieval call binding the contract method 0x55b42cbe.
//
// Solidity: function getRegisteredPubkey(string operator) view returns(bytes pubkey)
func (_Avs *AvsCallerSession) GetRegisteredPubkey(operator string) ([]byte, error) {
	return _Avs.Contract.GetRegisteredPubkey(&_Avs.CallOpts, operator)
}

// GetTaskInfo is a free data retrieval call binding the contract method 0xe2906f3d.
//
// Solidity: function getTaskInfo(address taskAddr, uint64 taskID) view returns(uint64[] info)
func (_Avs *AvsCaller) GetTaskInfo(opts *bind.CallOpts, taskAddr common.Address, taskID uint64) ([]uint64, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "getTaskInfo", taskAddr, taskID)

	if err != nil {
		return *new([]uint64), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint64)).(*[]uint64)

	return out0, err

}

// GetTaskInfo is a free data retrieval call binding the contract method 0xe2906f3d.
//
// Solidity: function getTaskInfo(address taskAddr, uint64 taskID) view returns(uint64[] info)
func (_Avs *AvsSession) GetTaskInfo(taskAddr common.Address, taskID uint64) ([]uint64, error) {
	return _Avs.Contract.GetTaskInfo(&_Avs.CallOpts, taskAddr, taskID)
}

// GetTaskInfo is a free data retrieval call binding the contract method 0xe2906f3d.
//
// Solidity: function getTaskInfo(address taskAddr, uint64 taskID) view returns(uint64[] info)
func (_Avs *AvsCallerSession) GetTaskInfo(taskAddr common.Address, taskID uint64) ([]uint64, error) {
	return _Avs.Contract.GetTaskInfo(&_Avs.CallOpts, taskAddr, taskID)
}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address operatorAddr) view returns(bool)
func (_Avs *AvsCaller) IsOperator(opts *bind.CallOpts, operatorAddr common.Address) (bool, error) {
	var out []interface{}
	err := _Avs.contract.Call(opts, &out, "isOperator", operatorAddr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address operatorAddr) view returns(bool)
func (_Avs *AvsSession) IsOperator(operatorAddr common.Address) (bool, error) {
	return _Avs.Contract.IsOperator(&_Avs.CallOpts, operatorAddr)
}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address operatorAddr) view returns(bool)
func (_Avs *AvsCallerSession) IsOperator(operatorAddr common.Address) (bool, error) {
	return _Avs.Contract.IsOperator(&_Avs.CallOpts, operatorAddr)
}

// Challenge is a paid mutator transaction binding the contract method 0xe4465980.
//
// Solidity: function challenge(address sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress) returns(bool success)
func (_Avs *AvsTransactor) Challenge(opts *bind.TransactOpts, sender common.Address, taskHash []byte, taskID uint64, taskResponseHash []byte, operatorAddress string) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "challenge", sender, taskHash, taskID, taskResponseHash, operatorAddress)
}

// Challenge is a paid mutator transaction binding the contract method 0xe4465980.
//
// Solidity: function challenge(address sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress) returns(bool success)
func (_Avs *AvsSession) Challenge(sender common.Address, taskHash []byte, taskID uint64, taskResponseHash []byte, operatorAddress string) (*types.Transaction, error) {
	return _Avs.Contract.Challenge(&_Avs.TransactOpts, sender, taskHash, taskID, taskResponseHash, operatorAddress)
}

// Challenge is a paid mutator transaction binding the contract method 0xe4465980.
//
// Solidity: function challenge(address sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress) returns(bool success)
func (_Avs *AvsTransactorSession) Challenge(sender common.Address, taskHash []byte, taskID uint64, taskResponseHash []byte, operatorAddress string) (*types.Transaction, error) {
	return _Avs.Contract.Challenge(&_Avs.TransactOpts, sender, taskHash, taskID, taskResponseHash, operatorAddress)
}

// CreateTask is a paid mutator transaction binding the contract method 0x8bf30a69.
//
// Solidity: function createTask(address sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod) returns(uint64 taskID)
func (_Avs *AvsTransactor) CreateTask(opts *bind.TransactOpts, sender common.Address, name string, hash []byte, taskResponsePeriod uint64, taskChallengePeriod uint64, thresholdPercentage uint64, taskStatisticalPeriod uint64) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "createTask", sender, name, hash, taskResponsePeriod, taskChallengePeriod, thresholdPercentage, taskStatisticalPeriod)
}

// CreateTask is a paid mutator transaction binding the contract method 0x8bf30a69.
//
// Solidity: function createTask(address sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod) returns(uint64 taskID)
func (_Avs *AvsSession) CreateTask(sender common.Address, name string, hash []byte, taskResponsePeriod uint64, taskChallengePeriod uint64, thresholdPercentage uint64, taskStatisticalPeriod uint64) (*types.Transaction, error) {
	return _Avs.Contract.CreateTask(&_Avs.TransactOpts, sender, name, hash, taskResponsePeriod, taskChallengePeriod, thresholdPercentage, taskStatisticalPeriod)
}

// CreateTask is a paid mutator transaction binding the contract method 0x8bf30a69.
//
// Solidity: function createTask(address sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod) returns(uint64 taskID)
func (_Avs *AvsTransactorSession) CreateTask(sender common.Address, name string, hash []byte, taskResponsePeriod uint64, taskChallengePeriod uint64, thresholdPercentage uint64, taskStatisticalPeriod uint64) (*types.Transaction, error) {
	return _Avs.Contract.CreateTask(&_Avs.TransactOpts, sender, name, hash, taskResponsePeriod, taskChallengePeriod, thresholdPercentage, taskStatisticalPeriod)
}

// DeregisterAVS is a paid mutator transaction binding the contract method 0x18cd2ab3.
//
// Solidity: function deregisterAVS(address sender, string avsName) returns(bool success)
func (_Avs *AvsTransactor) DeregisterAVS(opts *bind.TransactOpts, sender common.Address, avsName string) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "deregisterAVS", sender, avsName)
}

// DeregisterAVS is a paid mutator transaction binding the contract method 0x18cd2ab3.
//
// Solidity: function deregisterAVS(address sender, string avsName) returns(bool success)
func (_Avs *AvsSession) DeregisterAVS(sender common.Address, avsName string) (*types.Transaction, error) {
	return _Avs.Contract.DeregisterAVS(&_Avs.TransactOpts, sender, avsName)
}

// DeregisterAVS is a paid mutator transaction binding the contract method 0x18cd2ab3.
//
// Solidity: function deregisterAVS(address sender, string avsName) returns(bool success)
func (_Avs *AvsTransactorSession) DeregisterAVS(sender common.Address, avsName string) (*types.Transaction, error) {
	return _Avs.Contract.DeregisterAVS(&_Avs.TransactOpts, sender, avsName)
}

// DeregisterOperatorFromAVS is a paid mutator transaction binding the contract method 0xa364f4da.
//
// Solidity: function deregisterOperatorFromAVS(address sender) returns(bool success)
func (_Avs *AvsTransactor) DeregisterOperatorFromAVS(opts *bind.TransactOpts, sender common.Address) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "deregisterOperatorFromAVS", sender)
}

// DeregisterOperatorFromAVS is a paid mutator transaction binding the contract method 0xa364f4da.
//
// Solidity: function deregisterOperatorFromAVS(address sender) returns(bool success)
func (_Avs *AvsSession) DeregisterOperatorFromAVS(sender common.Address) (*types.Transaction, error) {
	return _Avs.Contract.DeregisterOperatorFromAVS(&_Avs.TransactOpts, sender)
}

// DeregisterOperatorFromAVS is a paid mutator transaction binding the contract method 0xa364f4da.
//
// Solidity: function deregisterOperatorFromAVS(address sender) returns(bool success)
func (_Avs *AvsTransactorSession) DeregisterOperatorFromAVS(sender common.Address) (*types.Transaction, error) {
	return _Avs.Contract.DeregisterOperatorFromAVS(&_Avs.TransactOpts, sender)
}

// OperatorSubmitTask is a paid mutator transaction binding the contract method 0x08da2762.
//
// Solidity: function operatorSubmitTask(address sender, uint64 taskID, bytes taskResponse, bytes blsSignature, address taskContractAddress, uint8 phase) returns(bool success)
func (_Avs *AvsTransactor) OperatorSubmitTask(opts *bind.TransactOpts, sender common.Address, taskID uint64, taskResponse []byte, blsSignature []byte, taskContractAddress common.Address, phase uint8) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "operatorSubmitTask", sender, taskID, taskResponse, blsSignature, taskContractAddress, phase)
}

// OperatorSubmitTask is a paid mutator transaction binding the contract method 0x08da2762.
//
// Solidity: function operatorSubmitTask(address sender, uint64 taskID, bytes taskResponse, bytes blsSignature, address taskContractAddress, uint8 phase) returns(bool success)
func (_Avs *AvsSession) OperatorSubmitTask(sender common.Address, taskID uint64, taskResponse []byte, blsSignature []byte, taskContractAddress common.Address, phase uint8) (*types.Transaction, error) {
	return _Avs.Contract.OperatorSubmitTask(&_Avs.TransactOpts, sender, taskID, taskResponse, blsSignature, taskContractAddress, phase)
}

// OperatorSubmitTask is a paid mutator transaction binding the contract method 0x08da2762.
//
// Solidity: function operatorSubmitTask(address sender, uint64 taskID, bytes taskResponse, bytes blsSignature, address taskContractAddress, uint8 phase) returns(bool success)
func (_Avs *AvsTransactorSession) OperatorSubmitTask(sender common.Address, taskID uint64, taskResponse []byte, blsSignature []byte, taskContractAddress common.Address, phase uint8) (*types.Transaction, error) {
	return _Avs.Contract.OperatorSubmitTask(&_Avs.TransactOpts, sender, taskID, taskResponse, blsSignature, taskContractAddress, phase)
}

// RegisterAVS is a paid mutator transaction binding the contract method 0xcde09950.
//
// Solidity: function registerAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsTransactor) RegisterAVS(opts *bind.TransactOpts, sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "registerAVS", sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// RegisterAVS is a paid mutator transaction binding the contract method 0xcde09950.
//
// Solidity: function registerAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsSession) RegisterAVS(sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.Contract.RegisterAVS(&_Avs.TransactOpts, sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// RegisterAVS is a paid mutator transaction binding the contract method 0xcde09950.
//
// Solidity: function registerAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsTransactorSession) RegisterAVS(sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.Contract.RegisterAVS(&_Avs.TransactOpts, sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// RegisterBLSPublicKey is a paid mutator transaction binding the contract method 0xaa57f77c.
//
// Solidity: function registerBLSPublicKey(address sender, string name, bytes pubKey, bytes pubkeyRegistrationSignature, bytes pubkeyRegistrationMessageHash) returns(bool success)
func (_Avs *AvsTransactor) RegisterBLSPublicKey(opts *bind.TransactOpts, sender common.Address, name string, pubKey []byte, pubkeyRegistrationSignature []byte, pubkeyRegistrationMessageHash []byte) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "registerBLSPublicKey", sender, name, pubKey, pubkeyRegistrationSignature, pubkeyRegistrationMessageHash)
}

// RegisterBLSPublicKey is a paid mutator transaction binding the contract method 0xaa57f77c.
//
// Solidity: function registerBLSPublicKey(address sender, string name, bytes pubKey, bytes pubkeyRegistrationSignature, bytes pubkeyRegistrationMessageHash) returns(bool success)
func (_Avs *AvsSession) RegisterBLSPublicKey(sender common.Address, name string, pubKey []byte, pubkeyRegistrationSignature []byte, pubkeyRegistrationMessageHash []byte) (*types.Transaction, error) {
	return _Avs.Contract.RegisterBLSPublicKey(&_Avs.TransactOpts, sender, name, pubKey, pubkeyRegistrationSignature, pubkeyRegistrationMessageHash)
}

// RegisterBLSPublicKey is a paid mutator transaction binding the contract method 0xaa57f77c.
//
// Solidity: function registerBLSPublicKey(address sender, string name, bytes pubKey, bytes pubkeyRegistrationSignature, bytes pubkeyRegistrationMessageHash) returns(bool success)
func (_Avs *AvsTransactorSession) RegisterBLSPublicKey(sender common.Address, name string, pubKey []byte, pubkeyRegistrationSignature []byte, pubkeyRegistrationMessageHash []byte) (*types.Transaction, error) {
	return _Avs.Contract.RegisterBLSPublicKey(&_Avs.TransactOpts, sender, name, pubKey, pubkeyRegistrationSignature, pubkeyRegistrationMessageHash)
}

// RegisterOperatorToAVS is a paid mutator transaction binding the contract method 0xd7a2398b.
//
// Solidity: function registerOperatorToAVS(address sender) returns(bool success)
func (_Avs *AvsTransactor) RegisterOperatorToAVS(opts *bind.TransactOpts, sender common.Address) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "registerOperatorToAVS", sender)
}

// RegisterOperatorToAVS is a paid mutator transaction binding the contract method 0xd7a2398b.
//
// Solidity: function registerOperatorToAVS(address sender) returns(bool success)
func (_Avs *AvsSession) RegisterOperatorToAVS(sender common.Address) (*types.Transaction, error) {
	return _Avs.Contract.RegisterOperatorToAVS(&_Avs.TransactOpts, sender)
}

// RegisterOperatorToAVS is a paid mutator transaction binding the contract method 0xd7a2398b.
//
// Solidity: function registerOperatorToAVS(address sender) returns(bool success)
func (_Avs *AvsTransactorSession) RegisterOperatorToAVS(sender common.Address) (*types.Transaction, error) {
	return _Avs.Contract.RegisterOperatorToAVS(&_Avs.TransactOpts, sender)
}

// UpdateAVS is a paid mutator transaction binding the contract method 0xd9e5daa0.
//
// Solidity: function updateAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsTransactor) UpdateAVS(opts *bind.TransactOpts, sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.contract.Transact(opts, "updateAVS", sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// UpdateAVS is a paid mutator transaction binding the contract method 0xd9e5daa0.
//
// Solidity: function updateAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsSession) UpdateAVS(sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.Contract.UpdateAVS(&_Avs.TransactOpts, sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// UpdateAVS is a paid mutator transaction binding the contract method 0xd9e5daa0.
//
// Solidity: function updateAVS(address sender, string avsName, uint64 minStakeAmount, address taskAddr, address slashAddr, address rewardAddr, string[] avsOwnerAddress, string[] assetIds, uint64 avsUnbondingPeriod, uint64 minSelfDelegation, string epochIdentifier, uint64[] params) returns(bool success)
func (_Avs *AvsTransactorSession) UpdateAVS(sender common.Address, avsName string, minStakeAmount uint64, taskAddr common.Address, slashAddr common.Address, rewardAddr common.Address, avsOwnerAddress []string, assetIds []string, avsUnbondingPeriod uint64, minSelfDelegation uint64, epochIdentifier string, params []uint64) (*types.Transaction, error) {
	return _Avs.Contract.UpdateAVS(&_Avs.TransactOpts, sender, avsName, minStakeAmount, taskAddr, slashAddr, rewardAddr, avsOwnerAddress, assetIds, avsUnbondingPeriod, minSelfDelegation, epochIdentifier, params)
}

// AvsAVSDeregisteredIterator is returned from FilterAVSDeregistered and is used to iterate over the raw logs and unpacked data for AVSDeregistered events raised by the Avs contract.
type AvsAVSDeregisteredIterator struct {
	Event *AvsAVSDeregistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsAVSDeregisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsAVSDeregistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsAVSDeregistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsAVSDeregisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsAVSDeregisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsAVSDeregistered represents a AVSDeregistered event raised by the Avs contract.
type AvsAVSDeregistered struct {
	AvsAddr common.Address
	Sender  string
	AvsName string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAVSDeregistered is a free log retrieval operation binding the contract event 0x67bc7cc9901ce497339884997842255d82165981b1600f96dfa7db090e177a22.
//
// Solidity: event AVSDeregistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) FilterAVSDeregistered(opts *bind.FilterOpts, avsAddr []common.Address) (*AvsAVSDeregisteredIterator, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "AVSDeregistered", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return &AvsAVSDeregisteredIterator{contract: _Avs.contract, event: "AVSDeregistered", logs: logs, sub: sub}, nil
}

// WatchAVSDeregistered is a free log subscription operation binding the contract event 0x67bc7cc9901ce497339884997842255d82165981b1600f96dfa7db090e177a22.
//
// Solidity: event AVSDeregistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) WatchAVSDeregistered(opts *bind.WatchOpts, sink chan<- *AvsAVSDeregistered, avsAddr []common.Address) (event.Subscription, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "AVSDeregistered", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsAVSDeregistered)
				if err := _Avs.contract.UnpackLog(event, "AVSDeregistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAVSDeregistered is a log parse operation binding the contract event 0x67bc7cc9901ce497339884997842255d82165981b1600f96dfa7db090e177a22.
//
// Solidity: event AVSDeregistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) ParseAVSDeregistered(log types.Log) (*AvsAVSDeregistered, error) {
	event := new(AvsAVSDeregistered)
	if err := _Avs.contract.UnpackLog(event, "AVSDeregistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsAVSRegisteredIterator is returned from FilterAVSRegistered and is used to iterate over the raw logs and unpacked data for AVSRegistered events raised by the Avs contract.
type AvsAVSRegisteredIterator struct {
	Event *AvsAVSRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsAVSRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsAVSRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsAVSRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsAVSRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsAVSRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsAVSRegistered represents a AVSRegistered event raised by the Avs contract.
type AvsAVSRegistered struct {
	AvsAddr common.Address
	Sender  string
	AvsName string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAVSRegistered is a free log retrieval operation binding the contract event 0x1b6757d434d8bec2e67cb54f5f69444370fbe7278eac51697934bc56f1eab4de.
//
// Solidity: event AVSRegistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) FilterAVSRegistered(opts *bind.FilterOpts, avsAddr []common.Address) (*AvsAVSRegisteredIterator, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "AVSRegistered", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return &AvsAVSRegisteredIterator{contract: _Avs.contract, event: "AVSRegistered", logs: logs, sub: sub}, nil
}

// WatchAVSRegistered is a free log subscription operation binding the contract event 0x1b6757d434d8bec2e67cb54f5f69444370fbe7278eac51697934bc56f1eab4de.
//
// Solidity: event AVSRegistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) WatchAVSRegistered(opts *bind.WatchOpts, sink chan<- *AvsAVSRegistered, avsAddr []common.Address) (event.Subscription, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "AVSRegistered", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsAVSRegistered)
				if err := _Avs.contract.UnpackLog(event, "AVSRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAVSRegistered is a log parse operation binding the contract event 0x1b6757d434d8bec2e67cb54f5f69444370fbe7278eac51697934bc56f1eab4de.
//
// Solidity: event AVSRegistered(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) ParseAVSRegistered(log types.Log) (*AvsAVSRegistered, error) {
	event := new(AvsAVSRegistered)
	if err := _Avs.contract.UnpackLog(event, "AVSRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsAVSUpdatedIterator is returned from FilterAVSUpdated and is used to iterate over the raw logs and unpacked data for AVSUpdated events raised by the Avs contract.
type AvsAVSUpdatedIterator struct {
	Event *AvsAVSUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsAVSUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsAVSUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsAVSUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsAVSUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsAVSUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsAVSUpdated represents a AVSUpdated event raised by the Avs contract.
type AvsAVSUpdated struct {
	AvsAddr common.Address
	Sender  string
	AvsName string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAVSUpdated is a free log retrieval operation binding the contract event 0xa2b66ea3e16df099ef9e6e0631e070f6da12910dd63de9f37904dfaea6356210.
//
// Solidity: event AVSUpdated(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) FilterAVSUpdated(opts *bind.FilterOpts, avsAddr []common.Address) (*AvsAVSUpdatedIterator, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "AVSUpdated", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return &AvsAVSUpdatedIterator{contract: _Avs.contract, event: "AVSUpdated", logs: logs, sub: sub}, nil
}

// WatchAVSUpdated is a free log subscription operation binding the contract event 0xa2b66ea3e16df099ef9e6e0631e070f6da12910dd63de9f37904dfaea6356210.
//
// Solidity: event AVSUpdated(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) WatchAVSUpdated(opts *bind.WatchOpts, sink chan<- *AvsAVSUpdated, avsAddr []common.Address) (event.Subscription, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "AVSUpdated", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsAVSUpdated)
				if err := _Avs.contract.UnpackLog(event, "AVSUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAVSUpdated is a log parse operation binding the contract event 0xa2b66ea3e16df099ef9e6e0631e070f6da12910dd63de9f37904dfaea6356210.
//
// Solidity: event AVSUpdated(address indexed avsAddr, string sender, string avsName)
func (_Avs *AvsFilterer) ParseAVSUpdated(log types.Log) (*AvsAVSUpdated, error) {
	event := new(AvsAVSUpdated)
	if err := _Avs.contract.UnpackLog(event, "AVSUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsChallengeInitiatedIterator is returned from FilterChallengeInitiated and is used to iterate over the raw logs and unpacked data for ChallengeInitiated events raised by the Avs contract.
type AvsChallengeInitiatedIterator struct {
	Event *AvsChallengeInitiated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsChallengeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsChallengeInitiated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsChallengeInitiated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsChallengeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsChallengeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsChallengeInitiated represents a ChallengeInitiated event raised by the Avs contract.
type AvsChallengeInitiated struct {
	Sender           string
	TaskHash         []byte
	TaskID           uint64
	TaskResponseHash []byte
	OperatorAddress  string
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterChallengeInitiated is a free log retrieval operation binding the contract event 0x01e63a7a029b771066771d185dc79bdf1457c962df4719b8217fc11becfd8fbc.
//
// Solidity: event ChallengeInitiated(string sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress)
func (_Avs *AvsFilterer) FilterChallengeInitiated(opts *bind.FilterOpts) (*AvsChallengeInitiatedIterator, error) {

	logs, sub, err := _Avs.contract.FilterLogs(opts, "ChallengeInitiated")
	if err != nil {
		return nil, err
	}
	return &AvsChallengeInitiatedIterator{contract: _Avs.contract, event: "ChallengeInitiated", logs: logs, sub: sub}, nil
}

// WatchChallengeInitiated is a free log subscription operation binding the contract event 0x01e63a7a029b771066771d185dc79bdf1457c962df4719b8217fc11becfd8fbc.
//
// Solidity: event ChallengeInitiated(string sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress)
func (_Avs *AvsFilterer) WatchChallengeInitiated(opts *bind.WatchOpts, sink chan<- *AvsChallengeInitiated) (event.Subscription, error) {

	logs, sub, err := _Avs.contract.WatchLogs(opts, "ChallengeInitiated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsChallengeInitiated)
				if err := _Avs.contract.UnpackLog(event, "ChallengeInitiated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseChallengeInitiated is a log parse operation binding the contract event 0x01e63a7a029b771066771d185dc79bdf1457c962df4719b8217fc11becfd8fbc.
//
// Solidity: event ChallengeInitiated(string sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress)
func (_Avs *AvsFilterer) ParseChallengeInitiated(log types.Log) (*AvsChallengeInitiated, error) {
	event := new(AvsChallengeInitiated)
	if err := _Avs.contract.UnpackLog(event, "ChallengeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsOperatorJoinedIterator is returned from FilterOperatorJoined and is used to iterate over the raw logs and unpacked data for OperatorJoined events raised by the Avs contract.
type AvsOperatorJoinedIterator struct {
	Event *AvsOperatorJoined // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsOperatorJoinedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsOperatorJoined)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsOperatorJoined)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsOperatorJoinedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsOperatorJoinedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsOperatorJoined represents a OperatorJoined event raised by the Avs contract.
type AvsOperatorJoined struct {
	AvsAddr common.Address
	Sender  string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOperatorJoined is a free log retrieval operation binding the contract event 0x5a81c8825b5902c1768a558eeb0bf35840fc04eed5399066ca4f6ecd74bc5d6d.
//
// Solidity: event OperatorJoined(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) FilterOperatorJoined(opts *bind.FilterOpts, avsAddr []common.Address) (*AvsOperatorJoinedIterator, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "OperatorJoined", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return &AvsOperatorJoinedIterator{contract: _Avs.contract, event: "OperatorJoined", logs: logs, sub: sub}, nil
}

// WatchOperatorJoined is a free log subscription operation binding the contract event 0x5a81c8825b5902c1768a558eeb0bf35840fc04eed5399066ca4f6ecd74bc5d6d.
//
// Solidity: event OperatorJoined(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) WatchOperatorJoined(opts *bind.WatchOpts, sink chan<- *AvsOperatorJoined, avsAddr []common.Address) (event.Subscription, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "OperatorJoined", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsOperatorJoined)
				if err := _Avs.contract.UnpackLog(event, "OperatorJoined", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOperatorJoined is a log parse operation binding the contract event 0x5a81c8825b5902c1768a558eeb0bf35840fc04eed5399066ca4f6ecd74bc5d6d.
//
// Solidity: event OperatorJoined(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) ParseOperatorJoined(log types.Log) (*AvsOperatorJoined, error) {
	event := new(AvsOperatorJoined)
	if err := _Avs.contract.UnpackLog(event, "OperatorJoined", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsOperatorLeftIterator is returned from FilterOperatorLeft and is used to iterate over the raw logs and unpacked data for OperatorLeft events raised by the Avs contract.
type AvsOperatorLeftIterator struct {
	Event *AvsOperatorLeft // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsOperatorLeftIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsOperatorLeft)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsOperatorLeft)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsOperatorLeftIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsOperatorLeftIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsOperatorLeft represents a OperatorLeft event raised by the Avs contract.
type AvsOperatorLeft struct {
	AvsAddr common.Address
	Sender  string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOperatorLeft is a free log retrieval operation binding the contract event 0x277a77a79a4bd668d43eb072d5b492880a7712d8ef4152c224e3f3846c26ab72.
//
// Solidity: event OperatorLeft(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) FilterOperatorLeft(opts *bind.FilterOpts, avsAddr []common.Address) (*AvsOperatorLeftIterator, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "OperatorLeft", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return &AvsOperatorLeftIterator{contract: _Avs.contract, event: "OperatorLeft", logs: logs, sub: sub}, nil
}

// WatchOperatorLeft is a free log subscription operation binding the contract event 0x277a77a79a4bd668d43eb072d5b492880a7712d8ef4152c224e3f3846c26ab72.
//
// Solidity: event OperatorLeft(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) WatchOperatorLeft(opts *bind.WatchOpts, sink chan<- *AvsOperatorLeft, avsAddr []common.Address) (event.Subscription, error) {

	var avsAddrRule []interface{}
	for _, avsAddrItem := range avsAddr {
		avsAddrRule = append(avsAddrRule, avsAddrItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "OperatorLeft", avsAddrRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsOperatorLeft)
				if err := _Avs.contract.UnpackLog(event, "OperatorLeft", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOperatorLeft is a log parse operation binding the contract event 0x277a77a79a4bd668d43eb072d5b492880a7712d8ef4152c224e3f3846c26ab72.
//
// Solidity: event OperatorLeft(address indexed avsAddr, string sender)
func (_Avs *AvsFilterer) ParseOperatorLeft(log types.Log) (*AvsOperatorLeft, error) {
	event := new(AvsOperatorLeft)
	if err := _Avs.contract.UnpackLog(event, "OperatorLeft", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsPublicKeyRegisteredIterator is returned from FilterPublicKeyRegistered and is used to iterate over the raw logs and unpacked data for PublicKeyRegistered events raised by the Avs contract.
type AvsPublicKeyRegisteredIterator struct {
	Event *AvsPublicKeyRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsPublicKeyRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsPublicKeyRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsPublicKeyRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsPublicKeyRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsPublicKeyRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsPublicKeyRegistered represents a PublicKeyRegistered event raised by the Avs contract.
type AvsPublicKeyRegistered struct {
	Sender string
	Name   string
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterPublicKeyRegistered is a free log retrieval operation binding the contract event 0xe6654ec72a4f93a677214779c6cc7752f1a68242588010c0ef0a810800e7ab55.
//
// Solidity: event PublicKeyRegistered(string sender, string name)
func (_Avs *AvsFilterer) FilterPublicKeyRegistered(opts *bind.FilterOpts) (*AvsPublicKeyRegisteredIterator, error) {

	logs, sub, err := _Avs.contract.FilterLogs(opts, "PublicKeyRegistered")
	if err != nil {
		return nil, err
	}
	return &AvsPublicKeyRegisteredIterator{contract: _Avs.contract, event: "PublicKeyRegistered", logs: logs, sub: sub}, nil
}

// WatchPublicKeyRegistered is a free log subscription operation binding the contract event 0xe6654ec72a4f93a677214779c6cc7752f1a68242588010c0ef0a810800e7ab55.
//
// Solidity: event PublicKeyRegistered(string sender, string name)
func (_Avs *AvsFilterer) WatchPublicKeyRegistered(opts *bind.WatchOpts, sink chan<- *AvsPublicKeyRegistered) (event.Subscription, error) {

	logs, sub, err := _Avs.contract.WatchLogs(opts, "PublicKeyRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsPublicKeyRegistered)
				if err := _Avs.contract.UnpackLog(event, "PublicKeyRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePublicKeyRegistered is a log parse operation binding the contract event 0xe6654ec72a4f93a677214779c6cc7752f1a68242588010c0ef0a810800e7ab55.
//
// Solidity: event PublicKeyRegistered(string sender, string name)
func (_Avs *AvsFilterer) ParsePublicKeyRegistered(log types.Log) (*AvsPublicKeyRegistered, error) {
	event := new(AvsPublicKeyRegistered)
	if err := _Avs.contract.UnpackLog(event, "PublicKeyRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsTaskCreatedIterator is returned from FilterTaskCreated and is used to iterate over the raw logs and unpacked data for TaskCreated events raised by the Avs contract.
type AvsTaskCreatedIterator struct {
	Event *AvsTaskCreated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsTaskCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsTaskCreated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsTaskCreated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsTaskCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsTaskCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsTaskCreated represents a TaskCreated event raised by the Avs contract.
type AvsTaskCreated struct {
	TaskContractAddress   common.Address
	TaskId                uint64
	Sender                string
	Name                  string
	Hash                  []byte
	TaskResponsePeriod    uint64
	TaskChallengePeriod   uint64
	ThresholdPercentage   uint64
	TaskStatisticalPeriod uint64
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterTaskCreated is a free log retrieval operation binding the contract event 0x4be522b3d07a47995b5e029698504adf28cb3baa503224889d88731b22df9b97.
//
// Solidity: event TaskCreated(address indexed taskContractAddress, uint64 indexed taskId, string sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod)
func (_Avs *AvsFilterer) FilterTaskCreated(opts *bind.FilterOpts, taskContractAddress []common.Address, taskId []uint64) (*AvsTaskCreatedIterator, error) {

	var taskContractAddressRule []interface{}
	for _, taskContractAddressItem := range taskContractAddress {
		taskContractAddressRule = append(taskContractAddressRule, taskContractAddressItem)
	}
	var taskIdRule []interface{}
	for _, taskIdItem := range taskId {
		taskIdRule = append(taskIdRule, taskIdItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "TaskCreated", taskContractAddressRule, taskIdRule)
	if err != nil {
		return nil, err
	}
	return &AvsTaskCreatedIterator{contract: _Avs.contract, event: "TaskCreated", logs: logs, sub: sub}, nil
}

// WatchTaskCreated is a free log subscription operation binding the contract event 0x4be522b3d07a47995b5e029698504adf28cb3baa503224889d88731b22df9b97.
//
// Solidity: event TaskCreated(address indexed taskContractAddress, uint64 indexed taskId, string sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod)
func (_Avs *AvsFilterer) WatchTaskCreated(opts *bind.WatchOpts, sink chan<- *AvsTaskCreated, taskContractAddress []common.Address, taskId []uint64) (event.Subscription, error) {

	var taskContractAddressRule []interface{}
	for _, taskContractAddressItem := range taskContractAddress {
		taskContractAddressRule = append(taskContractAddressRule, taskContractAddressItem)
	}
	var taskIdRule []interface{}
	for _, taskIdItem := range taskId {
		taskIdRule = append(taskIdRule, taskIdItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "TaskCreated", taskContractAddressRule, taskIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsTaskCreated)
				if err := _Avs.contract.UnpackLog(event, "TaskCreated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTaskCreated is a log parse operation binding the contract event 0x4be522b3d07a47995b5e029698504adf28cb3baa503224889d88731b22df9b97.
//
// Solidity: event TaskCreated(address indexed taskContractAddress, uint64 indexed taskId, string sender, string name, bytes hash, uint64 taskResponsePeriod, uint64 taskChallengePeriod, uint64 thresholdPercentage, uint64 taskStatisticalPeriod)
func (_Avs *AvsFilterer) ParseTaskCreated(log types.Log) (*AvsTaskCreated, error) {
	event := new(AvsTaskCreated)
	if err := _Avs.contract.UnpackLog(event, "TaskCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AvsTaskSubmittedByOperatorIterator is returned from FilterTaskSubmittedByOperator and is used to iterate over the raw logs and unpacked data for TaskSubmittedByOperator events raised by the Avs contract.
type AvsTaskSubmittedByOperatorIterator struct {
	Event *AvsTaskSubmittedByOperator // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AvsTaskSubmittedByOperatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AvsTaskSubmittedByOperator)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AvsTaskSubmittedByOperator)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AvsTaskSubmittedByOperatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AvsTaskSubmittedByOperatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AvsTaskSubmittedByOperator represents a TaskSubmittedByOperator event raised by the Avs contract.
type AvsTaskSubmittedByOperator struct {
	TaskContractAddress common.Address
	TaskId              uint64
	Sender              string
	TaskResponse        []byte
	BlsSignature        []byte
	Phase               uint8
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterTaskSubmittedByOperator is a free log retrieval operation binding the contract event 0xf58bb6dfa143b4c8016c4286efabb7d5fee03cdd6698a748c78e7fa5a6d5e464.
//
// Solidity: event TaskSubmittedByOperator(address indexed taskContractAddress, uint64 indexed taskId, string sender, bytes taskResponse, bytes blsSignature, uint8 phase)
func (_Avs *AvsFilterer) FilterTaskSubmittedByOperator(opts *bind.FilterOpts, taskContractAddress []common.Address, taskId []uint64) (*AvsTaskSubmittedByOperatorIterator, error) {

	var taskContractAddressRule []interface{}
	for _, taskContractAddressItem := range taskContractAddress {
		taskContractAddressRule = append(taskContractAddressRule, taskContractAddressItem)
	}
	var taskIdRule []interface{}
	for _, taskIdItem := range taskId {
		taskIdRule = append(taskIdRule, taskIdItem)
	}

	logs, sub, err := _Avs.contract.FilterLogs(opts, "TaskSubmittedByOperator", taskContractAddressRule, taskIdRule)
	if err != nil {
		return nil, err
	}
	return &AvsTaskSubmittedByOperatorIterator{contract: _Avs.contract, event: "TaskSubmittedByOperator", logs: logs, sub: sub}, nil
}

// WatchTaskSubmittedByOperator is a free log subscription operation binding the contract event 0xf58bb6dfa143b4c8016c4286efabb7d5fee03cdd6698a748c78e7fa5a6d5e464.
//
// Solidity: event TaskSubmittedByOperator(address indexed taskContractAddress, uint64 indexed taskId, string sender, bytes taskResponse, bytes blsSignature, uint8 phase)
func (_Avs *AvsFilterer) WatchTaskSubmittedByOperator(opts *bind.WatchOpts, sink chan<- *AvsTaskSubmittedByOperator, taskContractAddress []common.Address, taskId []uint64) (event.Subscription, error) {

	var taskContractAddressRule []interface{}
	for _, taskContractAddressItem := range taskContractAddress {
		taskContractAddressRule = append(taskContractAddressRule, taskContractAddressItem)
	}
	var taskIdRule []interface{}
	for _, taskIdItem := range taskId {
		taskIdRule = append(taskIdRule, taskIdItem)
	}

	logs, sub, err := _Avs.contract.WatchLogs(opts, "TaskSubmittedByOperator", taskContractAddressRule, taskIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AvsTaskSubmittedByOperator)
				if err := _Avs.contract.UnpackLog(event, "TaskSubmittedByOperator", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTaskSubmittedByOperator is a log parse operation binding the contract event 0xf58bb6dfa143b4c8016c4286efabb7d5fee03cdd6698a748c78e7fa5a6d5e464.
//
// Solidity: event TaskSubmittedByOperator(address indexed taskContractAddress, uint64 indexed taskId, string sender, bytes taskResponse, bytes blsSignature, uint8 phase)
func (_Avs *AvsFilterer) ParseTaskSubmittedByOperator(log types.Log) (*AvsTaskSubmittedByOperator, error) {
	event := new(AvsTaskSubmittedByOperator)
	if err := _Avs.contract.UnpackLog(event, "TaskSubmittedByOperator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
