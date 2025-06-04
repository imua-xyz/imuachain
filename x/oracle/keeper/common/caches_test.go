package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRotateStakerList(t *testing.T) {
	type result struct {
		stakerList []string
		updatedMap map[uint32]string
		errStr     string
	}
	testCases := []struct {
		name   string
		input  []string
		remove []uint32
		// expected []string
		expected *result
	}{
		{
			name:   "Rotate with 1 element removed",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{1},
			expected: &result{
				stakerList: []string{"a", "f", "c", "d", "e"},
				updatedMap: map[uint32]string{
					1: "f",
				},
				errStr: "",
			},
		},
		{
			name:   "Rotate with 2 elements removed",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{1, 3},
			expected: &result{
				stakerList: []string{"a", "f", "c", "e"},
				updatedMap: map[uint32]string{
					1: "f",
					3: "e",
				},
				errStr: "",
			},
		},
		{
			name:   "Rotate with 3 elements removed",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{1, 3, 4},
			expected: &result{
				stakerList: []string{"a", "f", "c"},
				updatedMap: map[uint32]string{
					1: "f",
				},
				errStr: "",
			},
		},
		{
			name:   "Rotate with 0 elements removed",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: nil,
			expected: &result{
				stakerList: []string{"a", "b", "c", "d", "e", "f"},
				updatedMap: nil,
				errStr:     "",
			},
		},
		{
			name:   "Rotate with all elements removed",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{0, 1, 2, 3, 4, 5},
			expected: &result{
				stakerList: nil,
				updatedMap: nil,
				errStr:     "",
			},
		},
		{
			name:   "Rotate with out of range index",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{10},
			expected: &result{
				stakerList: []string{"a", "b", "c", "d", "e", "f"},
				updatedMap: nil,
				errStr:     "remove index exceeds existing max index",
			},
		},
		{
			name:   "Rotate with duplicate indices",
			input:  []string{"a", "b", "c", "d", "e", "f"},
			remove: []uint32{1, 1, 3},
			expected: &result{
				stakerList: []string{"a", "f", "c", "e"},
				updatedMap: map[uint32]string{
					1: "f",
					3: "e",
				},
				errStr: "",
			},
		},
		{
			name:   "Rotate with empty input",
			input:  []string{},
			remove: []uint32{1, 3, 4},
			expected: &result{
				stakerList: []string{},
				updatedMap: nil,
				errStr:     "cannot remove stakers from empty list",
			},
		},
		{
			name:   "Rotate with single element input",
			input:  []string{"a"},
			remove: []uint32{0},
			expected: &result{
				stakerList: nil,
				updatedMap: nil,
				errStr:     "",
			},
		},
		{
			name:   "Rotate with single element input and no removal",
			input:  []string{"a"},
			remove: []uint32{},
			expected: &result{
				stakerList: []string{"a"},
				updatedMap: nil,
				errStr:     "",
			},
		},
		{
			name:   "Rotate with single element input and out of range removal",
			input:  []string{"a"},
			remove: []uint32{1},
			expected: &result{
				stakerList: []string{"a"},
				updatedMap: nil,
				errStr:     "remove index exceeds existing max index",
			},
		},
		{
			name:   "Rotate with single element input and duplicate removal",
			input:  []string{"a"},
			remove: []uint32{0, 0},
			expected: &result{
				stakerList: nil,
				updatedMap: nil,
				errStr:     "",
			},
		},
	}
	c := NewCaches()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//			if tc.name == "Rotate with 2 elements removed" {
			c.SetNSTStakerList(1, tc.input)
			removedmap, err := c.RotateStakerList(1, tc.remove)
			require.Equal(t, tc.expected.stakerList, c.GetNSTStakerList(1))
			require.Equal(t, tc.expected.updatedMap, removedmap)
			if err != nil {
				require.ErrorContains(t, err, tc.expected.errStr)
			}
		})
	}
}
