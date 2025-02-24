package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerkleTree(t *testing.T) {
	fmt.Println("test on 6 pieces")
	test6pieces()

	fmt.Println("test on 5 pieces")
	test5pieces()

	fmt.Println("get proofPath from 6 pieces")
	m := NewMT(20, 120)
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(0), []uint32{1, 7, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(1), []uint32{0, 7, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(2), []uint32{3, 6, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(3), []uint32{2, 6, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(4), []uint32{5, 9})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(5), []uint32{4, 9})

	fmt.Println("get proofPath from 5 pieces")
	m = NewMT(20, 100)
	// if limit the provider to upload pieces ordered, then with cache, we only need proof:
	// (1,6,4), (-), (3), (2)
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(0), []uint32{1, 6, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(1), []uint32{0, 6, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(2), []uint32{3, 5, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(3), []uint32{2, 5, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(4), []uint32{8})
}

func test6pieces() {
	m := NewMT(20, 120)
	fmt.Println(len(m.t))

	n := m.t[0]
	for n != nil {
		fmt.Println("node_index:", n.index)
		if n.left != nil {
			fmt.Println("  left sibling:", n.left.index)
			n = n.parent
			continue
		}
		if n.right != nil {
			fmt.Println("  right sibling:", n.right.index)
			n = n.parent
			continue
		}
		fmt.Println("this is root node")
		break
	}
	fmt.Println(m.t[8] == nil)
	fmt.Println(m.t[10].index == 10)
	fmt.Println(m.t[4].parent == m.t[5].parent)
	fmt.Println(m.t[4].parent.index == 10)
}

func test5pieces() {
	m := NewMT(20, 100)
	fmt.Println(len(m.t))
	n := m.t[0]
	for n != nil {
		fmt.Println("node_index:", n.index)
		if n.left != nil {
			fmt.Println("  left sibling:", n.left.index)
			n = n.parent
			continue
		}
		if n.right != nil {
			fmt.Println("  right sibling:", n.right.index)
			n = n.parent
			continue
		}
		fmt.Println("this is root node")
		break
	}
}
