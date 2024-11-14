package types

import (
	"fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		PricesList:           []Prices{},
		ValidatorUpdateBlock: nil,
		IndexRecentParams:    nil,
		IndexRecentMsg:       nil,
		RecentMsgList:        []RecentMsg{},
		RecentParamsList:     []RecentParams{},
		Params:               DefaultParams(),
		StakerInfosAssets:    []StakerInfosAssets{},
		StakerListAssets:     []StakerListAssets{},
		// this line is used by starport scaffolding # genesis/types/default
	}
}

func NewGenesisState(p Params) *GenesisState {
	return &GenesisState{
		PricesList:           []Prices{},
		ValidatorUpdateBlock: nil,
		IndexRecentParams:    nil,
		IndexRecentMsg:       nil,
		RecentMsgList:        []RecentMsg{},
		RecentParamsList:     []RecentParams{},
		Params:               p,
	}
}

func NewMissedRound(index uint64, missed bool) *MissedRound {
	return &MissedRound{
		Index:  index,
		Missed: missed,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in prices
	pricesIndexMap := make(map[string]struct{})

	for _, elem := range gs.PricesList {
		index := string(PricesKey(elem.TokenID))
		if _, ok := pricesIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for prices")
		}
		pricesIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in recentMsg
	recentMsgIndexMap := make(map[string]struct{})

	for _, elem := range gs.RecentMsgList {
		index := string(RecentMsgKey(elem.Block))
		if _, ok := recentMsgIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for recentMsg")
		}
		recentMsgIndexMap[index] = struct{}{}
	}
	// Check for duplicated index in recentParams
	recentParamsIndexMap := make(map[string]struct{})

	for _, elem := range gs.RecentParamsList {
		index := string(RecentParamsKey(elem.Block))
		if _, ok := recentParamsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for recentParams")
		}
		recentParamsIndexMap[index] = struct{}{}
	}

	// Check for stakerInfo length matches with stakerlist
	if len(gs.StakerListAssets) != len(gs.StakerInfosAssets) {
		return fmt.Errorf("length not equal for stakerListAssets and stakerInfosAssets")
	}

	for _, stakerInfosAsset := range gs.StakerInfosAssets {
		notFound := true
		for _, StakerListAsset := range gs.StakerListAssets {
			if StakerListAsset.AssetId == stakerInfosAsset.AssetId {
				notFound = false
				if len(StakerListAsset.StakerList.StakerAddrs) != len(stakerInfosAsset.StakerInfos) {
					return fmt.Errorf("length not equal for stakerListAsset and StakerInfosAsset of assetID:%s", StakerListAsset.AssetId)
				}
				stakerListIdx := make(map[string]int)
				for idx, staker := range StakerListAsset.StakerList.StakerAddrs {
					if _, ok := stakerListIdx[staker]; ok {
						return fmt.Errorf("duplicated staker in stakerList for assetID:%s", StakerListAsset.AssetId)
					}
					stakerListIdx[staker] = idx
				}
				for _, stakerInfo := range stakerInfosAsset.StakerInfos {
					if idx, ok := stakerListIdx[stakerInfo.StakerAddr]; !ok {
						return fmt.Errorf("staker %s from stakerInfo not exsists in stakerList for assetID:%s", stakerInfo.StakerAddr, StakerListAsset.AssetId)
					} else if idx != int(stakerInfo.StakerIndex) {
						return fmt.Errorf("staker %s from stakerInfo has index %d, not match which from stakerList %d", stakerInfo.StakerAddr, stakerInfo.StakerIndex, idx)
					}
				}
			}
		}
		if notFound {
			return fmt.Errorf("assetID %s in stakerInfosAssets not found in stakerLisetAssets", stakerInfosAsset.AssetId)
		}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
