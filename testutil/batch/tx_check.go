package batch

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	assettypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/xerrors"
)

func (m *Manager) GetPendingTxIDsByBatchAndType(batchID uint, txType string) ([]uint, int64, error) {
	var ids []uint
	pageSize := 1000
	page := 1
	var err error
	// Query only the ID field of transactions with the given TestBatchID and Type
	for {
		var pageIDs []uint
		err = m.GetDB().
			Model(&Transaction{}).
			Where("test_batch_id = ? AND type = ? AND status = ?", batchID, txType, Pending).
			Order("id ASC").
			Limit(pageSize).
			Offset((page-1)*pageSize).
			Pluck("id", &pageIDs).
			Error

		if err != nil || len(pageIDs) == 0 {
			break
		}

		ids = append(ids, pageIDs...)
		page++
	}

	if err != nil {
		return nil, 0, xerrors.Errorf("Failed to retrieve transaction IDs with TestBatchID %d and Type %s, err: %s", batchID, txType, err)
	}

	return ids, int64(len(ids)), nil
}

func (m *Manager) TxOnChainCheck(batchID uint, msgType string) error {
	isEndTicker := false
	txIDs, count, err := m.GetPendingTxIDsByBatchAndType(batchID, msgType)
	if err != nil {
		return err
	}
	txIndex := int64(0)
	evmNodeClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	handle := func() (bool, error) {
		txID := txIDs[txIndex]
		txRecord, err := LoadObjectByID[Transaction](m, txID)
		if err != nil {
			return false, err
		}
		// check if the evm txRecord is on chain
		if !txRecord.IsCosmosTx {
			receipt, err := evmNodeClient.TransactionReceipt(m.ctx, common.HexToHash(txRecord.TxHash))
			if err != nil {
				// If the receipt isn't found, continue addressing the next txRecord
				logger.Error("TxOnChainCheck, can't get the evm txRecord receipt", "txID", txRecord.TxHash, "err", err)
			} else {
				// update the transaction status
				if receipt.Status == types.ReceiptStatusSuccessful {
					txRecord.Status = OnChainAndSuccessful
				} else {
					txRecord.Status = OnChainButFailed
				}
				err = SaveObject[Transaction](m, txRecord)
				if err != nil {
					logger.Error("TxOnChainCheck, can't save the evm txRecord receipt", "txID", txRecord.TxHash, "err", err)
				}
			}
		}
		// todo: check if the cosmos txRecord is on chain

		txIndex++
		if txIndex == count {
			isEndTicker = true
		}
		return isEndTicker, nil
	}
	return m.TickHandle(m.config.TxChecksPerSecond, handle)
}

// DepositWithdrawLSTCheck : By default, we require each batch to follow the order of
// deposits -> delegations -> undelegations -> withdrawals for batch testing.
// During delegation, only half of the deposit amount is used, with the other half
// reserved for withdrawals. Therefore, when checking:
// 1. After the deposits are completed, the totalDepositAmount should be equal to:
// DefaultDepositAmount + batchID * (DefaultDepositAmount / 2) where batchID starts from 0.
// 2. After the withdrawal tests are completed, the totalDepositAmount should be:
// (DefaultDepositAmount / 2) * (batchID + 1).
func (m *Manager) DepositWithdrawLSTCheck(batchID uint, msgType string) error {
	var expectedAmount *big.Int
	switch msgType {
	case assets.MethodDepositLST:
		expectedAmount = big.NewInt(0).Add(
			DefaultDepositAmount,
			big.NewInt(0).Mul(HalfDefaultDepositAmount, big.NewInt(int64(batchID))),
		)
	case assets.MethodWithdrawLST:
		expectedAmount = big.NewInt(0).Mul(HalfDefaultDepositAmount, big.NewInt(int64(batchID+1)))
	default:
		return xerrors.Errorf("DepositWithdrawLSTCheck, invalid msgType:%s", msgType)
	}

	queryClient := assettypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	stakerOpFunc := func(stakerDBID uint, _ int64, staker *Staker) error {
		assetOpFunc := func(assetDBID uint, _ int64, asset *Asset) error {
			stakerID, assetID := assettypes.GetStakerIDAndAssetIDFromStr(asset.ClientChainID, staker.EvmAddress().String(), asset.Address.String())
			req := &assettypes.QuerySpecifiedAssetAmountReq{
				StakerID: stakerID, // already lowercase
				AssetID:  assetID,  // already lowercase
			}
			res, err := queryClient.QueStakerSpecifiedAssetAmount(m.ctx, req)
			if err != nil {
				logger.Error("DepositWithdrawLSTCheck, error occurs when calling QueStakerSpecifiedAssetAmount",
					"stakerDBID", stakerDBID, "assetDBID", assetDBID, "err", err)
				// return nil to continue the next check
				return nil
			}
			var transaction Transaction
			err = m.GetDB().
				Where("test_batch_id = ? AND type = ? AND staker_id = ? AND asset_id = ? AND status = ?",
					batchID, msgType, staker.ID, asset.ID, OnChainAndSuccessful).
				First(&transaction).
				Error
			if err != nil {
				logger.Error("DepositWithdrawLSTCheck, can't get the tx record",
					"stakerDBID", stakerDBID, "assetDBID", assetDBID, "err", err)
				// return nil to continue the next check
				return nil
			}
			if res.TotalDepositAmount.Equal(sdkmath.NewIntFromBigInt(expectedAmount)) {
				transaction.CheckResult = Successful
			} else {
				transaction.CheckResult = failed
				transaction.ExpectedCheckValue = expectedAmount.String()
				transaction.ActualCheckValue = res.TotalDepositAmount.String()
			}

			return nil
		}
		err := IterateObjects(m, &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m, &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) EvmDelegationCheck(batchID uint, msgType string) error {
	expectedAmount := sdkmath.LegacyZeroDec()
	queryClient := delegationtype.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	stakerOpFunc := func(stakerDBID uint, _ int64, staker *Staker) error {
		assetOpFunc := func(assetDBID uint, _ int64, asset *Asset) error {
			operatorOpFunc := func(_ uint, _ int64, operator *Operator) error {
				stakerID, assetID := assettypes.GetStakerIDAndAssetIDFromStr(asset.ClientChainID, staker.EvmAddress().String(), asset.Address.String())
				req := &delegationtype.SingleDelegationInfoReq{
					StakerID:     stakerID,         // already lowercase
					AssetID:      assetID,          // already lowercase
					OperatorAddr: operator.Address, // already lowercase
				}
				res, err := queryClient.QuerySingleDelegationInfo(m.ctx, req)
				if err != nil {
					return err
				}
				var transaction Transaction
				err = m.GetDB().
					Where("test_batch_id = ? AND type = ? AND staker_id = ? AND asset_id = ? AND operator_id = ? AND status = ?",
						batchID, msgType, staker.ID, asset.ID, operator.ID, OnChainAndSuccessful).
					First(&transaction).
					Error
				if err != nil {
					logger.Error("EvmDelegationCheck, can't get the tx record",
						"stakerDBID", stakerDBID, "assetDBID", assetDBID, "operator", operator.Address, "err", err)
					// return nil to continue the next check
					return nil
				}
				if msgType == delegation.MethodDelegate {
					expectedAmount, err = sdkmath.LegacyNewDecFromStr(transaction.OpAmount)
					if err != nil {
						logger.Error("EvmDelegationCheck, can't get the expectedAmount",
							"stakerDBID", stakerDBID, "assetDBID", assetDBID, "operator", operator.Address,
							"OpAmount", transaction.OpAmount, "err", err)
						// return nil to continue the next check
						return nil
					}
				}
				if res.UndelegatableShare.Equal(expectedAmount) {
					transaction.CheckResult = Successful
				} else {
					transaction.CheckResult = failed
					transaction.ExpectedCheckValue = expectedAmount.String()
					transaction.ActualCheckValue = res.UndelegatableShare.String()
				}
				return nil
			}
			err := IterateObjects(m, &Operator{}, operatorOpFunc)
			if err != nil {
				return err
			}
			return nil
		}
		err := IterateObjects(m, &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m, &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}
