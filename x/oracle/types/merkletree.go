package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// node represents leaf-node
type Node struct {
	// hash   *common.Hash
	// use []byte instead of common.Hash for conveniences for appending
	hash   []byte
	index  uint32
	parent *Node
	// left sibling
	left *Node
	// right sibling
	right *Node
}

func (n *Node) HashNode() *HashNode {
	return &HashNode{
		Index: n.index,
		Hash:  n.hash,
	}
}

type MerkleTree struct {
	root      []byte
	rootIndex uint32
	t         map[uint32]*Node
	pieces    [][]byte
	rawData   []byte
	leafCount uint32
	// size of bytes
	pieceSize        uint32
	minimalProofPath map[uint32][]uint32
}

// ordered bottom up
type Proof []*HashNode

func (p *Proof) getHashNodeByIndex(index uint32) *HashNode {
	for _, hn := range *p {
		if hn.Index == index {
			return hn
		}
	}
	return nil
}

func (p *Proof) GetCopy() Proof {
	if p == nil {
		return nil
	}
	tmp := *p
	ret := make([]*HashNode, 0, len(tmp))
	for _, hn := range tmp {
		hash := make([]byte, len(hn.Hash))
		copy(hash, hn.Hash)
		ret = append(ret, &HashNode{
			Index: hn.Index,
			Hash:  hash,
		})
	}
	return ret
}

func (p *Proof) FlattenString() (string, string) {
	tmp := *p
	if len(tmp) == 0 {
		return "", ""
	}
	idxStr := strconv.Itoa(int(tmp[0].Index))
	hashStr := base64.StdEncoding.EncodeToString(tmp[0].Hash)

	for i := 1; i < len(tmp); i++ {
		idxStr += DelimiterForBase64
		idxStr += strconv.Itoa(int(tmp[i].Index))
		hashStr += DelimiterForBase64
		hashStr += base64.StdEncoding.EncodeToString(tmp[i].Hash)
	}
	return idxStr, hashStr
}
func (m *MerkleTree) SetRawDataPieces(pieces [][]byte) {
	m.pieces = pieces
}

func (m *MerkleTree) SetProofNodes(nodes []*HashNode) {
	for _, node := range nodes {
		n := m.t[node.Index]
		if n == nil {
			continue
		}
		n.hash = node.Hash
	}
}

func (m *MerkleTree) ProofPathFromLeafIndex(index uint32, withCalculatedNodes bool) []uint32 {
	if index >= m.leafCount {
		return nil
	}
	node, ok := m.t[index]
	if !ok {
		panic("merkle is not initialized correctly")
	}
	path := make([]uint32, 0)
	for node != nil {
		if withCalculatedNodes {
			path = append(path, node.index)
		}
		if node.left == nil && node.right == nil {
			break
		}
		if node.left != nil {
			path = append(path, node.left.index)
		} else {
			path = append(path, node.right.index)
		}
		node = node.parent
	}
	return path
}

func (m *MerkleTree) UncachedProofPathFromLeafIndex(index uint32) []uint32 {
	if index >= m.leafCount {
		return nil
	}
	node, ok := m.t[index]
	if !ok {
		panic("merkle is not initialized correctly")
	}
	path := make([]uint32, 0)
	for node != nil {
		if node.left == nil && node.right == nil {
			break
		}
		if node.left != nil {
			if m.t[node.left.index].hash == nil {
				path = append(path, node.left.index)
			}
		} else {
			if m.t[node.right.index].hash == nil {
				path = append(path, node.right.index)
			}
		}
		node = node.parent
	}
	return path
}

// VerifyAndCacheOrdered verifies the target piece with provided proof, and it requires the targetPiece is exactly the next piece against cachec pieces
func (m *MerkleTree) VerifyAndCacheOrdered(targetIndex uint32, targetPiece []byte, proof Proof) (cachedProof Proof, verified bool) {
	// as an 'ordered' method, we require the targetIndex to be the next index against cached pieces
	// #nosec G115 - len(m.pieces) is limited by m.leafCount
	if targetIndex >= m.leafCount || targetIndex != uint32(len(m.pieces)) {
		return nil, false
	}
	hash := leafHash(targetPiece)
	// get hashed-leafnode
	node := m.t[targetIndex]
	if node.hash != nil {
		verified = bytes.Equal(node.hash, hash)
		if verified {
			m.pieces = append(m.pieces, targetPiece)
		}
		return nil, verified
	}
	cachedHashes := make(map[uint32][]byte)
	cachedProof = make([]*HashNode, 0, len(proof))
	cachedHashes[targetIndex] = hash
	cachedProof = append(cachedProof, &HashNode{Index: targetIndex, Hash: hash})
	for node.parent != nil {
		var pairNode *Node
		if node.left != nil {
			pairNode = node.left
		} else {
			pairNode = node.right
		}
		var pairHash []byte
		if pairNode.hash == nil {
			//			pairHash = proof.getHashByIndex(pairNode.index)
			hashNode := proof.getHashNodeByIndex(pairNode.index)
			if hashNode == nil || len(hashNode.Hash) == 0 {
				return nil, false
			}
			pairHash = hashNode.Hash
			cachedHashes[pairNode.index] = pairHash
			cachedProof = append(cachedProof, hashNode)
		} else {
			pairHash = pairNode.hash
		}
		//		var calculatedParentHash []byte
		if node.left != nil {
			//			calculatedParentHash = innerHash(pairHash, hash)
			hash = innerHash(pairHash, hash)
		} else {
			//			calculatedParentHash = innerHash(hash, pairHash)
			hash = innerHash(hash, pairHash)
		}
		if node.parent.hash != nil {
			// verified := bytes.Equal(node.parent.hash, calculatedParentHash)
			verified := bytes.Equal(node.parent.hash, hash)
			if verified {
				// copy cached into merkletree
				for idx, cachedHash := range cachedHashes {
					m.t[idx].hash = cachedHash
				}
				m.pieces = append(m.pieces, targetPiece)
				return cachedProof, true
			}
			return nil, false
		}
		cachedHashes[node.parent.index] = hash

		node = node.parent
	}
	return nil, false
}

// RawData return the collected rawData pieces and true/false to tell wether got the complete raw data
// slice index is the same with the leaf node index
func (m *MerkleTree) CollectedPieces() ([][]byte, bool) {
	if len(m.pieces) == int(m.leafCount) {
		return m.pieces, true
	}
	return m.pieces, false
}

func (m *MerkleTree) PieceByIndex(targetIndex uint32) ([]byte, bool) {
	if int(targetIndex) >= len(m.pieces) {
		return nil, false
	}
	return m.pieces[targetIndex], true
}

// return rawData as a whole, with true/false to tell if we got the completed rawData
// when fasel, the returned first value should be nil
func (m *MerkleTree) CompleteRawData() ([]byte, bool) {
	if m == nil || len(m.pieces) < int(m.leafCount) || m.leafCount == 0 {
		return nil, false
	}
	if len(m.rawData) > 0 {
		return m.rawData, true
	}

	for _, piece := range m.pieces {
		m.rawData = append(m.rawData, piece...)
	}
	return m.rawData, true
}

// Completed returns wether the merkle tree has collected all pieces of rawData (all leaf nodes)
// when the MerkleTree is cleared, both len(pieces) and leafCount equal to 0, so it's also marked as 'completed'
// only when MerkleTree is set to non-zero leafCount with less amount of pieces than that leafCount we got false returned
// so when the return value is false, it also indicates that this MerkleTree is collecting pieces
func (m *MerkleTree) Completed() bool {
	return m != nil && len(m.pieces) == int(m.leafCount)
}

func (m *MerkleTree) CollectingRawData() bool {
	return m != nil && len(m.pieces) < int(m.leafCount)
}

// (0, true) means the first leaf node is cached
// (0, false) means there's no node cached yet
func (m *MerkleTree) LatestLeafIndex() (uint32, bool) {
	if len(m.pieces) == 0 {
		return 0, false
	}
	// #nosec G115 - checked 0 case
	return uint32(len(m.pieces) - 1), true
}

func (m *MerkleTree) MinimalProofPathByIndex(index uint32) []uint32 {
	proofPath, ok := m.minimalProofPath[index]
	if !ok {
		return nil
	}
	return proofPath
}

func (m *MerkleTree) MinimalProofByIndex(index uint32) Proof {
	path := m.MinimalProofPathByIndex(index)
	ret := make([]*HashNode, 0, len(path))
	for _, idx := range path {
		ret = append(ret, m.t[idx].HashNode())
	}
	return ret
}

func (m *MerkleTree) LeafCount() uint32 {
	return m.leafCount
}

func (m *MerkleTree) NextPieceIndex() (uint32, bool) {
	// #nosec G115
	idx := uint32(len(m.pieces))
	if idx >= m.leafCount {
		return 0, false
	}

	return idx, true
}

func (m *MerkleTree) RootHash() []byte {
	return m.root
}

func (m *MerkleTree) RootIndex() uint32 {
	return m.rootIndex
}

func (m *MerkleTree) GetCopy() *MerkleTree {
	if m == nil {
		return nil
	}
	ret, _ := NewMT(m.pieceSize, m.leafCount, m.root)
	for pIdx, p := range m.t {
		if p.hash != nil {
			ret.t[pIdx].hash = make([]byte, len(p.hash))
			copy(ret.t[pIdx].hash, p.hash)
		}
	}
	if len(m.pieces) > 0 {
		ret.pieces = make([][]byte, len(m.pieces))
		copy(ret.pieces, m.pieces)
	}
	if len(m.rawData) > 0 {
		ret.rawData = make([]byte, len(m.rawData))
		copy(ret.rawData, m.rawData)
	}
	if len(m.minimalProofPath) > 0 {
		ret.minimalProofPath = make(map[uint32][]uint32)
		for pIdx, path := range m.minimalProofPath {
			ret.minimalProofPath[pIdx] = path
		}
	}
	return ret
}

// fromRawData
//   - true: bytes represents the rawData itself
//   - false: bytes represents the rootHash
func merkleTreeFromBytes(pieceSize uint32, leafCount uint32, bytes []byte, fromRawData bool) (*MerkleTree, error) {
	if pieceSize == 0 {
		return nil, errors.New("pieceSize can't be zero")
	}
	size := uint32(0)
	if fromRawData {
		tmp := len(bytes)
		if tmp == 0 || tmp > math.MaxUint32 {
			return nil, fmt.Errorf("rawData size is invalid, size=%d", tmp)
		}
		size = uint32(tmp)
		calculatedLeafCount := size / pieceSize
		if size%pieceSize > 0 {
			calculatedLeafCount++
		}
		if leafCount > 0 && calculatedLeafCount != leafCount {
			return nil, fmt.Errorf("input leafCount doesn't equals to the result calculated from rawData and pieceSize, leafCount%d, calculatedLeafCount:%d", leafCount, calculatedLeafCount)
		}
		if leafCount == 0 {
			leafCount = calculatedLeafCount
		}
	} else if len(bytes) != common.HashLength {
		return nil, fmt.Errorf("rootHash must have length of %d, got:%d", common.HashLength, len(bytes))
	}
	if leafCount == 1 {
		node := &Node{
			index: 0,
		}
		ret := &MerkleTree{
			rootIndex: 0,
			t:         map[uint32]*Node{0: node},
			pieceSize: pieceSize,
			leafCount: leafCount,
		}
		if fromRawData {
			hash := leafHash(bytes)
			node.hash = hash
			ret.pieces = [][]byte{bytes}
			ret.rawData = bytes
			ret.root = hash
		} else {
			ret.root = bytes
			node.hash = bytes
		}
		return ret, nil
	}

	originalLeafCount := leafCount
	prevLayersCount := uint32(0)
	t := make(map[uint32]*Node)
	pSize := uint32(0)
	if fromRawData {
		pSize = leafCount
	}
	pieces := make([][]byte, 0, pSize)
	for leafCount > 1 {
		for i := uint32(0); i < leafCount; i += 2 {
			idx := i + prevLayersCount
			lNode := t[idx]
			if lNode == nil {
				lNode = &Node{
					index: idx,
				}
				// this only happens in the botoom layer which consists from all hash nodes of leaves
				if fromRawData {
					endIdx := min((i+1)*pieceSize, size)
					piece := bytes[i*pieceSize : endIdx]
					pieces = append(pieces, piece)
					lNode.hash = leafHash(piece)
				}
				t[idx] = lNode
			}

			if i+1 < leafCount {
				rNode := t[idx+1]
				if rNode == nil {
					rNode = &Node{index: idx + 1}
					if fromRawData {
						endIdx := min((i+2)*pieceSize, size)
						piece := bytes[(i+1)*pieceSize : endIdx]
						pieces = append(pieces, piece)
						rNode.hash = leafHash(piece)
					}
					t[idx+1] = rNode
				}

				if lNode.right == nil {
					lNode.right = rNode
				}
				if rNode.left == nil {
					rNode.left = lNode
				}

				// node pair derived a parent on upper level
				parentIdx := i/2 + prevLayersCount + leafCount
				parentNode := t[parentIdx]
				if parentNode == nil {
					parentNode = &Node{index: parentIdx}
					if fromRawData {
						parentNode.hash = innerHash(lNode.hash, rNode.hash)
					}
					t[parentIdx] = parentNode
				}
				if lNode.parent == nil {
					lNode.parent = parentNode
				}
				if rNode.parent == nil {
					rNode.parent = parentNode
				}
			} else {
				// lNode is a single node without pair, linked to no parent at this level, move it up to the end of next upper level
				liftedNodeIndex := i/2 + prevLayersCount + leafCount
				liftedNode := t[liftedNodeIndex]
				if liftedNode == nil {
					if lNode.index >= originalLeafCount {
						delete(t, lNode.index)
						lNode.index = liftedNodeIndex
					}
					t[liftedNodeIndex] = lNode
				} else {
					panic("liftedNode must be nil when do lifting")
				}
				// #nosec G115
				break
			}
		}
		prevLayersCount += leafCount
		if leafCount%2 == 1 {
			leafCount = leafCount/2 + 1
		} else {
			leafCount /= 2
		}
	}

	ret := &MerkleTree{
		rootIndex:        prevLayersCount,
		t:                t,
		leafCount:        originalLeafCount,
		pieceSize:        pieceSize,
		minimalProofPath: make(map[uint32][]uint32),
	}

	seenIndex := make(map[uint32]struct{})
	for i := uint32(0); i < ret.leafCount; i++ {
		seenIndex[i] = struct{}{}
		path := ret.ProofPathFromLeafIndex(i, false)
		minimalPath := make([]uint32, 0, len(path))
		for _, pIndex := range path {
			if _, ok := seenIndex[pIndex]; !ok {
				seenIndex[pIndex] = struct{}{}
				minimalPath = append(minimalPath, pIndex)
			}
		}
		if len(minimalPath) > 0 {
			ret.minimalProofPath[i] = minimalPath
		}
		pathWithCalculatedNodes := ret.ProofPathFromLeafIndex(i, true)
		for _, pIndx := range pathWithCalculatedNodes {
			if _, ok := seenIndex[pIndx]; !ok {
				seenIndex[pIndx] = struct{}{}
			}
		}
	}
	if fromRawData {
		ret.root = t[prevLayersCount].hash
		ret.rawData = bytes
		ret.pieces = pieces
	} else {
		ret.root = bytes
		ret.t[prevLayersCount].hash = bytes
	}
	return ret, nil
}

func leafHash(node []byte) []byte {
	ret := sha256.Sum256(node)
	return ret[:]
}

func innerHash(lNode, rNode []byte) []byte {
	combined := make([]byte, 0, len(lNode)+len(rNode))
	combined = append(combined, lNode...)
	combined = append(combined, rNode...)
	ret := sha256.Sum256(combined)
	return ret[:]
}

// NewMT new a merkle tree initialized with the topology from input pieceSize and totalSize
func NewMT(pieceSize, leafCount uint32, root []byte) (*MerkleTree, error) {
	return merkleTreeFromBytes(pieceSize, leafCount, root, false)
}

func DeriveMT(pieceSize uint32, rawData []byte) (*MerkleTree, error) {
	return merkleTreeFromBytes(pieceSize, 0, rawData, true)
}
