package types

const (
	maxSize           = 100
	WithdrawValidator = "0xFFFFFFFFFFFFFFFF"
)

func (b *Balances) Append(bi *BalanceInfo) {
	if len(b.BalanceList) >= maxSize {
		b.BalanceList = b.BalanceList[1:]
	}
	b.BalanceList = append(b.BalanceList, bi)
}

// NOTE: This function assumes that the BalanceList/valdiatorList is sorted by version in ascending order.
// returns: balance at the given version, latest balance, latest version
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
