package batch

import (
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/precompiles/assets"
	"github.com/imua-xyz/imuachain/precompiles/delegation"
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
	toAddr             *common.Address
	isCosmosTx         bool
	opAmount           sdkmath.Int
	msgData            []byte
	assetID            uint
	operatorID         uint
	testBatchID        uint
	expectedCheckValue sdkmath.Int
}

func (m *Manager) enqueueTxAndSaveRecord(params *EnqueueTxParams) error {
	// construct the tx for the future check
	txRecord := &Transaction{
		StakerID:           params.staker.ID,
		Type:               params.msgType,
		IsCosmosTx:         params.isCosmosTx,
		OpAmount:           params.opAmount.String(),
		Nonce:              *params.nonce,
		Status:             Queued,
		CheckResult:        WaitToCheck,
		TestBatchID:        params.testBatchID,
		AssetID:            params.assetID,
		OperatorID:         params.operatorID,
		ExpectedCheckValue: params.expectedCheckValue.String(),
	}
	// send the tx info to the queue
	sk, err := crypto.ToECDSA(params.staker.Sk)
	if err != nil {
		return xerrors.Errorf("can't convert the Sk to ecdsa private key,staker:%v,err:%w", params.staker.ID, err)
	}

	evmTxInQueue := &EvmTxInQueue{
		Sk:               sk,
		From:             params.staker.EvmAddress(),
		UseExternalNonce: true,
		Nonce:            *params.nonce,
		ToAddr:           params.toAddr,
		Value:            big.NewInt(0),
		Data:             params.msgData,
		TxRecord:         txRecord,
	}
	select {
	case m.TxsQueue <- evmTxInQueue:
		// Successfully sent to the channel
		m.QueueSize.Add(1)
	case <-m.Shutdown:
		// Received a shutdown signal, return immediately
		logger.Info("Received shutdown signal, stopping...")
		return nil
	}
	// increase the nonce
	*params.nonce++
	return nil
}

func (m *Manager) EnqueueDepositWithdrawLSTTxs(batchID uint, msgType string) error {
	if msgType != assets.MethodDepositLST && msgType != assets.MethodWithdrawLST {
		return xerrors.Errorf("EnqueueDepositWithdrawLSTTxs invalid msg type:%s", msgType)
	}
	assetsAbi, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return xerrors.Errorf("EnqueueDepositWithdrawLSTTxs, error when call assets abi.JSON,err:%w", err)
	}
	opAmount := sdkmath.NewIntFromBigInt(DefaultDepositAmount)

	ethHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	// construct and push all messages into the queue
	stakerOpFunc := func(stakerId uint, _ int64, staker Staker) error {
		nonce, err := ethHTTPClient.NonceAt(m.ctx, staker.Address, nil)
		if err != nil {
			return xerrors.Errorf(
				"BatchDeposit: can't get staker's nonce, stakerId:%d, addr:%s,err:%w",
				stakerId, staker.Address.String(), err)
		}
		assetOpFunc := func(assetId uint, _ int64, asset Asset) error {
			// get the total deposit amount before deposit or withdrawal
			stakerAssetInfo, err := m.QueryStakerAssetInfo(uint64(asset.ClientChainID), staker.EvmAddress().String(), asset.Address.String())
			var expectedCheckValue sdkmath.Int
			if msgType == assets.MethodDepositLST {
				if err == nil {
					// the staker has already owned this asset
					expectedCheckValue = stakerAssetInfo.TotalDepositAmount.Add(opAmount)
				} else {
					// the staker hasn't owned this asset
					expectedCheckValue = opAmount
				}
			} else {
				if err != nil {
					// the staker asset info must exist if the msg type is withdrawal.
					logger.Error("EnqueueDepositWithdrawLSTTxs, error occurs when querying the staker asset info, skip this test tx",
						"msgType", msgType, "staker", staker.EvmAddress().String(),
						"asset", asset.Address.String(), "err", err)
					return nil
				}
				if !stakerAssetInfo.WithdrawableAmount.IsPositive() {
					logger.Error("EnqueueDepositWithdrawLSTTxs, the WithdrawableAmount isn't positive, skip the withdrawal", "staker", staker.EvmAddress().String(), "asset", asset.Address.String())
					return nil
				}
				// withdraw all amount
				opAmount = stakerAssetInfo.WithdrawableAmount
				expectedCheckValue = stakerAssetInfo.TotalDepositAmount.Sub(opAmount)
			}
			data, err := assetsAbi.Pack(msgType, asset.ClientChainID, PaddingAddressTo32(asset.Address), PaddingAddressTo32(staker.Address), opAmount.BigInt())
			if err != nil {
				return xerrors.Errorf("EnqueueDepositWithdrawLSTTxs, error when call assetsAbi.Pack,err:%w", err)
			}
			err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
				staker:             &staker,
				nonce:              &nonce,
				toAddr:             &AssetsPrecompileAddr,
				msgType:            msgType,
				isCosmosTx:         false,
				opAmount:           opAmount,
				msgData:            data,
				assetID:            assetId,
				testBatchID:        batchID,
				expectedCheckValue: expectedCheckValue,
			})
			if err != nil {
				return err
			}
			return nil
		}
		err = IterateObjects(m.GetDB(), Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err = IterateObjects(m.GetDB(), Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) EnqueueDelegationTxs(batchID uint, msgType string) error {
	if msgType != delegation.MethodDelegate && msgType != delegation.MethodUndelegate {
		return xerrors.Errorf("EnqueueDelegationTxs invalid msg type:%s", msgType)
	}
	delegationAbi, err := abi.JSON(strings.NewReader(delegation.DelegationABI))
	if err != nil {
		return err
	}
	operatorNumber, err := ObjectsNumber(m.GetDB(), Operator{})
	if err != nil {
		return err
	}

	ethHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	opAmount := sdkmath.ZeroInt()
	expectedCheckValue := sdkmath.ZeroInt()
	// construct and push all messages into the queue
	stakerOpFunc := func(stakerId uint, _ int64, staker Staker) error {
		nonce, err := ethHTTPClient.NonceAt(m.ctx, staker.Address, nil)
		if err != nil {
			return xerrors.Errorf(
				"BatchDeposit: can't get staker's nonce, stakerId:%d, addr:%s,err:%w",
				stakerId, staker.Address.String(), err)
		}

		assetOpFunc := func(assetId uint, _ int64, asset Asset) error {
			// Each asset needs to perform delegate and undelegate operations on all operators.
			stakerAssetInfo, err := m.QueryStakerAssetInfo(uint64(asset.ClientChainID), staker.EvmAddress().String(), asset.Address.String())
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
			operatorOpFunc := func(operatorId uint, _ int64, operator Operator) error {
				delegatedAmount, err := m.QueryDelegatedAmount(uint64(asset.ClientChainID), staker.EvmAddress().String(), asset.Address.String(), operator.Address)
				if msgType == delegation.MethodUndelegate {
					if err == nil {
						// Undelegates half of the total amount. The expected check value will also be half.
						// The reason for keeping some amount delegated is to ensure there is always a portion
						// delegated to the operators, which helps in testing the Imuachain chain.
						opAmount = delegatedAmount.Quo(sdkmath.NewInt(2))
						expectedCheckValue = delegatedAmount.Sub(opAmount)
					}
					// opAmount will be zero if the delegation amount can't be quried, then the undelegation
					// will be skipped when checking the opAmount.
				} else {
					if err != nil {
						// it's the first delegation if the delegation amount can't be quired, then the expected
						// amount should be equl to the opAmount
						expectedCheckValue = opAmount
					} else {
						expectedCheckValue = delegatedAmount.Add(opAmount)
					}
				}
				if !opAmount.IsPositive() {
					logger.Error("EnqueueDelegationTxs, the opAmount isn't positive, skip the test", "msgType", msgType, "staker", staker.EvmAddress().String(), "asset", asset.Address.String())
					return nil
				}
				data, err := delegationAbi.Pack(msgType, asset.ClientChainID, nonce, PaddingAddressTo32(asset.Address), PaddingAddressTo32(staker.Address), []byte(operator.Address), opAmount.BigInt())
				if err != nil {
					return err
				}
				err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
					staker:             &staker,
					nonce:              &nonce,
					msgType:            msgType,
					toAddr:             &DelegationPrecompileAddr,
					isCosmosTx:         false,
					opAmount:           opAmount,
					msgData:            data,
					assetID:            assetId,
					operatorID:         operatorId,
					testBatchID:        batchID,
					expectedCheckValue: expectedCheckValue,
				})
				if err != nil {
					return err
				}
				return nil
			}
			err = IterateObjects(m.GetDB(), Operator{}, operatorOpFunc)
			if err != nil {
				return err
			}
			return nil
		}
		err = IterateObjects(m.GetDB(), Asset{}, assetOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err = IterateObjects(m.GetDB(), Staker{}, stakerOpFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) SignAndSendTxs(tx interface{}) (string, string, error) {
	// sign and send the transaction
	var txID string
	evmTx, ok := tx.(*EvmTxInQueue)
	if ok {
		txHash, err := m.SignAndSendEvmTx(evmTx)
		if err != nil {
			logger.Error("can't sign and send the evm tx", "txHash", txHash, "err", err)
			return txID, evmTx.TxRecord.Type, err
		}
		txID = txHash.String()
	} else {
		return txID, "", xerrors.Errorf("unsupported transaction type: %v", reflect.TypeOf(tx))
	}
	// todo: address the cosmos transaction for the delegation/undelegation of IMUA token.

	// update the tx record in the local db for future check
	height, err := m.NodeEVMHTTPClients[DefaultNodeIndex].BlockNumber(m.ctx)
	if err != nil {
		return txID, evmTx.TxRecord.Type, err
	}
	evmTx.TxRecord.SendTime = time.Now().String()
	evmTx.TxRecord.Status = Pending
	evmTx.TxRecord.SendHeight = height
	evmTx.TxRecord.TxHash = txID
	err = SaveObject[Transaction](m.GetDB(), *evmTx.TxRecord)
	if err != nil {
		return txID, evmTx.TxRecord.Type, err
	}
	return txID, evmTx.TxRecord.Type, nil
}

func (m *Manager) TickHandle(handleRate int, handle func() (bool, error)) error {
	if handleRate <= 0 {
		return xerrors.New("handleRate must be greater than 0")
	}
	if handleRate > 1000 {
		return xerrors.New("handleRate must be less than 1000")
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
			logger.Info("Shutting down")
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
			m.QueueSize.Add(-1)
			txID, txType, err := m.SignAndSendTxs(tx)
			if err != nil {
				logger.Error("DequeueAndSignSendTxs: can't sign and send the tx", "err", err)
				return false, err
			}
			logger.Info("DequeueAndSignSendTxs, sign and send tx successfully", "txType", txType, "txID", txID)
		default:
			return false, nil
		}
		return false, nil
	}
	return m.TickHandle(m.config.TxNumberPerSec, handle)
}
