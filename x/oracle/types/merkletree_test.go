package types

import (
	"fmt"
	"math/rand"
	"testing"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

var (
	emptyHashArr = [32]byte{}
	emptyHash    = emptyHashArr[:]
)

func TestMerkleTreePath(t *testing.T) {
	test6pieces(t)

	test5pieces()

	m, _ := NewMT(20, 6, emptyHash)
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(0, false), []uint32{1, 7, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(1, false), []uint32{0, 7, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(2, false), []uint32{3, 6, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(3, false), []uint32{2, 6, 10})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(4, false), []uint32{5, 9})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(5, false), []uint32{4, 9})

	m, _ = NewMT(20, 5, emptyHash)
	// if limit the provider to upload pieces ordered, then with cache, we only need proof:
	// (1,6,4), (-), (3), (2)
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(0, false), []uint32{1, 6, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(1, false), []uint32{0, 6, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(2, false), []uint32{3, 5, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(3, false), []uint32{2, 5, 4})
	require.ElementsMatch(t, m.ProofPathFromLeafIndex(4, false), []uint32{8})
}

func TestMerkleTreeVerify(t *testing.T) {
	mt, _ := GetNstRootAndPiecesWithParams(30, 1, 32)
	mtEmpty, err := NewMT(32, mt.LeafCount(), mt.RootHash())
	require.NoError(t, err)
	require.Equal(t, uint32(9), mt.leafCount)

	verifyPiece(t, 0, mt, mtEmpty, []uint32{1, 10, 15, 8}, true, false)
	verifyPiece(t, 1, mt, mtEmpty, nil, false, false)
	// verify a not in order, failed
	verified := verifyPiece(t, 5, mt, mtEmpty, nil, false, true)
	pieces, found := mtEmpty.CollectedPieces()
	require.False(t, verified)
	require.False(t, found)
	require.Equal(t, 2, len(pieces))

	verifyPiece(t, 2, mt, mtEmpty, []uint32{3}, true, false)
	verifyPiece(t, 3, mt, mtEmpty, nil, false, false)
	verifyPiece(t, 4, mt, mtEmpty, []uint32{5, 12}, true, false)

	// pieces are slieces from rawData, so we use cpy to replace piece 5 inorder to modify it independently
	cpy := make([]byte, len(mt.pieces[5]))
	tmp := make([]byte, len(mt.pieces[5]))
	copy(cpy, mt.pieces[5])
	copy(tmp, mt.pieces[5])
	mt.pieces[5] = cpy
	mt.pieces[5] = append(mt.pieces[5][1:], []byte{9}...)
	verified = verifyPiece(t, 5, mt, mtEmpty, nil, false, true)
	require.False(t, verified)
	pieces, found = mtEmpty.CollectedPieces()
	require.False(t, found)
	require.Equal(t, 5, len(pieces))
	mt.pieces[5] = tmp

	verifyPiece(t, 5, mt, mtEmpty, nil, false, false)
	verifyPiece(t, 6, mt, mtEmpty, []uint32{7}, true, false)
	verifyPiece(t, 7, mt, mtEmpty, nil, false, false)
	verifyPiece(t, 8, mt, mtEmpty, nil, false, false)
}

func verifyPiece(t *testing.T, index uint32, mt, mtEmpty *MerkleTree, expectedPath []uint32, cacheLeaf bool, skipCheck bool) bool {
	proof := mt.MinimalProofByIndex(index)
	if !skipCheck {
		proofPath := mt.MinimalProofPathByIndex(index)
		for i, p := range proof {
			require.Equal(t, p.Index, proofPath[i])
			require.Equal(t, proofPath[i], expectedPath[i])
		}
	}

	piece := mt.pieces[index]
	cached, ok := mtEmpty.VerifyAndCacheOrdered(index, piece, proof)
	if skipCheck {
		return ok
	}

	require.True(t, ok)
	addLeaf := 0
	if cacheLeaf {
		addLeaf = 1
	}
	require.Len(t, cached, len(proof)+addLeaf)

	pieces, ok := mtEmpty.CollectedPieces()
	if index == mt.leafCount-1 {
		require.True(t, ok)
	} else {
		require.False(t, ok)
	}
	require.Equal(t, piece, pieces[index])

	for _, i := range expectedPath {
		require.Equal(t, mt.t[i].hash, mtEmpty.t[i].hash)
	}
	return ok
}

func GetNstRootAndPiecesWithParams(stakerCount, version uint32, pieceSize uint32) (*MerkleTree, []*NSTKV) {
	nstbc := RawDataNST{
		Version: uint64(version),
	}
	changes := make([]*NSTKV, 0, stakerCount)
	for i := uint32(0); i < stakerCount; i++ {
		changes = append(changes, &NSTKV{
			StakerIndex: i,
			Balance:     uint64(rand.Int63n(99999999) + 1),
		})
	}
	nstbc.NstBalanceChanges = changes
	bz, err := proto.Marshal(&nstbc)
	if err != nil {
		panic(err)
	}

	mt, err := DeriveMT(pieceSize, bz)
	if err != nil {
		panic(err)
	}
	_, ok := mt.CollectedPieces()
	if !ok {
		panic("derived mt is incorrect")
	}
	//	return mt.RootHash(), pieces, changes
	return mt, changes
}

func test6pieces(t *testing.T) {
	m, _ := NewMT(20, 6, emptyHash)
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
	require.Nil(t, m.t[8])
	//	fmt.Println(m.t[8] == nil)
	require.Equal(t, uint32(10), m.t[10].index)
	//	fmt.Println(m.t[10].index == 10)
	require.Equal(t, m.t[5].parent, m.t[4].parent)
	//	fmt.Println(m.t[4].parent == m.t[5].parent)
	require.Equal(t, uint32(10), m.t[4].parent.index)
	// fmt.Println(m.t[4].parent.index == 10)
}

func test5pieces() {
	m, _ := NewMT(20, 5, emptyHash)
	n := m.t[0]
	for n != nil {
		if n.left != nil {
			n = n.parent
			continue
		}
		if n.right != nil {
			n = n.parent
			continue
		}
		fmt.Println("this is root node")
		break
	}
}
