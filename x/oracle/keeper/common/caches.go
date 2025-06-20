package common

import (
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

type Caches struct {
	nstStakerList map[uint64][]*types.StakerListEntry
}

func NewCaches() *Caches {
	return &Caches{
		nstStakerList: make(map[uint64][]*types.StakerListEntry),
	}
}

func (c *Caches) ensureInitialized() {
	if c.nstStakerList == nil {
		c.nstStakerList = make(map[uint64][]*types.StakerListEntry)
	}
}

func (c *Caches) GetNSTStakerList(chainID uint64) []*types.StakerListEntry {
	c.ensureInitialized()
	return c.nstStakerList[chainID]
}

func (c *Caches) SetNSTStakerList(chainID uint64, sl []*types.StakerListEntry) {
	c.ensureInitialized()
	c.nstStakerList[chainID] = sl
}

func (c *Caches) UpdateWithdrawVersion(chainID uint64, stakerAddr string, index uint32, withdrawVersion uint64) bool {
	c.ensureInitialized()
	sl := c.nstStakerList[chainID]
	if int(index) >= len(sl) {
		return false
	}
	if sl[index].StakerAddr != stakerAddr {
		return false
	}
	sl[index].WithdrawVersion = withdrawVersion
	return true
}

// AddNSTStaker adds a staker to the list for the given chainID at the specified index.
// NOTE: not concurrent safe, caller must ensure synchronization.
func (c *Caches) AddNSTStaker(chainID uint64, staker types.StakerListEntry, index uint32) bool {
	if c.nstStakerList == nil {
		if index > 0 {
			return false
		}
		c.nstStakerList = make(map[uint64][]*types.StakerListEntry)
	}
	sl := c.nstStakerList[chainID]
	if int(index) != len(sl) {
		return false
	}
	c.nstStakerList[chainID] = append(sl, &staker)
	return true
}
