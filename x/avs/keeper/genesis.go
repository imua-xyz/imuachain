package keeper

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// InitGenesis initializes the module's state from a provided genesis state.
// Since this action typically occurs on chain starts, this function is allowed to panic.
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	// Store a lookup from codeHash to code. Since these are static parameters,
	// such a lookup is stored at genesis and never updated.
	k.evmKeeper.SetCode(ctx, types.ChainIDCodeHash.Bytes(), types.ChainIDCode)
	// Set all the avs infos
	for _, avs := range state.AvsInfos {
		avs.AvsAddress = strings.ToLower(avs.AvsAddress)
		avs.TaskAddr = strings.ToLower(avs.TaskAddr)
		avs.RewardAddr = strings.ToLower(avs.RewardAddr)
		avs.SlashAddr = strings.ToLower(avs.SlashAddr)
		err := k.SetAVSInfo(ctx, &avs) //nolint:gosec
		if err != nil {
			panic(errorsmod.Wrap(err, "failed to set all avs info"))
		}
	}
	// Set all the task infos
	for _, elem := range state.TaskInfos {
		elem.TaskContractAddress = strings.ToLower(elem.TaskContractAddress)
		err := k.SetTaskInfo(ctx, &elem) //nolint:gosec
		if err != nil {
			panic(errorsmod.Wrap(err, "failed to set all task info"))
		}
	}
	// Set all the bls infos
	for _, elem := range state.BlsPubKeys {
		err := k.SetOperatorPubKey(ctx, &elem) //nolint:gosec
		if err != nil {
			panic(errorsmod.Wrap(err, "failed to set all bls info"))
		}
	}
	// Set all the taskNum infos
	for _, elem := range state.TaskNums {
		elem.TaskAddr = strings.ToLower(elem.TaskAddr)
		k.SetTaskID(ctx, common.HexToAddress(elem.TaskAddr), elem.TaskId)
	}
	// Set all the task result infos
	for _, elem := range state.TaskResultInfos {
		elem.TaskContractAddress = strings.ToLower(elem.TaskContractAddress)
		k.SetTaskResultInfo(ctx, &elem) //nolint:gosec
	}
	// Set all the task challenge infos
	err := k.SetAllTaskChallengedInfo(ctx, state.ChallengeInfos)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all challenge info"))
	}
	// Set all the chainID infos
	for _, elem := range state.ChainIdInfos {
		elem.AvsAddress = strings.ToLower(elem.AvsAddress)
		k.SetAVSAddrToChainID(ctx, common.HexToAddress(elem.AvsAddress), elem.ChainId)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	res := types.GenesisState{}
	var err error
	res.AvsInfos, err = k.GetAllAVSInfos(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all avs infos"))
	}
	res.TaskInfos, err = k.GetAllTaskInfos(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all task infos").Error())
	}

	res.BlsPubKeys, err = k.GetAllBlsPubKeys(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all bls key info").Error())
	}

	res.TaskNums, err = k.GetAllTaskNums(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all TaskNums").Error())
	}

	res.TaskResultInfos, err = k.GetAllTaskResultInfos(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all TaskResultInfos").Error())
	}

	res.ChallengeInfos, err = k.GetAllChallengeInfos(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all ChallengeInfos").Error())
	}

	res.ChainIdInfos, err = k.GetAllChainIDInfos(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all ChainIdInfos").Error())
	}

	return &res
}
