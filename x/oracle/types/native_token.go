package types

const maxSize = 100

func NewStakerInfo(stakerAddr, validatorPubkey string, version, amount uint64) *StakerInfo {
	return &StakerInfo{
		StakerAddr:  stakerAddr,
		StakerIndex: 0,
		ValidatorList: []*ValidatorDeposit{
			{
				ValidatorPubkey: validatorPubkey,
				Version:         version,
				DepositAmount:   amount,
			},
		},
		BalanceList: make([]*BalanceInfo, 0, 1),
	}
}

func (s *StakerInfo) Append(b *BalanceInfo) {
	s.BalanceList = append(s.BalanceList, b)
	if len(s.BalanceList) > maxSize {
		s.BalanceList = s.BalanceList[len(s.BalanceList)-maxSize:]
	}
}

// returns: balance at the given version, latest balance, latest version, error
func (s *StakerInfo) GetBalanceAtVersion(version uint64) (uint64, uint64, uint64) {
	if len(s.BalanceList) == 0 || len(s.ValidatorList) == 0 {
		return 0, 0, 0
	}

	latestVersion := s.ValidatorList[len(s.ValidatorList)-1].Version

	latestBalance := s.BalanceList[len(s.BalanceList)-1].Balance

	if latestVersion <= version {
		return latestBalance, latestBalance, latestVersion
	}

	balance := latestBalance
	for i := len(s.ValidatorList) - 1; i >= 0; i-- {
		if s.ValidatorList[i].Version > version {
			balance -= s.ValidatorList[i].DepositAmount
		} else {
			break
		}
	}
	return balance, latestBalance, latestVersion
}
