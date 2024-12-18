package batch

import (
	"math/big"

	delegationkeeper "github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	assettypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/xerrors"
)

func (m *Manager) QueryAllBalance(addr sdktypes.AccAddress) (*banktypes.QueryAllBalancesResponse, error) {
	params := banktypes.NewQueryAllBalancesRequest(addr, &query.PageRequest{})
	queryClient := banktypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	return queryClient.AllBalances(m.ctx, params)
}

func (m *Manager) QueryBalance(addr sdktypes.AccAddress, denom string) (*banktypes.QueryBalanceResponse, error) {
	balanceReq := banktypes.NewQueryBalanceRequest(addr, denom)
	queryClient := banktypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	return queryClient.Balance(m.ctx, balanceReq)
}

func (m *Manager) GasPrice() (*big.Int, error) {
	evmHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	return evmHTTPClient.SuggestGasPrice(m.ctx)
}

func (m *Manager) QueryStakerAssetInfo(clientChainLzID uint64, stakerAddr, assetAddr string) (*assettypes.StakerAssetInfo, error) {
	queryClient := assettypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	stakerID, assetID := assettypes.GetStakerIDAndAssetIDFromStr(clientChainLzID, stakerAddr, assetAddr)
	req := &assettypes.QuerySpecifiedAssetAmountReq{
		StakerID: stakerID, // already lowercase
		AssetID:  assetID,  // already lowercase
	}
	return queryClient.QueStakerSpecifiedAssetAmount(m.ctx, req)
}

func (m *Manager) QueryDelegatedAmount(clientChainLzID uint64, stakerAddr, assetAddr, operatorAddr string) (sdkmath.Int, error) {
	queryDelegationClient := delegationtype.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	stakerID, assetID := assettypes.GetStakerIDAndAssetIDFromStr(clientChainLzID, stakerAddr, assetAddr)
	req := &delegationtype.SingleDelegationInfoReq{
		StakerID:     stakerID,     // already lowercase
		AssetID:      assetID,      // already lowercase
		OperatorAddr: operatorAddr, // already lowercase
	}
	delegationInfo, err := queryDelegationClient.QuerySingleDelegationInfo(m.ctx, req)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	operatorAssetReq := &assettypes.QueryOperatorSpecifiedAssetAmountReq{
		OperatorAddr: operatorAddr, // already lowercase
		AssetID:      assetID,      // already lowercase
	}
	queryAssetsClient := assettypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	operatorAssetInfo, err := queryAssetsClient.QueOperatorSpecifiedAssetAmount(m.ctx, operatorAssetReq)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	return delegationkeeper.TokensFromShares(delegationInfo.UndelegatableShare, operatorAssetInfo.TotalShare, operatorAssetInfo.TotalAmount)
}

func (m *Manager) PrecompileTxOnChainCheck(batchID uint, msgType string) error {
	isEndTicker := false
	txIDs, count, err := GetTxIDsByBatchTypeAndStatus(m.GetDB(), batchID, msgType, Pending)
	if err != nil {
		return err
	}
	if count <= 0 {
		return xerrors.Errorf("PrecompileTxOnChainCheck, there isn't any pending txs,msgType:%s,batchID:%d", msgType, batchID)
	}
	txIndex := int64(0)
	evmNodeClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	handle := func() (bool, error) {
		txID := txIDs[txIndex]
		txRecord, err := LoadObjectByID[Transaction](m.GetDB(), txID)
		if err != nil {
			return false, err
		}
		// check if the evm txRecord is on chain
		if !txRecord.IsCosmosTx {
			receipt, err := evmNodeClient.TransactionReceipt(m.ctx, common.HexToHash(txRecord.TxHash))
			if err != nil {
				// If the receipt isn't found, continue addressing the next txRecord
				logger.Error("PrecompileTxOnChainCheck, can't get the evm txRecord receipt", "txID", txRecord.TxHash, "err", err)
			} else {
				// update the transaction status
				// TODO: The status will always show as successful because we package the result as a successful execution,
				// even if an error occurs. To verify whether the execution is truly successful, some event logs need to be
				// emitted in the precompile. We can then perform the check based on the log.
				// Currently, we only verify if the transaction is on-chain. The actual success of the execution can be
				// determined through the subsequent asset state check.
				if receipt.Status == types.ReceiptStatusSuccessful {
					txRecord.Status = OnChainAndSuccessful
				} else {
					txRecord.Status = OnChainButFailed
				}
				logger.Info("PrecompileTxOnChainCheck, the evm tx has been on chain successfully", "status", receipt.Status, "msgType", msgType, "batchID", batchID, "height", receipt.BlockNumber, "txID", receipt.TxHash)
				err = SaveObject[Transaction](m.GetDB(), txRecord)
				if err != nil {
					logger.Error("PrecompileTxOnChainCheck, can't save the evm txRecord receipt", "txID", txRecord.TxHash, "err", err)
				}
			}
		}
		// todo: check if the cosmos txRecord is on chain

		txIndex++
		if txIndex == count {
			logger.Info("end the ticker for PrecompileTxOnChainCheck")
			isEndTicker = true
		}
		return isEndTicker, nil
	}
	return m.TickHandle(m.config.TxChecksPerSecond, handle)
}

// DepositWithdrawLSTCheck : By default, we require each batch to follow the order of
// deposits -> delegations -> undelegations -> withdrawals for batch testing.
// `DefaultDepositAmount` is used as the amount for batch deposit tests.
// Therefore, we only need to verify if `StakerAssetInfo.total_deposit_amount`
// has increased by `DefaultDepositAmount`.
// For batch withdrawal tests, we will withdraw the total withdrawable amount
// and record the opAmount. Thus, when checking withdrawal test transactions,
// we only need to verify if `StakerAssetInfo.total_deposit_amount` has decreased by opAmount.
func (m *Manager) DepositWithdrawLSTCheck(batchID uint, msgType string) error {
	stakerOpFunc := func(_ uint, _ int64, staker *Staker) error {
		assetOpFunc := func(_ uint, _ int64, asset *Asset) error {
			res, err := m.QueryStakerAssetInfo(uint64(asset.ClientChainID), staker.EvmAddress().String(), asset.Address.String())
			if err != nil {
				logger.Error("DepositWithdrawLSTCheck, error occurs when querying the staker asset info",
					"staker", staker.Name, "asset", asset.Name, "err", err)
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
					"staker", staker.Name, "asset", asset.Name, "err", err)
				// return nil to continue the next check
				return nil
			}
			expectedCheckAmount, ok := sdkmath.NewIntFromString(transaction.ExpectedCheckValue)
			if !ok {
				logger.Error("DepositWithdrawLSTCheck, can't parse the expected check value",
					"staker", staker.Name, "stakerAddr", staker.EvmAddress(),
					"asset", asset.Name, "assetAddr", asset.Address,
					"expectedCheckValue", transaction.ExpectedCheckValue, "err", err)
				// return nil to continue the next check
				return nil
			}
			if res.TotalDepositAmount.Equal(expectedCheckAmount) {
				logger.Info("DepositWithdrawLSTCheck, the check is successful", "txID", transaction.TxHash,
					"staker", staker.Name, "stakerAddr", staker.EvmAddress(),
					"asset", asset.Name, "assetAddr", asset.Address)
				transaction.CheckResult = Successful
			} else {
				logger.Error("DepositWithdrawLSTCheck, the check is Failed",
					"expectedAmount", expectedCheckAmount,
					"actualAmount", res.TotalDepositAmount,
					"txID", transaction.TxHash,
					"staker", staker.Name, "stakerAddr", staker.EvmAddress(),
					"asset", asset.Name, "assetAddr", asset.Address)
				transaction.CheckResult = Failed
				transaction.ActualCheckValue = res.TotalDepositAmount.String()
			}
			// update the transaction record.
			err = SaveObject[Transaction](m.GetDB(), transaction)
			if err != nil {
				logger.Error("DepositWithdrawLSTCheck, can't update the transaction record", "txID", transaction.TxHash, "staker", staker.Name, "asset", asset.Name, "err", err)
			}
			return nil
		}
		err := IterateObjects(m.GetDB(), &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) EvmDelegationCheck(batchID uint, msgType string) error {
	if msgType != delegation.MethodDelegate && msgType != delegation.MethodUndelegate {
		return xerrors.Errorf("EvmDelegationCheck invalid msg type:%s", msgType)
	}
	stakerOpFunc := func(_ uint, _ int64, staker *Staker) error {
		assetOpFunc := func(_ uint, _ int64, asset *Asset) error {
			operatorOpFunc := func(_ uint, _ int64, operator *Operator) error {
				delegatedAmount, err := m.QueryDelegatedAmount(uint64(asset.ClientChainID), staker.EvmAddress().String(), asset.Address.String(), operator.Address)
				if err != nil {
					logger.Error("EvmDelegationCheck, error occurs when querying the delegated amount",
						"staker", staker.Name, "asset", asset.Name, "operator", operator.Name, "err", err)
					// return nil to continue the next check
					return nil
				}
				var transaction Transaction
				err = m.GetDB().
					Where("test_batch_id = ? AND type = ? AND staker_id = ? AND asset_id = ? AND operator_id = ? AND status = ?",
						batchID, msgType, staker.ID, asset.ID, operator.ID, OnChainAndSuccessful).
					First(&transaction).
					Error
				if err != nil {
					logger.Error("EvmDelegationCheck, can't get the tx record",
						"staker", staker.Name, "asset", asset.Name, "operator", operator.Name, "err", err)
					// return nil to continue the next check
					return nil
				}

				expectedCheckAmount, ok := sdkmath.NewIntFromString(transaction.ExpectedCheckValue)
				if !ok {
					logger.Error("EvmDelegationCheck, can't parse the expected check value",
						"staker", staker.Name, "asset", asset.Name, "operator", operator.Name,
						"expectedCheckValue", transaction.ExpectedCheckValue, "err", err)
					// return nil to continue the next check
					return nil
				}
				if delegatedAmount.Equal(expectedCheckAmount) {
					logger.Info("EvmDelegationCheck, the check is successful", "txID", transaction.TxHash,
						"staker", staker.Name, "stakerAddr", staker.EvmAddress(),
						"asset", asset.Name, "assetAddr", asset.Address,
						"operatorName", operator.Name, "operatorAddr", operator.AccAddress())
					transaction.CheckResult = Successful
				} else {
					logger.Error("EvmDelegationCheck, the check is Failed",
						"expectedAmount", expectedCheckAmount,
						"actualAmount", delegatedAmount,
						"txID", transaction.TxHash,
						"staker", staker.Name, "stakerAddr", staker.EvmAddress(),
						"asset", asset.Name, "assetAddr", asset.Address,
						"operatorName", operator.Name, "operatorAddr", operator.AccAddress())
					transaction.CheckResult = Failed
					transaction.ActualCheckValue = delegatedAmount.String()
				}
				// update the transaction record.
				err = SaveObject[Transaction](m.GetDB(), transaction)
				if err != nil {
					logger.Error("EvmDelegationCheck, can't update the transaction record", "txID", transaction.TxHash, "staker", staker.Name, "asset", asset.Name, "operator", operator.Name, "err", err)
				}
				return nil
			}
			err := IterateObjects(m.GetDB(), &Operator{}, operatorOpFunc)
			if err != nil {
				return err
			}
			return nil
		}
		err := IterateObjects(m.GetDB(), &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) FundingCheck() error {
	err := CheckObjectsBalance(m, &Staker{}, m.config.StakerExoAmount)
	if err != nil {
		return err
	}
	err = CheckObjectsBalance(m, &Operator{}, m.config.OperatorExoAmount)
	if err != nil {
		return err
	}
	err = CheckObjectsBalance(m, &AVS{}, m.config.AVSExoAmount)
	if err != nil {
		return err
	}
	return nil
}
