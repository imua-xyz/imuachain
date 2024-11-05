package batch

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/xerrors"
)

var (
	AssetDecimalReduction    = new(big.Int).Exp(big.NewInt(10), big.NewInt(DefaultAssetDecimal), nil)
	DefaultDepositAmount     = big.NewInt(0).Mul(big.NewInt(10000), AssetDecimalReduction)
	HalfDefaultDepositAmount = big.NewInt(0).Quo(DefaultDepositAmount, big.NewInt(2))
)

type EnqueueTxParams struct {
	staker     *Staker
	nonce      *uint64
	msgType    string
	IsCosmosTx bool
	opAmount   *big.Int
	msgData    []byte
	assetID    uint
	operatorID uint
}

func (m *Manager) enqueueTxAndSaveRecord(params *EnqueueTxParams) error {
	// save the tx info in the local db for the future check
	helperRecord, err := LoadObjectByID[HelperRecord](m, SqliteDefaultStartID)
	if err != nil {
		return err
	}
	txRecord := &Transaction{
		StakerID:    params.staker.ID,
		Type:        params.msgType,
		IsCosmosTx:  params.IsCosmosTx,
		OpAmount:    params.opAmount.String(),
		Nonce:       *params.nonce,
		Status:      Queued,
		CheckResult: WaitToCheck,
		TestBatchID: helperRecord.CurrentBatchID,
		AssetID:     params.assetID,
		OperatorID:  params.operatorID,
	}
	err = SaveObject(m, txRecord)
	if err != nil {
		return err
	}
	// send the tx info to the queue
	sk, err := crypto.ToECDSA(params.staker.Sk)
	if err != nil {
		return xerrors.Errorf("can't convert the sk to ecdsa private key,staker:%v,err:%s", params.staker.ID, err)
	}

	evmTxInQueue := &EvmTxInQueue{
		sk:               sk,
		From:             params.staker.EvmAddress(),
		UseExternalNonce: true,
		Nonce:            *params.nonce,
		ToAddr:           &AssetsPrecompileAddr,
		Value:            big.NewInt(0),
		Data:             params.msgData,
		TxRecordID:       txRecord.ID,
	}
	m.TxsQueue <- evmTxInQueue
	// increase the nonce
	*params.nonce++
	return nil
}

func (m *Manager) EnqueueDepositWithdrawLSTTxs(isWithdrawal bool) error {
	assetsAbi, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return err
	}
	opAmount := DefaultDepositAmount
	msgType := assets.MethodDepositLST
	if isWithdrawal {
		msgType = assets.MethodWithdrawLST
		// The remaining amount has been delegated to the operators.
		// Therefore, we use half of the total deposit amount as the withdrawal amount.
		opAmount = HalfDefaultDepositAmount
	}

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
			err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
				staker:     staker,
				nonce:      &nonce,
				msgType:    msgType,
				IsCosmosTx: false,
				opAmount:   opAmount,
				msgData:    data,
				assetID:    assetId,
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

func (m *Manager) EnqueueDelegationTxs(isUndelegation bool) error {
	delegationAbi, err := abi.JSON(strings.NewReader(delegation.DelegationABI))
	if err != nil {
		return err
	}
	operatorNumber, err := ObjectsNumber(m, &Operator{})
	if err != nil {
		return err
	}
	msgType := delegation.MethodDelegate
	if isUndelegation {
		msgType = delegation.MethodUndelegate
	}

	ethHTTPClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	// construct and push all messages into the queue
	stakerOpFunc := func(stakerId uint, _ int64, staker *Staker) error {
		nonce, err := ethHTTPClient.NonceAt(m.ctx, staker.Address, nil)
		if err != nil {
			return xerrors.Errorf(
				"BatchDeposit: can't get staker's nonce, stakerId:%d, addr:%s,err:%s",
				stakerId, staker.Address.String(), err)
		}
		// using DefaultDepositAmount/(2*operatorNumber) as the amount of each delegation.
		// then we can use DefaultDepositAmount/2 as the withdrawal amount.
		totalDelegationAmount := HalfDefaultDepositAmount

		// todo: When calculating the delegation amount for the last operator, it's necessary
		// to account for the precision loss caused by integer division. Otherwise, it could
		// lead to failures in the deposit and withdrawal checks.
		singleDelegationAmount := big.NewInt(0).Quo(totalDelegationAmount, big.NewInt(operatorNumber))

		assetOpFunc := func(assetId uint, _ int64, asset *Asset) error {
			// Each asset needs to perform delegate and undelegate operations on all operators.
			operatorOpFunc := func(operatorId uint, _ int64, operator *Operator) error {
				data, err := delegationAbi.Pack(msgType, asset.ClientChainID, nonce, PaddingAddressTo32(asset.Address), PaddingAddressTo32(staker.Address), []byte(operator.Address), singleDelegationAmount)
				if err != nil {
					return err
				}
				err = m.enqueueTxAndSaveRecord(&EnqueueTxParams{
					staker:     staker,
					nonce:      &nonce,
					msgType:    msgType,
					IsCosmosTx: false,
					opAmount:   singleDelegationAmount,
					msgData:    data,
					assetID:    assetId,
					operatorID: operatorId,
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
		txHash, err := SignAndSendEvmTx(m.DefaultEvmTxRequirements, evmTx)
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
