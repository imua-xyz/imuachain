package types

const maxSize = 100

func (b *Balances) Append(bi *BalanceInfo) {
	if len(b.BalanceList) >= maxSize {
		b.BalanceList = b.BalanceList[1:]
	}
	b.BalanceList = append(b.BalanceList, bi)
}
