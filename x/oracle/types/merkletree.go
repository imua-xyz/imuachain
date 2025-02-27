package types

import (
	"bytes"
	"crypto/sha256"
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

func (p *Proof) getHashByIndex(index uint32) []byte {
	for _, hn := range *p {
		if hn.Index == index {
			return hn.Hash
		}
	}
	return nil
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

func (m *MerkleTree) ProofPathFromLeafIndex(index uint32) []uint32 {
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

func (m *MerkleTree) VerifyAndCache(targetIndex uint32, targetPiece []byte, proof Proof) (cachedProof Proof, verified bool) {
	if targetIndex >= m.leafCount {
		return nil, false
	}
	tmpHash := sha256.Sum256(targetPiece)
	hash := tmpHash[:]
	// get hashed-leafnode
	node := m.t[targetIndex]
	if node.hash != nil {
		return nil, bytes.Equal(node.hash, hash)
	}
	ret := make([]*HashNode, 0, len(proof))
	ret = append(ret, &HashNode{Index: node.index, Hash: hash})
	newNode := &Node{
		// tmp cache the unverified hash in a new node
		hash:   hash,
		index:  node.index,
		parent: node.parent,
		// left sibling
		left: node.left,
		// right sibling
		right: node.right,
	}
	tmpNode := newNode
	// only root have no sibling, and root must have hash
	for tmpNode.right != nil || tmpNode.left != nil {
		var parentHash []byte
		var pairHash []byte
		var pairNode *Node
		var combinednodes []byte

		if tmpNode.right != nil {
			pairNode = tmpNode.right
		} else {
			pairNode = tmpNode.left
		}

		if pairHash = pairNode.hash; pairHash == nil {
			pairHash = proof.getHashByIndex(pairNode.index)
			if len(pairHash) == 0 {
				return nil, false
			}
			ret = append(ret, &HashNode{Index: pairNode.index, Hash: pairHash})
			pairNode = &Node{
				index:  pairNode.index,
				hash:   pairHash,
				right:  pairNode.right,
				left:   pairNode.left,
				parent: pairNode.parent,
			}
		}
		if tmpNode.right != nil {
			tmpNode.right = pairNode
			combinednodes = append(tmpNode.hash, pairHash...)
		} else {
			tmpNode.left = pairNode
			combinednodes = append(pairHash, tmpNode.hash...)
		}

		tmpHash = sha256.Sum256(combinednodes)
		parentHash = tmpHash[:]

		if tmpNode.parent == nil {
			// this should not happen
			return nil, false
		}

		parentNode := tmpNode.parent
		if parentNode.hash != nil {
			if bytes.Equal(parentNode.hash, parentHash) {
				// update cache
				m.t[targetIndex] = newNode
				return ret, true
			}
			return nil, false
		}

		// parent.hash == nil, new a cache node
		parentNode = &Node{
			index:  parentNode.index,
			hash:   parentHash,
			left:   parentNode.left,
			right:  parentNode.right,
			parent: parentNode.parent,
		}

		tmpNode.parent = parentNode
		pairNode.parent = parentNode

		tmpNode = parentNode
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

// NewMT new a merkle tree initialized with the topology from input pieceSize and totalSize
func NewMT(pieceSize, leafCount uint32, root []byte) *MerkleTree {
	if leafCount < 1 {
		return nil
	}
	originalLeafCount := leafCount

	ret := &MerkleTree{
		pieces:           make([][]byte, 0, leafCount),
		leafCount:        leafCount,
		pieceSize:        pieceSize,
		minimalProofPath: make(map[uint32][]uint32),
	}

	if leafCount == 1 {
		ret.t = map[uint32]*Node{0: {index: 1}}
		return ret
	}

	t := make(map[uint32]*Node)
	prevLayersCount := uint32(0)

	for leafCount > 1 {
		for i := uint32(0); i < leafCount; i += 2 {
			idx := i + prevLayersCount

			lNode := t[idx]
			if lNode == nil {
				lNode = &Node{index: idx}
				t[idx] = lNode
			}

			if i+1 < leafCount {
				rNode := t[idx+1]
				if rNode == nil {
					rNode = &Node{index: idx + 1}
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
	t[prevLayersCount] = &Node{
		index: prevLayersCount,
		hash:  root,
		// root node, got no parent or siblings
	}

	ret.t = t
	ret.root = root
	ret.rootIndex = prevLayersCount

	tmpIndex := make(map[uint32]struct{})
	for i := uint32(0); i < ret.leafCount; i++ {
		path := ret.ProofPathFromLeafIndex(i)
		minimalPath := make([]uint32, 0, len(path))
		for _, pIndex := range path {
			if _, ok := tmpIndex[pIndex]; !ok {
				tmpIndex[pIndex] = struct{}{}
				minimalPath = append(minimalPath, pIndex)
			}
		}
		if len(minimalPath) > 0 {
			ret.minimalProofPath[i] = minimalPath
		}
	}

	return ret
}
