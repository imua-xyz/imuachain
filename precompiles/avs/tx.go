//nolint:dupl
package avs

import (
	"fmt"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
)

const (
	MethodRegisterAVS               = "registerAVS"
	MethodUpdateAVS                 = "updateAVS"
	MethodDeregisterAVS             = "deregisterAVS"
	MethodRegisterOperatorToAVS     = "registerOperatorToAVS"
	MethodDeregisterOperatorFromAVS = "deregisterOperatorFromAVS"
	MethodCreateAVSTask             = "createTask"
	MethodRegisterBLSPublicKey      = "registerBLSPublicKey"
	MethodChallenge                 = "challenge"
	MethodOperatorSubmitTask        = "operatorSubmitTask"
)

// RegisterAVS AVSInfoRegister register the avs related information and change the state in avs keeper module.
func (p Precompile) RegisterAVS(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// parse the avs input params first.
	avsParams, err := p.GetAVSParamsFromInputs(contract, origin, method, args)
	if err != nil {
		return nil, errorsmod.Wrap(err, "parse args error")
	}

	// verification of the calling address to ensure it is avs contract owner
	if !slices.Contains(avsParams.AvsOwnerAddresses, avsParams.CallerAddress.String()) {
		return nil, errorsmod.Wrap(err, "not qualified to registerOrDeregister")
	}
	// The AVS registration is done by the calling contract.
	avsParams.AvsAddress = contract.CallerAddress
	avsParams.Action = avstypes.RegisterAction
	// Finally, update the AVS information in the keeper.
	err = p.avsKeeper.UpdateAVSInfo(ctx, avsParams)
	if err != nil {
		fmt.Println("Failed to update AVS info", err)
		return nil, err
	}
	if err = p.EmitAVSRegistered(ctx, stateDB, avsParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) DeregisterAVS(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodDeregisterAVS].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodDeregisterAVS].Inputs), len(args))
	}
	avsParams := &avstypes.AVSRegisterOrDeregisterParams{}
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	avsParams.CallerAddress = callerAddress[:]
	avsName, ok := args[1].(string)
	if !ok || avsName == "" {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "string", avsName)
	}
	avsParams.AvsName = avsName
	avsParams.AvsAddress = contract.CallerAddress
	avsParams.Action = avstypes.DeRegisterAction
	err := p.avsKeeper.UpdateAVSInfo(ctx, avsParams)
	if err != nil {
		return nil, err
	}
	if err = p.EmitAVSDeregistered(ctx, stateDB, avsParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) UpdateAVS(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// parse the avs input params first.
	avsParams, err := p.GetAVSParamsFromInputs(contract, origin, method, args)
	if err != nil {
		return nil, errorsmod.Wrap(err, "parse args error")
	}

	avsParams.AvsAddress = contract.CallerAddress
	avsParams.Action = avstypes.UpdateAction
	previousAVSInfo, err := p.avsKeeper.GetAVSInfo(ctx, avsParams.AvsAddress.String())
	if err != nil {
		return nil, err
	}
	// If avs UpdateAction check CallerAddress
	if !slices.Contains(previousAVSInfo.Info.AvsOwnerAddresses, avsParams.CallerAddress.String()) {
		return nil, fmt.Errorf("this caller not qualified to update %s", avsParams.CallerAddress)
	}
	err = p.avsKeeper.UpdateAVSInfo(ctx, avsParams)
	if err != nil {
		return nil, err
	}

	if err = p.EmitAVSUpdated(ctx, stateDB, avsParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) BindOperatorToAVS(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodRegisterOperatorToAVS].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterOperatorToAVS].Inputs), len(args))
	}
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}

	operatorParams := &avstypes.OperatorOptParams{}
	operatorParams.OperatorAddress = callerAddress[:]
	operatorParams.AvsAddress = contract.CallerAddress
	operatorParams.Action = avstypes.RegisterAction
	err := p.avsKeeper.OperatorOptAction(ctx, operatorParams)
	if err != nil {
		return nil, err
	}
	if err = p.EmitOperatorJoined(ctx, stateDB, operatorParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) UnbindOperatorToAVS(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodRegisterOperatorToAVS].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterOperatorToAVS].Inputs), len(args))
	}
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	operatorParams := &avstypes.OperatorOptParams{}
	operatorParams.OperatorAddress = callerAddress[:]
	operatorParams.AvsAddress = contract.CallerAddress
	operatorParams.Action = avstypes.DeRegisterAction
	err := p.avsKeeper.OperatorOptAction(ctx, operatorParams)
	if err != nil {
		return nil, err
	}
	if err = p.EmitOperatorOuted(ctx, stateDB, operatorParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

// CreateAVSTask Middleware uses imuachain's default avstask template to create tasks in avstask module.
func (p Precompile) CreateAVSTask(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	params, err := p.GetTaskParamsFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}
	params.TaskContractAddress = contract.CallerAddress
	taskID, err := p.avsKeeper.CreateAVSTask(ctx, params)
	if err != nil {
		return nil, err
	}
	if err = p.EmitTaskCreated(ctx, stateDB, params); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(taskID)
}

// Challenge Middleware uses imuadefault avstask template to create tasks in avstask module.
func (p Precompile) Challenge(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodChallenge].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodChallenge].Inputs), len(args))
	}
	challengeParams := &avstypes.ChallengeParams{}
	challengeParams.TaskContractAddress = contract.CallerAddress
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	challengeParams.CallerAddress = callerAddress[:]

	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "uint64", args[1])
	}
	challengeParams.TaskID = taskID

	taskAddress, ok := args[2].(common.Address)
	if !ok || (taskAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "common.Address", taskAddress)
	}
	challengeParams.TaskContractAddress = taskAddress

	actualThreshold, ok := args[3].(uint8)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 3, "uint8", actualThreshold)
	}
	challengeParams.ActualThreshold = actualThreshold

	isExpected, ok := args[4].(bool)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 4, "bool", isExpected)
	}
	challengeParams.IsExpected = isExpected

	eligibleRewardOperators, ok := args[5].([]common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 5, "[]common.Address", eligibleRewardOperators)
	}
	challengeParams.EligibleRewardOperators = eligibleRewardOperators

	eligibleSlashOperators, ok := args[6].([]common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 6, "[]common.Address", eligibleSlashOperators)
	}

	challengeParams.EligibleSlashOperators = eligibleSlashOperators
	err := p.avsKeeper.RaiseAndResolveChallenge(ctx, challengeParams)
	if err != nil {
		return nil, err
	}

	if err = p.EmitChallengeInitiated(ctx, stateDB, challengeParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) RegisterBLSPublicKey(
	ctx sdk.Context,
	_ common.Address,
	_ *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodRegisterBLSPublicKey].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterBLSPublicKey].Inputs), len(args))
	}
	blsParams := &avstypes.BlsParams{}
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	blsParams.OperatorAddress = callerAddress[:]
	avsAddress, ok := args[1].(common.Address)
	if !ok || (avsAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "common.Address", avsAddress)
	}
	blsParams.AvsAddress = avsAddress

	pubKeyBz, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "[]byte", pubKeyBz)
	}
	blsParams.PubKey = pubKeyBz

	pubKeyRegistrationSignature, ok := args[3].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 3, "[]byte", pubKeyRegistrationSignature)
	}
	blsParams.PubkeyRegistrationSignature = pubKeyRegistrationSignature

	pubKeyRegistrationMessageHash, ok := args[4].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 4, "[]byte", pubKeyRegistrationMessageHash)
	}
	blsParams.PubkeyRegistrationMessageHash = pubKeyRegistrationMessageHash

	err := p.avsKeeper.RegisterBLSPublicKey(ctx, blsParams)
	if err != nil {
		return nil, err
	}

	if err = p.EmitPublicKeyRegistered(ctx, stateDB, blsParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

// OperatorSubmitTask operator submit results
func (p Precompile) OperatorSubmitTask(
	ctx sdk.Context,
	_ common.Address,
	_ *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodOperatorSubmitTask].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodOperatorSubmitTask].Inputs), len(args))
	}
	resultParams := &avstypes.TaskResultParams{}

	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	resultParams.CallerAddress = callerAddress[:]

	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "uint64", args[1])
	}
	resultParams.TaskID = taskID

	taskResponse, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "[]byte", taskResponse)
	}
	resultParams.TaskResponse = taskResponse

	if len(taskResponse) == 0 {
		resultParams.TaskResponse = nil
	}

	blsSignature, ok := args[3].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 3, "[]byte", blsSignature)
	}
	resultParams.BlsSignature = blsSignature

	taskAddress, ok := args[4].(common.Address)
	if !ok || (taskAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 4, "common.Address", taskAddress)
	}
	resultParams.TaskContractAddress = taskAddress

	phase, ok := args[5].(uint8)
	if !ok {
		return nil, fmt.Errorf("invalid phase type: expected uint8, got %T", args[5])
	}

	// The phase of the Two-Phase Commit protocol:
	// 1 = Prepare phase (commit preparation)
	// 2 = Commit phase (final commitment)
	// validation of the phase number
	phaseEnum := avstypes.Phase(phase)
	if err := avstypes.ValidatePhase(phaseEnum); err != nil {
		return nil, fmt.Errorf("invalid phase value: %d. Expected 1 (Prepare) or 2 (Commit)", phase)
	}
	resultParams.Phase = phaseEnum

	result := &avstypes.TaskResultInfo{
		TaskId:              resultParams.TaskID,
		OperatorAddress:     resultParams.CallerAddress.String(),
		TaskContractAddress: resultParams.TaskContractAddress.String(),
		TaskResponse:        resultParams.TaskResponse,
		BlsSignature:        resultParams.BlsSignature,
		Phase:               phaseEnum,
	}
	err := p.avsKeeper.SubmitTaskResult(ctx, resultParams.CallerAddress.String(), result)
	if err != nil {
		return nil, err
	}

	if err := p.EmitTaskSubmittedByOperator(ctx, stateDB, resultParams); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}
