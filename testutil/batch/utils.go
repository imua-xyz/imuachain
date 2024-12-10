package batch

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"gorm.io/gorm"

	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/cmd/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/xerrors"
)

var SqliteDefaultStartID = uint(1)

func ObjectsNumber[T any](db *gorm.DB, model T) (int64, error) {
	// Get the current count of objects in the table
	var count int64
	err := db.Model(&model).Count(&count).Error
	if err != nil {
		return 0, xerrors.Errorf("Failed to count %T objects, err: %w", model, err)
	}
	return count, nil
}

// CreateObjects now accepts a `createNewObject` function to customize how objects are created
func CreateObjects[T any](db *gorm.DB, model T, targetCount int64, createNewObject func(id uint) (T, error)) error {
	// Automatically migrate the schema for the model
	err := db.AutoMigrate(&model)
	if err != nil {
		return xerrors.Errorf("Auto migration Failed for %T table, err: %w", model, err)
	}

	// Get the current count of objects in the table
	count, err := ObjectsNumber(db, model)
	if err != nil {
		return err
	}

	// If the number of objects is less than the target count, create new ones
	if count < targetCount {
		numToAdd := targetCount - count
		objects := make([]T, numToAdd)
		for i := int64(0); i < numToAdd; i++ {
			// Use the provided `createNewObject` function to create the new object
			objects[i], err = createNewObject(uint(count + i + 1))
			if err != nil {
				return err
			}
		}

		// Insert the new objects into the database
		err = db.Create(&objects).Error
		if err != nil {
			return xerrors.Errorf("Failed to insert new %T objects, err: %w", model, err)
		}
	}

	return nil
}

func LoadObjectByID[T any](db *gorm.DB, id uint) (T, error) {
	var obj T
	err := db.First(&obj, id).Error
	if err != nil {
		return obj, xerrors.Errorf("Failed to load %T object with ID %d, err: %w", obj, id, err)
	}
	return obj, nil
}

func SaveObject[T any](db *gorm.DB, obj T) error {
	// Automatically migrate the schema, creating the table if it doesn't exist
	if err := db.AutoMigrate(&obj); err != nil {
		return xerrors.Errorf("Failed to auto migrate schema for %T, err: %w", obj, err)
	}

	// Now save the object
	err := db.Save(&obj).Error
	if err != nil {
		return xerrors.Errorf("Failed to save %T object, err: %w", obj, err)
	}
	return nil
}

func IterateObjects[T any](db *gorm.DB, model T, opFunc func(id uint, objectNumber int64, object T) error) error {
	objectNumber, err := ObjectsNumber(db, model)
	if err != nil {
		return err
	}

	for id := uint(1); id <= uint(objectNumber); id++ {
		// check if the balance is enough
		object, err := LoadObjectByID[T](db, id)
		if err != nil {
			return err
		}
		err = opFunc(id, objectNumber, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetTxIDsByBatchTypeAndStatus(db *gorm.DB, batchID uint, txType string, status int) ([]uint, int64, error) {
	var ids []uint
	pageSize := 1000
	page := 1
	var err error
	// Query only the ID field of transactions with the given TestBatchID and Type
	for {
		var pageIDs []uint
		err = db.
			Model(&Transaction{}).
			Where("test_batch_id = ? AND type = ? AND status = ?", batchID, txType, status).
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
		return nil, 0, xerrors.Errorf("Failed to retrieve transaction IDs with TestBatchID %d and Type %s, err: %w", batchID, txType, err)
	}

	return ids, int64(len(ids)), nil
}

func FundingObjects[T AddressForFunding](m *Manager, model T, needExo int64) error {
	if m.config.AddrNumberInMultiSend <= 0 {
		return xerrors.Errorf("invalid AddrNumberInMultiSend:%d", m.config.AddrNumberInMultiSend)
	}
	if len(m.NodeEVMHTTPClients) == 0 {
		return xerrors.Errorf("no available EVM HTTP clients")
	}

	faucetAddr := sdktypes.AccAddress(crypto.PubkeyToAddress(m.FaucetSK.PublicKey).Bytes())
	input := banktypes.Input{
		Address: faucetAddr.String(), // Sender address
	}
	outputs := make([]banktypes.Output, 0)

	multiSendMsgs := make([]sdktypes.Msg, 0)
	inputAmount := sdktypes.ZeroInt()
	totalInputAmount := sdktypes.ZeroInt()
	addrNumberInOneMsg := 0
	opFunc := func(id uint, objectNumber int64, object T) error {
		if !object.ShouldFund() {
			// skip this object then continue the other objects
			return nil
		}
		// select evm http client
		selectNode := int(id) % len(m.NodeEVMHTTPClients)
		balance, err := m.NodeEVMHTTPClients[selectNode].BalanceAt(m.ctx, object.EvmAddress(), nil)
		if err != nil {
			return xerrors.Errorf("can't get balance,addr:%s, err: %w", object.EvmAddress().String(), err)
		}
		exoBalance := big.NewInt(0).Quo(balance, ExoDecimalReduction).Int64()
		logger.Info("the exo balance is:", "addr", object.EvmAddress(), "balance", balance, "exoBalance", exoBalance, "needExo", needExo)
		if exoBalance < needExo {
			objectAccAddr := object.AccAddress()
			amount := sdktypes.NewInt(needExo - exoBalance)

			addrNumberInOneMsg++
			inputAmount = inputAmount.Add(amount)
			huaAmount := amount.Mul(sdkmath.NewIntFromBigInt(ExoDecimalReduction))
			outputs = append(outputs, banktypes.Output{
				Address: objectAccAddr.String(), // Sender address
				Coins:   sdktypes.Coins{sdktypes.NewCoin(config.BaseDenom, huaAmount)},
			})
		}
		logger.Info("generate inputs and outputs", "id", id, "objectNumber", objectNumber, "addrNumberInOneMsg", addrNumberInOneMsg, "AddrNumberInMultiSend", m.config.AddrNumberInMultiSend)
		if addrNumberInOneMsg != 0 &&
			(addrNumberInOneMsg == m.config.AddrNumberInMultiSend || id == uint(objectNumber)) {
			inputHuaAmount := inputAmount.Mul(sdkmath.NewIntFromBigInt(ExoDecimalReduction))
			totalInputAmount = totalInputAmount.Add(inputHuaAmount)
			input.Coins = sdktypes.Coins{sdktypes.NewCoin(config.BaseDenom, inputHuaAmount)}
			multiSendMsgs = append(multiSendMsgs, &banktypes.MsgMultiSend{
				Inputs:  []banktypes.Input{input},
				Outputs: outputs,
			})
			// clear the inputAmount and addrNumberInOneMsg
			inputAmount = sdktypes.ZeroInt()
			addrNumberInOneMsg = 0
			// using a new outputs
			outputs = make([]banktypes.Output, 0)
		}
		return nil
	}

	err := IterateObjects(m.GetDB(), model, opFunc)
	if err != nil {
		return err
	}
	// check if the object needs to be funded
	if len(multiSendMsgs) == 0 {
		logger.Info("FundingObjects: no object needs to be funded", "objectType", reflect.TypeOf(model))
		return nil
	}
	// check if the faucet balance is enough
	faucetBalance, err := m.QueryBalance(faucetAddr, config.BaseDenom)
	if err != nil {
		return err
	}
	totalInputCoin := sdktypes.NewCoin(config.BaseDenom, totalInputAmount)
	if faucetBalance.Balance.IsLT(totalInputCoin) {
		return xerrors.Errorf("insufficient faucet balance,addr:%s, need:%v,current:%v", faucetAddr, totalInputCoin, faucetBalance.Balance)
	}

	clientCtx := m.NodeClientCtx[DefaultNodeIndex]
	err = m.SignSendMultiMsgsAndWait(clientCtx, FaucetSKName, flags.BroadcastSync, multiSendMsgs...)
	if err != nil {
		return err
	}
	return nil
}

func CheckObjectsBalance[T AddressForFunding](m *Manager, model T, needExo int64) error {
	ethClient := m.NodeEVMHTTPClients[DefaultNodeIndex]
	opFunc := func(_ uint, _ int64, object T) error {
		if !object.ShouldFund() {
			// skip this object then continue the other objects
			return nil
		}
		balance, err := ethClient.BalanceAt(m.ctx, object.EvmAddress(), nil)
		if err != nil {
			return xerrors.Errorf("can't get balance,addr:%s, err:%w", object.EvmAddress().String(), err)
		}
		exoBalance := big.NewInt(0).Quo(balance, ExoDecimalReduction).Int64()
		if exoBalance < needExo {
			logger.Info("the exo balance isn't enough:", "object", object.ObjectName(), "addr", object.EvmAddress(), "exoBalance", exoBalance, "needExo", needExo)
			return xerrors.Errorf("the exo balance isn't enough, object:%s, need:%d, cur:%d", object.ObjectName(), needExo, exoBalance)
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), model, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func PaddingAddressTo32(address common.Address) []byte {
	paddingLen := 32 - len(address)
	ret := make([]byte, len(address))
	copy(ret, address[:])
	for i := 0; i < paddingLen; i++ {
		ret = append(ret, 0)
	}
	return ret
}

func PrintObject(object interface{}) error {
	objctBytes, err := json.MarshalIndent(object, " ", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(objctBytes))
	return nil
}
