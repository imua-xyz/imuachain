package keeper

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	rtypes "github.com/ExocoreNetwork/exocore/x/reward/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/evmos/v16/rpc/namespaces/ethereum/eth/filters"
)

type RewardParams struct {
	ClientChainLzID       uint64
	Action                types.CrossChainOpType
	AssetsAddress         []byte
	WithdrawRewardAddress []byte
	OpAmount              sdkmath.Int
}

func getRewardParamsFromEventLog(log *ethtypes.Log) (*RewardParams, error) {
	// check if action is to get reward
	var action types.CrossChainOpType
	var err error
	readStart := 0
	readEnd := types.CrossChainActionLength
	r := bytes.NewReader(log.Data[readStart:readEnd])
	err = binary.Read(r, binary.BigEndian, &action)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read action")
	}
	if action != types.WithdrawReward {
		// not handle the actions that isn't deposit
		return nil, nil
	}

	// decode the action parameters
	readStart = readEnd
	readEnd += types.GeneralAssetsAddrLength
	r = bytes.NewReader(log.Data[readStart:readEnd])
	var assetsAddress []byte
	err = binary.Read(r, binary.BigEndian, assetsAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read assets address")
	}

	readStart = readEnd
	readEnd += types.GeneralClientChainAddrLength
	r = bytes.NewReader(log.Data[readStart:readEnd])
	var rewardAddr []byte
	err = binary.Read(r, binary.BigEndian, rewardAddr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read assets address")
	}

	readStart = readEnd
	readEnd += types.CrossChainOpAmountLength
	amount := sdkmath.NewIntFromBigInt(big.NewInt(0).SetBytes(log.Data[readStart:readEnd]))

	var clientChainLzID uint64
	r = bytes.NewReader(log.Topics[types.ClientChainLzIDIndexInTopics][:])
	err = binary.Read(r, binary.BigEndian, &clientChainLzID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error occurred when binary read clientChainLzID from topic")
	}

	return &RewardParams{
		ClientChainLzID:       clientChainLzID,
		Action:                action,
		AssetsAddress:         assetsAddress,
		WithdrawRewardAddress: rewardAddr,
		OpAmount:              amount,
	}, nil
}

func (k Keeper) PostTxProcessing(ctx sdk.Context, _ core.Message, receipt *ethtypes.Receipt) error {
	// TODO check if contract address is valid layerZero relayer address
	// check if log address and topicId is valid
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
	if len(needLogs) == 0 {
		log.Println("the hook message doesn't have any event needed to handle")
		return nil
	}

	for _, log := range needLogs {
		rewardParams, err := getRewardParamsFromEventLog(log)
		if err != nil {
			return err
		}
		if rewardParams != nil {
			err = k.RewardForWithdraw(ctx, rewardParams)
			if err != nil {
				// todo: need to test if the changed storage state will be reverted if there is an error occurred
				return err
			}
		}
	}
	return nil
}

func (k Keeper) RewardForWithdraw(sdk.Context, *RewardParams) error {
	// TODO: rewards aren't yet supported
	// it is safe to return an error, since the precompile call will prevent an error
	// if err != nil return false
	// the false will ensure no unnecessary LZ messages are sent by the gateway
	return rtypes.ErrNotSupportYet
	// // check event parameter then execute RewardForWithdraw operation
	// if event.OpAmount.IsNegative() {
	// 	return errorsmod.Wrap(rtypes.ErrRewardAmountIsNegative, fmt.Sprintf("the amount is:%s", event.OpAmount))
	// }
	// stakeID, assetID := getStakeIDAndAssetID(event)
	// // check is asset exist
	// if !k.assetsKeeper.IsStakingAsset(ctx, assetID) {
	// 	return errorsmod.Wrap(rtypes.ErrRewardAssetNotExist, fmt.Sprintf("the assetID is:%s", assetID))
	// }

	// // TODO verify the reward amount is valid
	// changeAmount := types.DeltaStakerSingleAsset{
	// 	TotalDepositAmount: event.OpAmount,
	// 	WithdrawableAmount: event.OpAmount,
	// }
	// // TODO: there should be a reward pool to be transferred from for native tokens' reward, don't update staker-asset-info, just transfer exo-native-token:pool->staker or handled by validators since the reward would be transferred to validators directly.
	// if assetID != types.ExocoreAssetID {
	// 	err := k.assetsKeeper.UpdateStakerAssetState(ctx, stakeID, assetID, changeAmount)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err = k.assetsKeeper.UpdateStakingAssetTotalAmount(ctx, assetID, event.OpAmount); err != nil {
	// 		return err
	// 	}
	// }
	// return nil
}

// WithdrawDelegationRewards is an implementation of a function in the distribution interface.
// Since this module acts as the distribution module for our network, this function is here.
// When implemented, this function should find the pending (native token) rewards for the
// specified delegator and validator address combination and send them to the delegator address.
func (Keeper) WithdrawDelegationRewards(
	sdk.Context, sdk.AccAddress, sdk.ValAddress,
) (sdk.Coins, error) {
	return nil, rtypes.ErrNotSupportYet
}
