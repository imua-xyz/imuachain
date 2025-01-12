package batch

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ExocoreNetwork/exocore/testutil"

	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"

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

// InitSequences : Fetches the address sequences of the faucet, operators, and AVSs,
// and then saves them to the sync.Map. These sequences can be used in subsequent tests.
// This function should be called after all objects have been created.
func (m *Manager) InitSequences() error {
	defaultClientCtx := m.NodeClientCtx[DefaultNodeIndex]
	// set the address sequence of faucet
	_, seq, err := defaultClientCtx.AccountRetriever.GetAccountNumberSequence(defaultClientCtx, crypto.PubkeyToAddress(m.FaucetSK.PublicKey).Bytes())
	if err != nil {
		logger.Error("InitSequences, faucet address doesn't exist", "evmAddr", crypto.PubkeyToAddress(m.FaucetSK.PublicKey), "err", err)
		return err
	}
	m.Sequences.Store(crypto.PubkeyToAddress(m.FaucetSK.PublicKey), seq)

	// set the address sequences of all operators
	opOperatorFunc := func(_ uint, _ int64, operator Operator) error {
		if operator.IsDefaultOperator() {
			// skip the default operator then continue the other objects
			return nil
		}
		// set the address sequence of test operator
		seq = uint64(0)
		_, seq, err = defaultClientCtx.AccountRetriever.GetAccountNumberSequence(defaultClientCtx, operator.AccAddress())
		if err != nil {
			// set 0 as the sequence if the address doesn't exist
			logger.Info("the operator address doesn't exist, set 0 as the sequence", "operator", operator.Name, "accAddr", operator.AccAddress(), "err", err)
		}
		m.Sequences.Store(operator.EvmAddress(), seq)
		return nil
	}
	err = IterateObjects(m.GetDB(), Operator{}, opOperatorFunc)
	if err != nil {
		return err
	}

	// set the address sequences of all AVSs
	opAVSFunc := func(_ uint, _ int64, avs AVS) error {
		if avs.IsDogfood() {
			// skip the dogfood then continue the other AVSs
			return nil
		}
		// set the address sequence of test avs
		seq = uint64(0)
		_, seq, err = defaultClientCtx.AccountRetriever.GetAccountNumberSequence(defaultClientCtx, avs.AccAddress())
		if err != nil {
			// set 0 as the sequence if the address doesn't exist
			logger.Info("the avs address doesn't exist, set 0 as the sequence", "avs", avs.Name, "accAddr", avs.AccAddress(), "err", err)
		}
		m.Sequences.Store(avs.EvmAddress(), seq)
		return nil
	}
	err = IterateObjects(m.GetDB(), AVS{}, opAVSFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) LoadSequence(addr common.Address) (uint64, error) {
	if value, ok := m.Sequences.Load(addr); ok {
		return value.(uint64), nil
	}
	return 0, xerrors.Errorf("can't load the sequence from the sync map, addr:%s", addr)
}

func (m *Manager) FundAndCheckStakers() error {
	logger.Info("start funding stakers")
	err := FundingObjects(m, &Staker{}, m.config.StakerExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund stakers,err:%w", err)
	}
	time.Sleep(time.Duration(m.config.BatchTxsCheckInterval) * time.Second)
	err = CheckObjectsBalance(m, &Staker{}, m.config.StakerExoAmount)
	if err != nil {
		return err
	}
	return nil
}

// Funding : send Exo token to the test objects, which can be used for the tx fee in the next tests.
// all Exo token is sent from the faucet sk.
func (m *Manager) Funding() error {
	logger.Info("start funding stakers")
	err := FundingObjects(m, &Staker{}, m.config.StakerExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund stakers,err:%w", err)
	}
	logger.Info("start funding operators")
	err = FundingObjects(m, &Operator{}, m.config.OperatorExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund operators,err:%w", err)
	}
	logger.Info("start funding AVSs")
	err = FundingObjects(m, &AVS{}, m.config.AVSExoAmount)
	if err != nil {
		return xerrors.Errorf("can't fund AVSs,err:%w", err)
	}
	return nil
}

func (m *Manager) AssetsCheck(opFuncIfCheckFail func(assetID string, asset *Asset) error) error {
	queryClient := assetstypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	opFunc := func(_ uint, _ int64, asset *Asset) error {
		// check if the asset has been registered.
		_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
			uint64(asset.ClientChainID), "", asset.Address.String())
		req := &assetstypes.QueryStakingAssetInfo{
			AssetId: assetID, // already lowercase
		}
		_, err := queryClient.QueStakingAssetInfo(m.ctx, req)
		if err != nil {
			// register the asset.
			if opFuncIfCheckFail != nil {
				err = opFuncIfCheckFail(assetID, asset)
				if err != nil {
					return err
				}
			} else {
				return xerrors.Errorf("the asset hasn't been registered,name:%s,addr:%s", asset.Name, asset.Address)
			}
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &Asset{}, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) RegisterAssets() ([]string, error) {
	assetsAbi, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return nil, err
	}

	allAssetsID := make([]string, 0)
	opFuncIfCheckFail := func(assetID string, asset *Asset) error {
		// register the asset.
		data, err := assetsAbi.Pack(assets.MethodRegisterToken, asset.ClientChainID, PaddingAddressTo32(asset.Address), asset.Decimal, asset.Name, asset.MetaInfo, asset.OracleInfo)
		if err != nil {
			return err
		}

		err = m.SignSendEvmTxAndWait(&EvmTxInQueue{
			From:   crypto.PubkeyToAddress(m.FaucetSK.PublicKey),
			ToAddr: &AssetsPrecompileAddr,
			Value:  big.NewInt(0),
			Data:   data,
			Sk:     m.FaucetSK,
		})
		if err != nil {
			return err
		}
		allAssetsID = append(allAssetsID, assetID)
		return nil
	}
	err = m.AssetsCheck(opFuncIfCheckFail)
	if err != nil {
		return nil, err
	}
	return allAssetsID, nil
}

func (m *Manager) AVSsCheck(opFuncIfCheckFail func(id uint, avs *AVS) error) error {
	queryClient := avstypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	opFunc := func(id uint, _ int64, avs *AVS) error {
		if avs.IsDogfood() {
			// skip the dogfood AVS and continue addressing the other AVSs
			return nil
		}
		// check if the avs has been registered.
		req := &avstypes.QueryAVSInfoReq{
			AVSAddress: avs.Address,
		}
		_, err := queryClient.QueryAVSInfo(m.ctx, req)
		if err != nil {
			// register the AVS.
			if opFuncIfCheckFail != nil {
				err = opFuncIfCheckFail(id, avs)
				if err != nil {
					return err
				}
			} else {
				return xerrors.Errorf("the AVS hasn't been registered,name:%s,addr:%s", avs.Name, avs.Address)
			}
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &AVS{}, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) RegisterAVSs(allAssetsID []string) error {
	avsAbi, err := abi.JSON(strings.NewReader(avsprecompile.AvsABI))
	if err != nil {
		return err
	}
	minStakeAmount := uint64(3)
	minSelfDelegation := uint64(0)
	params := []uint64{2, 3, 4, 4}

	opFuncIfCheckFail := func(id uint, avs *AVS) error {
		// register the AVS.
		name := fmt.Sprintf("%s%d", AVSNamePrefix, id)
		epochIdentifier := testutil.EpochsForTest[int(id-1)%len(testutil.EpochsForTest)]
		avsUnbondingPeriod := uint64(id) % MaxUnbondingDuration
		data, err := avsAbi.Pack(
			avsprecompile.MethodRegisterAVS,
			avs.EvmAddress(),
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
		sk, err := crypto.ToECDSA(avs.Sk)
		if err != nil {
			return xerrors.Errorf("can't convert the Sk to ecdsa private key,avs:%v,err:%w", avs, err)
		}
		logger.Info("the caller and AVS address is:", "caller", crypto.PubkeyToAddress(sk.PublicKey), "avsAddr", avs.EvmAddress())
		err = m.SignSendEvmTxAndWait(&EvmTxInQueue{
			From:   crypto.PubkeyToAddress(sk.PublicKey),
			ToAddr: &AVSPrecompileAddr,
			Value:  big.NewInt(0),
			Data:   data,
			Sk:     sk,
		})
		if err != nil {
			return err
		}
		return nil
	}
	err = m.AVSsCheck(opFuncIfCheckFail)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) OperatorsCheck(opFuncIfCheckFail func(operator *Operator) error) error {
	queryClient := operatortypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	opFunc := func(_ uint, _ int64, operator *Operator) error {
		if operator.IsDefaultOperator() {
			// skip the default operator and continue addressing the test operators
			return nil
		}
		// check if the operator has been registered.
		req := &operatortypes.GetOperatorInfoReq{
			OperatorAddr: operator.Address,
		}
		_, err := queryClient.QueryOperatorInfo(m.ctx, req)
		if err != nil {
			// register the operator.
			if opFuncIfCheckFail != nil {
				err = opFuncIfCheckFail(operator)
				if err != nil {
					return err
				}
			} else {
				return xerrors.Errorf("the operator hasn't been registered,name:%s,addr:%s", operator.Name, operator.Address)
			}
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &Operator{}, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) RegisterOperators() error {
	opFuncIfCheckFail := func(operator *Operator) error {
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
		err := m.SignSendMultiMsgsAndWait(clientCtx, operator.Name, flags.BroadcastSync, msg)
		if err != nil {
			return err
		}
		return nil
	}
	err := m.OperatorsCheck(opFuncIfCheckFail)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) DogfoodAssetsCheck(allAssetsID []string) (*dogfoodtypes.Params, bool, error) {
	clientCtx := m.NodeClientCtx[DefaultNodeIndex]
	queryClient := dogfoodtypes.NewQueryClient(clientCtx)

	res, err := queryClient.Params(m.ctx, &dogfoodtypes.QueryParamsRequest{})
	if err != nil {
		return nil, false, err
	}
	// Create a set (using a map) to keep track of existing AssetIDs in res.Params.AssetIDs
	existingAssets := make(map[string]struct{})

	// Populate the map with the current AssetIDs
	for _, id := range res.Params.AssetIDs {
		existingAssets[id] = struct{}{}
	}

	// Iterate through allAssetsID to find any missing IDs
	isUpdateParam := false
	for _, id := range allAssetsID {
		if _, exists := existingAssets[id]; !exists {
			// If the ID is missing, add it to the existing AssetIDs
			res.Params.AssetIDs = append(res.Params.AssetIDs, id)
			isUpdateParam = true
		}
	}
	return &res.Params, isUpdateParam, nil
}

func (m *Manager) AddAssetsToDogfoodAVS(allAssetsID []string) error {
	updatedParam, isUpdate, err := m.DogfoodAssetsCheck(allAssetsID)
	if err != nil {
		return err
	}
	if isUpdate {
		record, err := m.KeyRing.Key(FaucetSKName)
		if err != nil {
			return err
		}
		// Retrieve the address from the Record
		address, err := record.GetAddress()
		if err != nil {
			return err
		}
		msg := &dogfoodtypes.MsgUpdateParams{
			Authority: address.String(),
			Params:    *updatedParam,
		}
		clientCtx := m.NodeClientCtx[DefaultNodeIndex]
		err = m.SignSendMultiMsgsAndWait(clientCtx, FaucetSKName, flags.BroadcastSync, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) OperatorsOptInCheck(opFuncIfCheckFail func(operator *Operator, avs *AVS) error) error {
	queryClient := operatortypes.NewQueryClient(m.NodeClientCtx[DefaultNodeIndex])
	operatorOpFunc := func(_ uint, _ int64, operator *Operator) error {
		if operator.IsDefaultOperator() {
			// skip the default operator and continue addressing the test operators
			// todo: the related private key need to be imported if we want to address the default operator
			return nil
		}
		avsOpFunc := func(_ uint, _ int64, avs *AVS) error {
			if avs.IsDogfood() {
				// skip the dogfood AVS and continue addressing the other AVSs
				// todo: the test operators need to launch the Exocore node if they want to opt into the dogfood
				return nil
			}
			// check if the operator has been registered.
			req := &operatortypes.QueryOptInfoRequest{
				OperatorAVSAddress: &operatortypes.OperatorAVSAddress{
					OperatorAddr: operator.Address,
					AvsAddress:   avs.Address,
				},
			}
			optInfo, err := queryClient.QueryOptInfo(m.ctx, req)
			if err != nil || optInfo.OptedOutHeight != operatortypes.DefaultOptedOutHeight {
				// opts the operator into the AVS
				if opFuncIfCheckFail != nil {
					err = opFuncIfCheckFail(operator, avs)
					if err != nil {
						return err
					}
				} else {
					return xerrors.Errorf("the operator hasn't been opted into the AVS,operatorName:%s,operatorAddr:%s,AVSName:%s,AVSAddr:%s", operator.Name, operator.Address, avs.Name, avs.Address)
				}
			}
			return nil
		}
		err := IterateObjects(m.GetDB(), &AVS{}, avsOpFunc)
		if err != nil {
			return err
		}
		return nil
	}
	err := IterateObjects(m.GetDB(), &Operator{}, operatorOpFunc)
	if err != nil {
		return err
	}
	return nil
}

// OptOperatorsIntoAVSs opts all operators into all AVSs for the test.
func (m *Manager) OptOperatorsIntoAVSs() error {
	clientCtx := m.NodeClientCtx[DefaultNodeIndex]
	opFuncIfCheckFail := func(operator *Operator, avs *AVS) error {
		// opts the operator into the AVS
		msg := &operatortypes.OptIntoAVSReq{
			FromAddress: operator.Address,
			AvsAddress:  avs.Address,
		}
		if avs.IsDogfood() {
			msg.PublicKeyJSON = operator.ConsensusPubKey
		}
		err := m.SignSendMultiMsgsAndWait(clientCtx, operator.Name, flags.BroadcastSync, msg)
		if err != nil {
			return err
		}
		return nil
	}
	err := m.OperatorsOptInCheck(opFuncIfCheckFail)
	if err != nil {
		return err
	}
	return nil
}
