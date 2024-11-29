package batch

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/xerrors"
)

var (
	AssetDecimalReduction = new(big.Int).Exp(big.NewInt(10), big.NewInt(DefaultAssetDecimal), nil)
	DefaultDepositAmount  = big.NewInt(0).Mul(big.NewInt(10000), AssetDecimalReduction)
)

type EnqueueTxParams struct {
	staker             *Staker
	nonce              *uint64
	msgType            string
	IsCosmosTx         bool
	opAmount           sdkmath.Int
	msgData            []byte
	assetID            uint
	operatorID         uint
	expectedCheckValue sdkmath.Int
}

func (m *Manager) enqueueTxAndSaveRecord(params *EnqueueTxParams) error {
	// save the tx info in the local db for the future check
	helperRecord, err := LoadObjectByID[HelperRecord](m, SqliteDefaultStartID)
	if err != nil {
		return err
	}
	txRecord := &Transaction{
		StakerID:           params.staker.ID,
		Type:               params.msgType,
		IsCosmosTx:         params.IsCosmosTx,
		OpAmount:           params.opAmount.String(),
		Nonce:              *params.nonce,
		Status:             Queued,
		CheckResult:        WaitToCheck,
		TestBatchID:        helperRecord.CurrentBatchID,
		AssetID:            params.assetID,
		OperatorID:         params.operatorID,
		ExpectedCheckValue: params.expectedCheckValue.String(),
	}
	err = SaveObject(m, txRecord)
	if err != nil {
		return err
	}
	// send the tx info to the queue
	sk, err := crypto.ToECDSA(params.staker.Sk)
	if err != nil {
		return xerrors.Errorf("can't convert the Sk to ecdsa private key,staker:%v,err:%s", params.staker.ID, err)
	}

	evmTxInQueue := &EvmTxInQueue{
		Sk:               sk,
		From:             params.staker.EvmAddress(),
		UseExternalNonce: true,
		Nonce:            *params.nonce,
		ToAddr:           &AssetsPrecompileAddr,
		Value:            big.NewInt(0),
		Data:             params.msgData,
		TxRecordID:       txRecord.ID,
	}
	select {
	case m.TxsQueue <- evmTxInQueue:
		// Successfully sent to the channel
	case <-m.Shutdown:
		// Received a shutdown signal, return immediately
		fmt.Println("Received shutdown signal, stopping...")
		return nil
	}
	// increase the nonce
	*params.nonce++
	return nil
}

func (m *Manager) EnqueueDepositWithdrawLSTTxs(msgType string) error {
	if msgType != assets.MethodDepositLST && msgType != assets.MethodWithdrawLST {
		return xerrors.Errorf("EnqueueDepositWithdrawLSTTxs invalid msg type:%s", msgType)
	}
	assetsAbi, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return err
	}
	opAmount := sdkmath.NewIntFromBigInt(DefaultDepositAmount)

	ethHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	// construct and push all messages into the queue
	stakerOpFunc := func(stakerId uint, _ int64, staker *Staker) error {
		nonce, err := ethHTTPClient.NonceAt(m.ctx, staker.Address, nil)
		if err != nil {
			return xerrors.Errorf(
				"BatchDeposit: can't get staker's nonce, stakerId:%d, addr:%s,err:%s",
				stakerId, staker.Address.String(), err)
		}
		assetOpFunc := func(assetId uint, _ int64, asset *Asset) error {
			data, err := assetsAbi.Pack(msgType, asset.ClientChainID, PaddingAddressTo32(asset.Address), PaddingAddressTo32(staker.Address), opAmount)
			if err != nil {
				return err
			}
			// get the total deposit amount before deposit or withdrawal
			stakerAssetInfo, err := m.QueryStakerAssetInfo(asset.ClientChainID, staker.EvmAddress().String(), asset.Address.String())
			if err != nil {
				logger.Error("EnqueueDepositWithdrawLSTTxs, error occurs when querying the staker asset info",
					"staker", staker.EvmAddress().String(), "asset", asset.Address.String(), "err", err)
				return err
			}
			expectedCheckValue := stakerAssetInfo.TotalDepositAmount
			if msgType == assets.MethodDepositLST {
				expectedCheckValue = expectedCheckValue.Add(opAmount)
			} else {
				if !stakerAssetInfo.WithdrawableAmount.IsPositive() {
					logger.Error("EnqueueDepositWithdrawLSTTxs, the WithdrawableAmount isn't positive, skip the withdrawal", "staker", staker.EvmAddress().String(), "asset", asset.Address.String())
					return nil
				}
				// withdraw all amount
				opAmount = stakerAssetInfo.WithdrawableAmount
				expectedCheckValue = expectedCheckValue.Sub(opAmount)
			}

			err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
				staker:             staker,
				nonce:              &nonce,
				msgType:            msgType,
				IsCosmosTx:         false,
				opAmount:           opAmount,
				msgData:            data,
				assetID:            assetId,
				expectedCheckValue: expectedCheckValue,
			})
			if err != nil {
				return err
			}
			return nil
		}
		err = IterateObjects(m, &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err = IterateObjects(m, &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) EnqueueDelegationTxs(msgType string) error {
	if msgType != delegation.MethodDelegate && msgType != delegation.MethodUndelegate {
		return xerrors.Errorf("EnqueueDelegationTxs invalid msg type:%s", msgType)
	}
	delegationAbi, err := abi.JSON(strings.NewReader(delegation.DelegationABI))
	if err != nil {
		return err
	}
	operatorNumber, err := ObjectsNumber(m, &Operator{})
	if err != nil {
		return err
	}

	ethHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	opAmount := sdkmath.ZeroInt()
	expectedCheckValue := sdkmath.ZeroInt()
	// construct and push all messages into the queue
	stakerOpFunc := func(stakerId uint, _ int64, staker *Staker) error {
		nonce, err := ethHTTPClient.NonceAt(m.ctx, staker.Address, nil)
		if err != nil {
			return xerrors.Errorf(
				"BatchDeposit: can't get staker's nonce, stakerId:%d, addr:%s,err:%s",
				stakerId, staker.Address.String(), err)
		}

		assetOpFunc := func(assetId uint, _ int64, asset *Asset) error {
			// Each asset needs to perform delegate and undelegate operations on all operators.
			stakerAssetInfo, err := m.QueryStakerAssetInfo(asset.ClientChainID, staker.EvmAddress().String(), asset.Address.String())
			if err != nil {
				logger.Error("EnqueueDelegationTxs, error occurs when querying the staker asset info",
					"staker", staker.EvmAddress().String(), "asset", asset.Address.String(), "err", err)
				return err
			}
			if msgType == delegation.MethodDelegate {
				if !stakerAssetInfo.WithdrawableAmount.IsPositive() {
					logger.Error("EnqueueDelegationTxs, the WithdrawableAmount isn't positive, skip the delegation", "staker", staker.EvmAddress().String(), "asset", asset.Address.String())
					return nil
				}
				// delegates half of the withdrawable amount to the operators
				opAmount = stakerAssetInfo.WithdrawableAmount.Quo(sdkmath.NewInt(2)).Quo(sdkmath.NewInt(operatorNumber))
			}
			operatorOpFunc := func(operatorId uint, _ int64, operator *Operator) error {
				delegatedAmount, err := m.QueryDelegatedAmount(asset.ClientChainID, staker.EvmAddress().String(), asset.Address.String(), operator.Address)
				if err != nil {
					return err
				}
				if msgType == delegation.MethodUndelegate {
					// undelegates all amount.
					opAmount = delegatedAmount
				} else {
					expectedCheckValue = delegatedAmount.Add(opAmount)
				}
				if !opAmount.IsPositive() {
					logger.Error("EnqueueDelegationTxs, the opAmount isn't positive, skip the test", "msgType", msgType, "staker", staker.EvmAddress().String(), "asset", asset.Address.String())
					return nil
				}
				data, err := delegationAbi.Pack(msgType, asset.ClientChainID, nonce, PaddingAddressTo32(asset.Address), PaddingAddressTo32(staker.Address), []byte(operator.Address), opAmount)
				if err != nil {
					return err
				}
				err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
					staker:             staker,
					nonce:              &nonce,
					msgType:            msgType,
					IsCosmosTx:         false,
					opAmount:           opAmount,
					msgData:            data,
					assetID:            assetId,
					operatorID:         operatorId,
					expectedCheckValue: expectedCheckValue,
				})
				if err != nil {
					return err
				}
				return nil
			}
			err = IterateObjects(m, &Operator{}, operatorOpFunc)
			if err != nil {
				return err
			}
			return nil
		}
		err = IterateObjects(m, &Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err = IterateObjects(m, &Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) SignAndSendTxs(tx interface{}) error {
	// sign and send the transaction
	var txID string
	evmTx, ok := tx.(*EvmTxInQueue)
	if ok {
		txHash, err := m.SignAndSendEvmTx(evmTx)
		if err != nil {
			logger.Error("can't sign and send the evm tx", "txHash", txHash, "err", err)
			return err
		}
		txID = txHash.String()
	}
	// todo: address the cosmos transaction for the delegation/undelegation of Exo token.

	// update the tx record in the local db for future check
	txRecord, err := LoadObjectByID[Transaction](m, evmTx.TxRecordID)
	if err != nil {
		return err
	}
	height, err := m.NodeEVMHTTPClients[DefaultNodeIndex].BlockNumber(m.ctx)
	if err != nil {
		return err
	}
	txRecord.SendTime = time.Now().String()
	txRecord.Status = Pending
	txRecord.SendHeight = height
	txRecord.TxHash = txID
	err = SaveObject[Transaction](m, txRecord)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) TickHandle(handleRate int, handle func() (bool, error)) error {
	if handleRate <= 0 {
		return xerrors.New("handleRate must be greater than 0")
	}

	// Calculate the interval time in milliseconds for each transaction based on the rate
	interval := time.Duration(1000/handleRate) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			isEndTicker, err := handle()
			if err != nil {
				logger.Error("TickHandle, error when call handle", "err", err)
			}
			if isEndTicker {
				logger.Info("TickHandle: end Ticker")
				return nil
			}
		case <-m.Shutdown:
			fmt.Println("Shutting down")
			return nil
		default:
			continue
		}
	}
}

func (m *Manager) DequeueAndSignSendTxs() error {
	handle := func() (bool, error) {
		select {
		case tx := <-m.TxsQueue:
			err := m.SignAndSendTxs(tx)
			if err != nil {
				logger.Error("DequeueAndSignSendTxs: can't sign and send the tx", "err", err)
				return false, err
			}
		default:
			return false, nil
		}
		return false, nil
	}
	return m.TickHandle(m.config.TxNumberPerSec, handle)
}
