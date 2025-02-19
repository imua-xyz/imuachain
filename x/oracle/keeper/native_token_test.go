package keeper_test

import (
	"encoding/binary"
	"strings"

	sdkmath "cosmossdk.io/math"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imroc/biu"
)

// workflow:
// 1. Deposit. into staker_A
//  1. stakerInfo {totalDeposit, price} - new
//  2. stakerList - new
//
// 2. Msg. minus staker_A's amountOriginal
//  1. stakerInfo_A {price-change} - update
//  2. operatorInfo_A {price-change} - update
//
// 3. Deposit more. into staker_A
//  1. stakerInfo {totalDeposit-change} -update
//
// 4. Msg. add staker_A's amountOriginal
//  1. stakerInfo_A {price-change} - update
//  2. operatorInfo_A {price-change} - update
//
// 5. withdraw from staker_A
//  1. revmoed validatorIndex from stakerInfo
//
// 6. withdraw all from staker_A
//  1. removed stakerInfo
//  2. removed stakerList

func (ks *KeeperSuite) TestNSTLifeCycleOneStaker() {
	stakerStr := ks.StakerAddr
	NSTAssetAddr := assetstypes.GenerateNSTAddr(
		ks.ClientChains[0].AddressLength,
	)
	_, assetID := assetstypes.GetStakerIDAndAssetID(ks.ClientChains[0].LayerZeroChainID, []byte{}, NSTAssetAddr)
	validators := []string{"0xv1", "0xv2"}

	stakerID, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		ks.ClientChains[0].LayerZeroChainID,
		ks.StakerAddr, "",
	)

	// deposit amount 32e18 into stakerID(related to stakerAddr), just update kv instead of call 'deposit'
	// we separate actions to check if single module works properly
	amount32 := sdkmath.NewIntWithDecimal(32, 18)
	ks.App.AssetsKeeper.UpdateStakerAssetState(ks.Ctx, stakerID, assetID, assetstypes.DeltaStakerSingleAsset{
		TotalDepositAmount:        amount32,
		WithdrawableAmount:        amount32,
		PendingUndelegationAmount: sdk.ZeroInt(),
	})

	// 1. update oracle NST related info caused by the 'deposit amount 32e18'
	ks.App.OracleKeeper.UpdateNSTValidatorListForStaker(ks.Ctx, assetID, stakerStr, validators[0], amount32)

	// - 1.1 check stakerInfo
	stakerInfo := ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.BalanceInfo{
		Block:   1,
		RoundID: 0,
		Change:  types.Action_ACTION_DEPOSIT,
		Balance: 32,
	}, *stakerInfo.BalanceList[0])
	ks.Equal([]string{validators[0]}, stakerInfo.ValidatorPubkeyList)
	// - 1.2 check stakerList
	stakerList := ks.App.OracleKeeper.GetStakerList(ks.Ctx, assetID)
	ks.Equal(stakerList.StakerAddrs[0], stakerStr)

	// 2. Msg. minus staker's balance
	stakerChanges := [][]int{
		{0, -10},
	}
	rawData := convertBalanceChangeToBytes(stakerChanges)
	// ks.App.OracleKeeper.UpdateNSTByBalanceChange(ks.Ctx, assetID, rawData, 9)
	ks.App.OracleKeeper.UpdateNSTByBalanceChange(ks.Ctx, assetID, types.PriceTimeRound{Price: string(rawData), RoundID: 9}, 1)
	// - 2.1 check stakerInfo
	stakerInfo = ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.BalanceInfo{
		Block:   1,
		Index:   1,
		RoundID: 9,
		Change:  types.Action_ACTION_SLASH_REFUND,
		// this is expected to be 32-10=22, not 100-10
		Balance: 22,
	}, *stakerInfo.BalanceList[1])

	// stakerAssetInfo should be updated triggered by oracle module through UpdateNSTByBalanceChange
	stakerAssetInfo, _ := ks.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(ks.Ctx, stakerID, assetID)
	amount22 := sdkmath.NewIntWithDecimal(22, 18)
	ks.Equal(assetstypes.StakerAssetInfo{
		TotalDepositAmount:        amount22,
		WithdrawableAmount:        amount22,
		PendingUndelegationAmount: sdk.ZeroInt(),
	}, *stakerAssetInfo)

	// 3. deposit more. 32e18
	// update assetInfo first, should update the staker's totalAmount,withdrawableAmount to 54e18
	ks.App.AssetsKeeper.UpdateStakerAssetState(ks.Ctx, stakerID, assetID, assetstypes.DeltaStakerSingleAsset{
		TotalDepositAmount:        amount32,
		WithdrawableAmount:        amount32,
		PendingUndelegationAmount: sdk.ZeroInt(),
	})
	// check stakerAssetInfo is updated correctly in assets module
	stakerAssetInfo, _ = ks.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(ks.Ctx, stakerID, assetID)
	amount54 := sdkmath.NewIntWithDecimal(54, 18)
	ks.Equal(assetstypes.StakerAssetInfo{
		TotalDepositAmount:        amount54,
		WithdrawableAmount:        amount54,
		PendingUndelegationAmount: sdk.ZeroInt(),
	}, *stakerAssetInfo)

	// update related infor in oracle module causd by new deposit NST
	ks.App.OracleKeeper.UpdateNSTValidatorListForStaker(ks.Ctx, assetID, stakerStr, validators[1], amount32) // 999
	// - 3.1 check stakerInfo
	stakerInfo = ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.BalanceInfo{
		Block:   1,
		RoundID: 9,
		Index:   2,
		Change:  types.Action_ACTION_DEPOSIT,
		Balance: 54,
	}, *stakerInfo.BalanceList[2])
	ks.Equal(validators, stakerInfo.ValidatorPubkeyList)

	// 4. Msg. add staker's balance
	// at this point the system correct number should be 32*2-10 = 54, if some validator do refund, means the delta should be less than 10
	stakerChanges = [][]int{
		// means delta from -10 change to -5
		{0, -5},
	}
	rawData = convertBalanceChangeToBytes(stakerChanges)
	ks.App.OracleKeeper.UpdateNSTByBalanceChange(ks.Ctx, assetID, types.PriceTimeRound{Price: string(rawData), RoundID: 11}, 2)
	// - 4.1 check stakerInfo
	stakerInfo = ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.BalanceInfo{
		Balance: 59,
		Block:   1,
		RoundID: 11,
		Index:   3,
		Change:  types.Action_ACTION_SLASH_REFUND,
	}, *stakerInfo.BalanceList[3])
	// check stakerAssetInfo is updated correctly in assets module, this should be triggered in assets module by oracle module's UpdateNSTByBalanceChange
	stakerAssetInfo, _ = ks.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(ks.Ctx, stakerID, assetID)
	amount59 := sdkmath.NewIntWithDecimal(59, 18)
	ks.Equal(assetstypes.StakerAssetInfo{
		TotalDepositAmount:        amount59,
		WithdrawableAmount:        amount59,
		PendingUndelegationAmount: sdk.ZeroInt(),
	}, *stakerAssetInfo)

	// 5. withdraw
	amount30N := sdkmath.NewIntWithDecimal(-30, 18)
	ks.App.OracleKeeper.UpdateNSTValidatorListForStaker(ks.Ctx, assetID, stakerStr, validators[0], amount30N)
	// - 5.1 check stakerInfo
	stakerInfo = ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.BalanceInfo{
		Balance: 29,
		Block:   1,
		RoundID: 11,
		Index:   4,
		Change:  types.Action_ACTION_WITHDRAW,
	}, *stakerInfo.BalanceList[4])
	// withdraw will remove this validator
	ks.Equal([]string{validators[1]}, stakerInfo.ValidatorPubkeyList)

	// 6.withdrawall
	amount29N := sdkmath.NewIntWithDecimal(-29, 18)
	ks.App.OracleKeeper.UpdateNSTValidatorListForStaker(ks.Ctx, assetID, stakerStr, validators[1], amount29N)
	// - 6.1 check stakerInfo
	stakerInfo = ks.App.OracleKeeper.GetStakerInfo(ks.Ctx, assetID, stakerStr)
	ks.Equal(types.StakerInfo{}, stakerInfo)
	// - 6.2 check stakerList
	stakerList = ks.App.OracleKeeper.GetStakerList(ks.Ctx, assetID)
}

func convertBalanceChangeToBytes(stakerChanges [][]int) []byte {
	if len(stakerChanges) == 0 {
		return nil
	}
	str := ""
	index := 0
	changeBytesList := make([][]byte, 0, len(stakerChanges))
	bitsList := make([]int, 0, len(stakerChanges))
	for _, stakerChange := range stakerChanges {
		str += strings.Repeat("0", stakerChange[0]-index) + "1"
		index = stakerChange[0] + 1

		// change amount -> bytes
		change := stakerChange[1]
		var changeBytes []byte
		symbol := 1
		if change < 0 {
			symbol = -1
			change *= -1
		}
		change--
		bits := 0
		if change == 0 {
			bits = 1
			changeBytes = []byte{byte(0)}
		} else {
			tmpChange := change
			for tmpChange > 0 {
				bits++
				tmpChange /= 2
			}
			if change < 256 {
				// 1 byte
				changeBytes = []byte{byte(change)}
				changeBytes[0] <<= (8 - bits)
			} else {
				// 2 byte
				changeBytes = make([]byte, 2)
				binary.BigEndian.PutUint16(changeBytes, uint16(change))
				moveLength := 16 - bits
				changeBytes[0] <<= moveLength
				tmp := changeBytes[1] >> (8 - moveLength)
				changeBytes[0] |= tmp
				changeBytes[1] <<= moveLength
			}
		}

		// use lower 4 bits to represent the length of valid change value in bits format
		bitsLengthBytes := []byte{byte(bits)}
		bitsLengthBytes[0] <<= 4
		if symbol < 0 {
			bitsLengthBytes[0] |= 8
		}

		tmp := changeBytes[0] >> 5
		bitsLengthBytes[0] |= tmp
		if bits <= 3 {
			changeBytes = nil
		} else {
			changeBytes[0] <<= 3
		}

		if len(changeBytes) == 2 {
			tmp = changeBytes[1] >> 5
			changeBytes[0] |= tmp
			if bits <= 11 {
				changeBytes = changeBytes[:1]
			} else {
				changeBytes[1] <<= 3
			}
		}
		bitsLengthBytes = append(bitsLengthBytes, changeBytes...)
		changeBytesList = append(changeBytesList, bitsLengthBytes)
		bitsList = append(bitsList, bits)
	}

	l := len(bitsList)
	changeResult := changeBytesList[l-1]
	bitsList[len(bitsList)-1] = bitsList[len(bitsList)-1] + 5
	for i := l - 2; i >= 0; i-- {
		prev := changeBytesList[i]

		byteLength := 8 * len(prev)
		bitsLength := bitsList[i] + 5
		// delta must <8
		delta := byteLength - bitsLength
		if delta == 0 {
			changeResult = append(prev, changeResult...)
			bitsList[i] = bitsLength + bitsList[i+1]
		} else {
			// delta : (0,8)
			tmp := changeResult[0] >> (8 - delta)
			prev[len(prev)-1] |= tmp
			if len(changeResult) > 1 {
				for j := 1; j < len(changeResult); j++ {
					changeResult[j-1] <<= delta
					tmp := changeResult[j] >> (8 - delta)
					changeResult[j-1] |= tmp
				}
			}
			changeResult[len(changeResult)-1] <<= delta
			left := bitsList[i+1] % 8
			if bitsList[i+1] > 0 && left == 0 {
				left = 8
			}
			if left <= delta {
				changeResult = changeResult[:len(changeResult)-1]
			}
			changeResult = append(prev, changeResult...)
			bitsList[i] = bitsLength + bitsList[i+1]
		}
	}
	str += strings.Repeat("0", 256-index)
	bytesIndex := biu.BinaryStringToBytes(str)

	result := append(bytesIndex, changeResult...)
	return result
}
