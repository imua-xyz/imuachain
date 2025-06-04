package types

import (
	"math/rand"
	"testing"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

var (
	emptyHashArr = [32]byte{}

	emptyHash = emptyHashArr[:]
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
		if expectedPath == nil {
			require.Empty(t, proofPath, "expected empty proof path")
		} else {
			for i, p := range proof {
				require.Equal(t, p.Index, proofPath[i])
				require.Equal(t, proofPath[i], expectedPath[i])
			}
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
	return mt, changes
}

func test6pieces(t *testing.T) {
	m, _ := NewMT(20, 6, emptyHash)
	require.Equal(t, 11, len(m.t))
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
		break
	}
	require.Nil(t, m.t[8])
	require.Equal(t, uint32(10), m.t[10].index)
	require.Equal(t, m.t[5].parent, m.t[4].parent)
	require.Equal(t, uint32(10), m.t[4].parent.index)
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
		break
	}
}

func TestMerkleTreeSinglePiece(t *testing.T) {
	piece := []byte("single piece test")
	mt, err := DeriveMT(uint32(len(piece)), piece)
	require.NoError(t, err)
	require.Equal(t, uint32(1), mt.leafCount)
	// Proof path for the only leaf should be empty
	path := mt.ProofPathFromLeafIndex(0, false)
	require.Empty(t, path)
	// Completed should be true
	_, ok := mt.CollectedPieces()
	require.True(t, ok)
	// CompleteRawData should return the original data
	data, ok := mt.CompleteRawData()
	require.True(t, ok)
	require.Equal(t, piece, data)
}

func TestMerkleTreeInvalidInputs(t *testing.T) {
	// Zero piece size
	_, err := DeriveMT(0, []byte("data"))
	require.Error(t, err)
	// Empty data
	_, err = DeriveMT(32, []byte{})
	require.Error(t, err)
	// Mismatched leaf count and data size, 3 leaves, but no data
	_, err = NewMT(32, 3, emptyHash)
	// Should not error, but tree is empty
	require.NoError(t, err)
}

func TestMerkleTreeProofVerificationAllLeaves(t *testing.T) {
	pieceSize := uint32(16)
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}
	mt, err := DeriveMT(pieceSize, data)
	require.NoError(t, err)
	mtEmpty, err := NewMT(pieceSize, mt.LeafCount(), mt.RootHash())
	require.NoError(t, err)
	for i := uint32(0); i < mt.LeafCount(); i++ {
		proof := mt.MinimalProofByIndex(i)
		piece := mt.pieces[i]
		mtEmpty.VerifyAndCacheOrdered(i, piece, proof)
		if i == mt.LeafCount()-1 {
			// Last piece should complete the tree
			_, completed := mtEmpty.CollectedPieces()
			require.True(t, completed)
		} else {
			_, completed := mtEmpty.CollectedPieces()
			require.False(t, completed)
		}
	}
}

func TestMerkleTreeGetCopyIndependence(t *testing.T) {
	pieceSize := uint32(8)
	data := []byte("abcdefghABCDEFGH")
	mt, err := DeriveMT(pieceSize, data)
	require.NoError(t, err)
	mtCopy := mt.GetCopy()
	require.NotNil(t, mtCopy)
	// Modify the copy
	mtCopy.pieces[0] = []byte{'X'}
	require.NotEqual(t, mt.pieces[0][0], mtCopy.pieces[0][0])
}

func TestMerkleTreeMinimalProofPathUniqueness(t *testing.T) {
	pieceSize := uint32(4)
	data := []byte("abcdefghijkl")
	mt, err := DeriveMT(pieceSize, data)
	require.NoError(t, err)
	for i := uint32(0); i < mt.LeafCount(); i++ {
		path := mt.MinimalProofPathByIndex(i)
		seen := make(map[uint32]struct{})
		for _, idx := range path {
			_, exists := seen[idx]
			require.False(t, exists, "duplicate index in minimal proof path")
			seen[idx] = struct{}{}
		}
	}
}
