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

	// this line is used by starport scaffolding # genesis/types/validate
	return gs.Params.Validate()
}
