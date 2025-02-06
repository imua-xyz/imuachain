//go:build local

package feedermanagement

import (
	"fmt"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

// this test elaborate all combinations for {4validators, 3maxNonce, 3detIDs}
func TestRoundTallyElaborate(t *testing.T) {
	ret := generateAllValidatorSets(prices, []string{"v1", "v2", "v3", "v4"})
	tests := generateAllBlocks(ret, 3, 4)
	ctrl := gomock.NewController(t)
	c := NewMockCacheReader(ctrl)
	c.EXPECT().
		GetPowerForValidator(gomock.Any()).
		Return(big1, true).
		AnyTimes()
	c.EXPECT().
		IsDeterministic(gomock.Eq(int64(1))).
		Return(true, nil).
		AnyTimes()
	c.EXPECT().
		GetThreshold().
		Return(th).
		AnyTimes()
	c.EXPECT().
		IsRuleV1(gomock.Any()).
		Return(true).
		AnyTimes()

	total := len(tests)
	for idx, tt := range tests {
		fmt.Printf("total:%d, running:%d\r\n", total, idx)
		t.Run(fmt.Sprintf("case_%d", idx), func(t *testing.T) {
			r := tData.NewRoundWithFeederID(c, 1)
			r.cache = c
			r.PrepareForNextBlock(int64(params.TokenFeeders[1].StartBaseBlock))
			p, rslt := tt.Next()
			for p != nil {
				var pRslt *PriceResult
				for _, pItem := range p {
					if pItem == nil {
						continue
					}
					if pRsltTmp, _, _ := r.Tally(pItem.msgItem()); pRsltTmp != nil {
						pRslt = pRsltTmp
					}
				}
				if rslt != nil && (pRslt == nil || rslt.Price != pRslt.Price) {
					tt.Reset()
					fmt.Println("fail case:", idx, "Tally.result:", pRslt)
					p, rslt := tt.Next()
					idx2 := 1
					for p != nil {
						fmt.Printf("block_%d ", idx2)
						idx2++
						for idx3, pi := range p {
							fmt.Printf("msgItem_%d: %v ", idx3, pi)
						}
						fmt.Println()
						fmt.Println(rslt)
						p, rslt = tt.Next()
					}
					t.Fatal("failed")
				}
				if rslt != nil {
					break
				}
				p, rslt = tt.Next()
			}
		})
	}
}
