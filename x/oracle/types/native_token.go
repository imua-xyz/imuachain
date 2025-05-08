package types

const maxSize = 100

func (b *Balances) Append(bi *BalanceInfo) {
	if len(b.BalanceList) >= maxSize {
		b.BalanceList = b.BalanceList[1:]
	}
	b.BalanceList = append(b.BalanceList, bi)
}

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
	if len(s.BalanceList) >= maxSize {
		s.BalanceList = s.BalanceList[len(s.BalanceList)-maxSize:]
	}
	s.BalanceList = append(s.BalanceList, b)
}
