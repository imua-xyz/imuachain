package common

import (
	"fmt"
	"slices"
)

type Caches struct {
	nstStakerList map[uint64][]string
}

func NewCaches() *Caches {
	return &Caches{
		nstStakerList: make(map[uint64][]string),
	}
}

func (c *Caches) ensureInitialized() {
	if c.nstStakerList == nil {
		c.nstStakerList = make(map[uint64][]string)
	}
}

func (c *Caches) GetNSTStakerList(chainID uint64) []string {
	c.ensureInitialized()
	return c.nstStakerList[chainID]
}

func (c *Caches) SetNSTStakerList(chainID uint64, sl []string) {
	c.ensureInitialized()
	c.nstStakerList[chainID] = sl
}

func (c *Caches) RemoveNSTStakerList(chainID uint64) {
	c.ensureInitialized()
	delete(c.nstStakerList, chainID)
}

func (c *Caches) RotateStakerList(chainID uint64, indexes []uint32) (map[uint32]string, error) {
	c.ensureInitialized()
	l := len(indexes)
	if l == 0 {
		return nil, nil
	}
	sl := c.nstStakerList[chainID]
	l2 := len(sl)
	if l2 == 0 {
		return nil, fmt.Errorf("cannot remove stakers from empty list for chainID %d", chainID)
	}
	// Sort indexes in ascending order so we can safely remove from the end
	slices.Sort(indexes)
	// Remove duplicates from indexes
	for i := 0; i < l-1; i++ {
		if indexes[i] == indexes[i+1] {
			indexes = slices.Delete(indexes, i, i+1)
			l--
			i--
		}
	}
	// Validate all indexes are in range
	if int(indexes[l-1]) >= l2 {
		return nil, fmt.Errorf("remove index exceeds existing max index, max:%d, remove:%d", l2-1, indexes[l-1])
	}
	// ret will map removed index to the staker that replaced it (if any)
	ret := make(map[uint32]string)
	removeMap := make(map[uint32]bool)
	for _, i := range indexes {
		removeMap[i] = true
	}
	// The main loop: for each index to remove, swap in a staker from the end (if not also being removed), then truncate
	i := 0
	j := 1
	for ; j <= l2 && j <= l; j++ {
		// Skip if the end index is also being removed
		// #nosec G115
		if int(indexes[i]) < l2-j && removeMap[uint32(l2-j)] {
			continue
		}
		// If the index to remove is now at the end, just truncate
		if int(indexes[i]) == l2-j {
			j++
			break
		}
		// If the index to remove is before the end, swap in the last staker
		if int(indexes[i]) > l2-j {
			break
		}
		ret[indexes[i]] = sl[l2-j]
		sl[indexes[i]] = sl[l2-j]
		i++
	}
	// Truncate the slice to remove the last j elements
	if j > l2 {
		j = l2
	} else if j > 0 {
		j--
	}
	sl = sl[:l2-j]
	if len(sl) == 0 {
		delete(c.nstStakerList, chainID)
	} else {
		c.nstStakerList[chainID] = sl
	}
	if len(ret) == 0 {
		ret = nil
	}
	return ret, nil
}
