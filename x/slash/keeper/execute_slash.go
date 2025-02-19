package keeper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	//	"github.com/imua-xyz/imuachain/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/evmos/v16/rpc/namespaces/ethereum/eth/filters"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	rtypes "github.com/imua-xyz/imuachain/x/slash/types"
)

type SlashParams struct {
	ClientChainLzID           uint64
	Action                    assetstypes.CrossChainOpType
	AssetsAddress             []byte
	OperatorAddress           sdk.AccAddress
	StakerAddress             []byte
	MiddlewareContractAddress []byte
	Proportion                sdkmath.LegacyDec
	OpAmount                  sdkmath.Int
	Proof                     []byte
}

// nolint: unused // This is to be implemented.
type OperatorFrozenStatus struct {
	// nolint: unused // This is to be implemented.
	operatorAddress sdk.AccAddress
	// nolint: unused // This is to be implemented.
	status bool
}

func (k Keeper) getParamsFromEventLog(ctx sdk.Context, log *ethtypes.Log) (*SlashParams, error) {
	// check if action is deposit
	var action assetstypes.CrossChainOpType
	var err error
	readStart := uint32(0)
	readEnd := uint32(assetstypes.CrossChainActionLength)
	r := bytes.NewReader(log.Data[readStart:readEnd])
	err = binary.Read(r, binary.BigEndian, &action)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read action")
	}
	if action != assetstypes.DelegateTo && action != assetstypes.UndelegateFrom {
		// not handle the actions that isn't deposit
		return nil, nil
	}

	var clientChainLzID uint64
	r = bytes.NewReader(log.Topics[assetstypes.ClientChainLzIDIndexInTopics][:])
	err = binary.Read(r, binary.BigEndian, &clientChainLzID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read clientChainLzID from topic")
	}
	clientChainInfo, err := k.assetsKeeper.GetClientChainInfoByIndex(ctx, clientChainLzID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when get client chain info")
	}

	// decode the action parameters
	readStart = readEnd
	readEnd += clientChainInfo.AddressLength
	r = bytes.NewReader(log.Data[readStart:readEnd])
	assetsAddress := make([]byte, clientChainInfo.AddressLength)
	err = binary.Read(r, binary.BigEndian, assetsAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read assets address")
	}

	readStart = readEnd
	readEnd += assetstypes.ExoCoreOperatorAddrLength
	r = bytes.NewReader(log.Data[readStart:readEnd])
	operatorAddress := [assetstypes.ExoCoreOperatorAddrLength]byte{}
	err = binary.Read(r, binary.BigEndian, operatorAddress[:])
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read operator address")
	}
	opAccAddr, err := sdk.AccAddressFromBech32(string(operatorAddress[:]))
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when parse acc address from Bech32")
	}

	readStart = readEnd
	readEnd += clientChainInfo.AddressLength
	r = bytes.NewReader(log.Data[readStart:readEnd])
	stakerAddress := make([]byte, clientChainInfo.AddressLength)
	err = binary.Read(r, binary.BigEndian, stakerAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read staker address")
	}

	readStart = readEnd
	readEnd += assetstypes.CrossChainOpAmountLength
	amount := sdkmath.NewIntFromBigInt(big.NewInt(0).SetBytes(log.Data[readStart:readEnd]))

	return &SlashParams{
		ClientChainLzID: clientChainLzID,
		Action:          action,
		AssetsAddress:   assetsAddress,
		StakerAddress:   stakerAddress,
		OperatorAddress: opAccAddr,
		OpAmount:        amount,
	}, nil
}

func getStakeIDAndAssetID(params *SlashParams) (stakeID string, assetID string) {
	clientChainLzIDStr := hexutil.EncodeUint64(params.ClientChainLzID)
	stakeID = strings.Join([]string{hexutil.Encode(params.StakerAddress), clientChainLzIDStr}, "_")
	assetID = strings.Join([]string{hexutil.Encode(params.AssetsAddress), clientChainLzIDStr}, "_")
	return
}

func (k Keeper) FilterCrossChainEventLogs(ctx sdk.Context, _ core.Message, receipt *ethtypes.Receipt) ([]*ethtypes.Log, error) {
	params, err := k.assetsKeeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	// filter needed logs
	addresses := []common.Address{common.HexToAddress(params.ExocoreLzAppAddress)}
	topics := [][]common.Hash{
		{common.HexToHash(params.ExocoreLzAppEventTopic)},
	}
	needLogs := filters.FilterLogs(receipt.Logs, nil, nil, addresses, topics)
	return needLogs, nil
}

func (k Keeper) PostTxProcessing(ctx sdk.Context, _ core.Message, receipt *ethtypes.Receipt) error {
	params, err := k.assetsKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	// filter needed logs
	addresses := []common.Address{common.HexToAddress(params.ExocoreLzAppAddress)}
	topics := [][]common.Hash{
		{common.HexToHash(params.ExocoreLzAppEventTopic)},
	}
	needLogs := filters.FilterLogs(receipt.Logs, nil, nil, addresses, topics)
	if err != nil {
		return err
	}

	if len(needLogs) == 0 {
		log.Println("the hook message doesn't have any event needed to handle")
		return nil
	}

	for _, log := range needLogs {
		slashParams, err := k.getParamsFromEventLog(ctx, log)
		if err != nil {
			return err
		}
		if slashParams != nil {
			err = k.Slash(ctx, slashParams)
			if err != nil {
				// todo: need to test if the changed storage state will be reverted if there is an error occurred
				return err
			}
		}
	}
	return nil
}

// func (k Keeper) OptIntoSlashing(ctx sdk.Context, event *SlashParams) error {
// 	//TODO implement me
// 	panic("implement me")
// }

func (k Keeper) Slash(ctx sdk.Context, event *SlashParams) error {
	// TODO the stakes are frozen for the impacted middleware, and deposits and withdrawals are disabled as well. All pending deposits and withdrawals for the current epoch will be invalidated.
	//	_ = k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)

	// check event parameter then execute slash operation
	if event.OpAmount.IsNegative() {
		return errorsmod.Wrap(rtypes.ErrSlashAmountIsNegative, fmt.Sprintf("the amount is:%s", event.OpAmount))
	}
	stakeID, assetID := getStakeIDAndAssetID(event)
	// check if asset exists
	if !k.assetsKeeper.IsStakingAsset(ctx, assetID) {
		return errorsmod.Wrap(rtypes.ErrSlashAssetNotExist, fmt.Sprintf("the assetID is:%s", assetID))
	}

	// dont't create stakerasset info for native token.
	// TODO: do we need to do any other process for native token 'else{}' ?
	if assetID != assetstypes.ExocoreAssetID {
		changeAmount := assetstypes.DeltaStakerSingleAsset{
			TotalDepositAmount: event.OpAmount.Neg(),
			WithdrawableAmount: event.OpAmount.Neg(),
		}

		err := k.assetsKeeper.UpdateStakerAssetState(ctx, stakeID, assetID, changeAmount)
		if err != nil {
			return err
		}
		if err = k.assetsKeeper.UpdateStakingAssetTotalAmount(ctx, assetID, event.OpAmount.Neg()); err != nil {
			return err
		}
	}
	return nil
}

// func (k Keeper) FreezeOperator(ctx sdk.Context, event *SlashParams) error {
// 	k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)
// 	return nil
// }

//	func (k Keeper) ResetFrozenStatus(ctx sdk.Context, event *SlashParams) error {
//		k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)
//		return nil
//	}
// func (k Keeper) IsOperatorFrozen(ctx sdk.Context, event *SlashParams) (bool, error) {
// 	return k.GetFrozenStatus(ctx, string(event.OperatorAddress))

// }
// func (k Keeper) OperatorAssetSlashedProportion(ctx sdk.Context, opAddr sdk.AccAddress, assetID string, startHeight, endHeight uint64) sdkmath.LegacyDec {
// 	//TODO
// 	return sdkmath.LegacyNewDec(3)
// }
