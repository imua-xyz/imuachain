package batch

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ExocoreNetwork/exocore/precompiles/assets"
	avsprecompile "github.com/ExocoreNetwork/exocore/precompiles/avs"
	"github.com/ExocoreNetwork/exocore/testutil/tx"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/xerrors"
)

func (m *Manager) Funding() error {
	err := FundingObjects(m, &Staker{}, m.config.StakerExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund stakers,err:%s", err)
	}
	err = FundingObjects(m, &Operator{}, m.config.OperatorExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund operators,err:%s", err)
	}
	err = FundingObjects(m, &AVS{}, m.config.AVSExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund AVSs,err:%s", err)
	}
	return nil
}

func (m *Manager) RegisterAssets() ([]string, error) {
	queryClient := assetstypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	assetsAbi, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return nil, err
	}

	allAssetsID := make([]string, 0)
	opFunc := func(_ uint, _ int64, asset *Asset) error {
		// check if the asset has been registered.
		_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
			asset.ClientChainID, "", asset.Address.String())
		req := &assetstypes.QueryStakingAssetInfo{
			AssetID: assetID, // already lowercase
		}
		_, err := queryClient.QueStakingAssetInfo(m.ctx, req)
		if err != nil {
			// register the asset.
			data, err := assetsAbi.Pack(assets.MethodRegisterToken, asset.ClientChainID, asset.Address, asset.Decimal, asset.Name, asset.MetaInfo, asset.OracleInfo)
			if err != nil {
				return err
			}

			err = SignSendEvmTxAndWait(m.DefaultEvmTxRequirements, &EvmTxInQueue{
				ToAddr: &AssetsPrecompileAddr,
				Value:  big.NewInt(0),
				Data:   data,
			})
			if err != nil {
				return err
			}
		}
		allAssetsID = append(allAssetsID, assetID)
		return nil
	}
	err = IterateObjects(m, &Asset{}, opFunc)
	if err != nil {
		return nil, err
	}
	return allAssetsID, nil
}

func (m *Manager) RegisterAVSs(allAssetsID []string) error {
	queryClient := avstypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	avsAbi, err := abi.JSON(strings.NewReader(avsprecompile.AvsABI))
	if err != nil {
		return err
	}
	minStakeAmount := uint64(3)
	minSelfDelegation := uint64(0)
	params := []uint64{2, 3, 4, 4}
	opFunc := func(id uint, _ int64, avs *AVS) error {
		// check if the avs has been registered.
		req := &avstypes.QueryAVSInfoReq{
			AVSAddress: strings.ToLower(avs.Address.String()),
		}
		_, err := queryClient.QueryAVSInfo(m.ctx, req)
		if err != nil {
			// register the AVS.
			name := fmt.Sprintf("%s%d", AVSNamePrefix, id)
			epochIdentifier := AllEpochs[int(id-1)%len(AllEpochs)]
			avsUnbondingPeriod := uint64(id) % MaxUnbondingDuration
			data, err := avsAbi.Pack(
				avsprecompile.MethodRegisterAVS,
				avs.Address,
				name,
				minStakeAmount,
				tx.GenerateAddress(),
				tx.GenerateAddress(),
				tx.GenerateAddress(),
				[]string{avs.AccAddress().String()},
				allAssetsID,
				avsUnbondingPeriod,
				minSelfDelegation,
				epochIdentifier,
				params,
			)
			if err != nil {
				return err
			}
			err = SignSendEvmTxAndWait(m.DefaultEvmTxRequirements, &EvmTxInQueue{
				ToAddr: &AssetsPrecompileAddr,
				Value:  big.NewInt(0),
				Data:   data,
			})
			if err != nil {
				return err
			}
		}
		return nil
	}
	err = IterateObjects(m, &AVS{}, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) RegisterOperators() error {
	queryClient := operatortypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	opFunc := func(_ uint, _ int64, operator *Operator) error {
		// check if the operator has been registered.
		req := &operatortypes.GetOperatorInfoReq{
			OperatorAddr: operator.Address,
		}
		_, err := queryClient.QueryOperatorInfo(m.ctx, req)
		if err != nil {
			// register operator
			metaInfo := fmt.Sprintf("the meta info of %s", operator.Name)
			msg := &operatortypes.RegisterOperatorReq{
				FromAddress: operator.Address,
				Info: &operatortypes.OperatorInfo{
					EarningsAddr:     operator.Address,
					ApproveAddr:      operator.Address,
					OperatorMetaInfo: metaInfo,
					Commission:       DefaultOperatorCommission,
				},
			}
			clientCtx := m.NodeClientCtx[DefaultNodeIndex]
			err = SignAndSendMultiMsgs(clientCtx, operator.Name, flags.BroadcastSync, msg)
			if err != nil {
				return err
			}
		}
		return nil
	}
	err := IterateObjects(m, &Operator{}, opFunc)
	if err != nil {
		return err
	}
	return nil
}

// OptOperatorsIntoAVSs opts all operators into all AVSs for the test.
func (m *Manager) OptOperatorsIntoAVSs() error {
	queryClient := operatortypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	operatorOpFunc := func(_ uint, _ int64, operator *Operator) error {
		avsOpFunc := func(_ uint, _ int64, avs *AVS) error {
			avsAddrStr := strings.ToLower(avs.Address.String())
			// check if the operator has been registered.
			req := &operatortypes.QueryOptInfoRequest{
				OperatorAVSAddress: &operatortypes.OperatorAVSAddress{
					OperatorAddr: operator.Address,
					AvsAddress:   avsAddrStr,
				},
			}
			optInfo, err := queryClient.QueryOptInfo(m.ctx, req)
			if err != nil || optInfo.OptedOutHeight == operatortypes.DefaultOptedOutHeight {
				// register operator
				msg := &operatortypes.OptIntoAVSReq{
					FromAddress:   operator.Address,
					AvsAddress:    avsAddrStr,
					PublicKeyJSON: operator.ConsensusPubKey,
				}
				clientCtx := m.NodeClientCtx[DefaultNodeIndex]
				err = SignAndSendMultiMsgs(clientCtx, operator.Name, flags.BroadcastSync, msg)
				if err != nil {
					return err
				}
			}
			return nil
		}
		err := IterateObjects(m, &AVS{}, avsOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m, &Operator{}, operatorOpFunc)
	if err != nil {
		return err
	}
	return nil
}
