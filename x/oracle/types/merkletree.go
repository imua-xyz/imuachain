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

// Node represents a node in the Merkle tree.
// Each node contains:
//   - hash: The hash value of the node's data or combined data of its children
//   - index: The position of the node in the tree
//   - parent: Reference to the parent node
//   - left: Reference to the left sibling
//   - right: Reference to the right sibling
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

// HashNode returns a HashNode representation of the current node.
// This is used for serialization and proof generation.
func (n *Node) HashNode() *HashNode {
	return &HashNode{
		Index: n.index,
		Hash:  n.hash,
	}
}

// MerkleTree represents a complete Merkle tree structure.
// It contains:
//   - root: The root hash of the tree
//   - rootIndex: The index of the root node
//   - t: A map of all nodes in the tree, indexed by their position
//   - pieces: The actual data pieces (only for leaves)
//   - rawData: The complete raw data (when available)
//   - leafCount: Total number of leaf nodes
//   - pieceSize: Size of each data piece
//   - minimalProofPath: Precomputed proof paths for each leaf
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

// Proof represents an ordered list of hash nodes needed to verify a piece of data.
// The nodes are ordered from bottom to top in the tree.
type Proof []*HashNode

// getHashNodeByIndex finds a HashNode in the proof by its index.
// Returns nil if the node is not found in the proof.
func (p *Proof) getHashNodeByIndex(index uint32) *HashNode {
	for _, hn := range *p {
		if hn.Index == index {
			return hn
		}
	}
	return nil
}

// GetCopy creates a deep copy of the proof.
// This is useful when the proof needs to be modified without affecting the original.
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

// FlattenString converts the proof into two strings:
//   - A string of indices separated by DelimiterForBase64
//   - A string of base64-encoded hashes separated by DelimiterForBase64
//
// This is used for serialization and transmission of proofs.
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

// SetRawDataPieces sets the raw data pieces for the tree.
// This is used when reconstructing a tree from pieces.
func (m *MerkleTree) SetRawDataPieces(pieces [][]byte) {
	m.pieces = pieces
}

// SetProofNodes updates the hash values of nodes in the tree using the provided proof nodes.
// This is used when reconstructing a tree from a proof.
func (m *MerkleTree) SetProofNodes(nodes []*HashNode) {
	for _, node := range nodes {
		n := m.t[node.Index]
		if n == nil {
			continue
		}
		n.hash = node.Hash
	}
}

// ProofPathFromLeafIndex returns the indices of nodes needed to verify a leaf node.
// If withCalculatedNodes is true, includes intermediate nodes in the path.
func (m *MerkleTree) ProofPathFromLeafIndex(index uint32, withCalculatedNodes bool) []uint32 {
	if index >= m.leafCount {
		return nil
	}
	node, ok := m.t[index]
	if !ok {
		return nil
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

// UncachedProofPathFromLeafIndex returns the indices of nodes that need to be provided
// to verify a leaf node, excluding nodes that are already cached in the tree.
func (m *MerkleTree) UncachedProofPathFromLeafIndex(index uint32) []uint32 {
	if index >= m.leafCount {
		return nil
	}
	node, ok := m.t[index]
	if !ok {
		return nil
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

// VerifyAndCacheOrdered verifies a target piece against a proof and caches the nodes in the tree.
// It requires that the target piece is the next piece in sequence after the cached pieces.
// Returns:
//   - cachedProof: The proof nodes that were cached during verification
//   - verified: Whether the verification was successful
func (m *MerkleTree) VerifyAndCacheOrdered(targetIndex uint32, targetPiece []byte, proof Proof) (cachedProof Proof, verified bool) {
	// as an 'ordered' method, we require the targetIndex to be the next index against cached pieces
	// #nosec G115 - len(m.pieces) is limited by m.leafCount
	if targetIndex >= m.leafCount || targetIndex != uint32(len(m.pieces)) {
		return nil, false
	}
	// #nosec G115
	if m.leafCount > 1 &&
		targetIndex < m.leafCount-1 &&
		uint32(len(targetPiece)) != m.pieceSize {
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
		if node.left != nil {
			hash = innerHash(pairHash, hash)
		} else {
			hash = innerHash(hash, pairHash)
		}
		if node.parent.hash != nil {
			verified := bytes.Equal(node.parent.hash, hash)
			if verified {
				// Safe to range map: assignments by index are order-independent
				// and yield the same final state regardless of iteration order.
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

// CollectedPieces returns the currently collected pieces and whether all pieces have been collected.
// The slice index corresponds to the leaf node index.
func (m *MerkleTree) CollectedPieces() ([][]byte, bool) {
	if len(m.pieces) == int(m.leafCount) {
		return m.pieces, true
	}
	return m.pieces, false
}

// PieceByIndex returns the piece at the specified index and whether it exists.
func (m *MerkleTree) PieceByIndex(targetIndex uint32) ([]byte, bool) {
	if int(targetIndex) >= len(m.pieces) {
		return nil, false
	}
	return m.pieces[targetIndex], true
}

// CompleteRawData returns the complete raw data and whether all pieces have been collected.
// Returns nil and false if the data is incomplete.
func (m *MerkleTree) CompleteRawData() ([]byte, bool) {
	// #nosec G115 - the size of m.pieces is limited
	if m == nil || uint32(len(m.pieces)) < m.leafCount || m.leafCount == 0 {
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

// Completed returns whether all pieces of the raw data have been collected.
// Note: An empty tree (leafCount = 0) is considered completed.
func (m *MerkleTree) Completed() bool {
	return m != nil && len(m.pieces) == int(m.leafCount)
}

// CollectingRawData returns whether the tree is in the process of collecting pieces.
func (m *MerkleTree) CollectingRawData() bool {
	// #nosec G115 - the length of m.pieces is limited, its size won't exceed m.leafCount
	return m != nil && uint32(len(m.pieces)) < m.leafCount
}

// LatestLeafIndex returns the index of the most recently collected leaf node.
// The second return value indicates whether any nodes have been collected.
func (m *MerkleTree) LatestLeafIndex() (uint32, bool) {
	if len(m.pieces) == 0 {
		return 0, false
	}
	// #nosec G115 - checked 0 case
	return uint32(len(m.pieces) - 1), true
}

// MinimalProofPathByIndex returns the minimal proof path for a given leaf index.
// Returns nil if the index is invalid or no proof path exists.
func (m *MerkleTree) MinimalProofPathByIndex(index uint32) []uint32 {
	proofPath, ok := m.minimalProofPath[index]
	if !ok {
		return nil
	}
	return proofPath
}

// MinimalProofByIndex returns the minimal proof for a given leaf index.
// The proof consists of HashNode objects containing the necessary hash values.
func (m *MerkleTree) MinimalProofByIndex(index uint32) Proof {
	path := m.MinimalProofPathByIndex(index)
	ret := make([]*HashNode, 0, len(path))
	for _, idx := range path {
		ret = append(ret, m.t[idx].HashNode())
	}
	return ret
}

// LeafCount returns the total number of leaf nodes in the tree.
func (m *MerkleTree) LeafCount() uint32 {
	return m.leafCount
}

// NextPieceIndex returns the index of the next piece to be collected.
// The second return value indicates whether there are more pieces to collect.
func (m *MerkleTree) NextPieceIndex() (uint32, bool) {
	// #nosec G115
	idx := uint32(len(m.pieces))
	if idx >= m.leafCount {
		return 0, false
	}

	return idx, true
}

// RootHash returns the root hash of the tree.
func (m *MerkleTree) RootHash() []byte {
	return m.root
}

// RootIndex returns the index of the root node in the tree.
func (m *MerkleTree) RootIndex() uint32 {
	return m.rootIndex
}

// GetCopy return a copy of the MerkleTree
// NOTE: the returned MerkleTree is not a deep copy, the hash of each node is copied, but the pieces and rawData, path are not copied
// for the shallow copies, the outter slice/map is copied, but the inner slice/map is not copied
func (m *MerkleTree) GetCopy() *MerkleTree {
	if m == nil {
		return nil
	}
	ret, err := NewMT(m.pieceSize, m.leafCount, m.root)
	// err should always be nil when copy on a valid MerkleTree
	if err != nil {
		return nil
	}

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

// calculateProofPaths calculates and caches the minimal proof paths for all leaves.
// This is called during tree construction to optimize proof generation.
func (m *MerkleTree) calculateProofPaths() map[uint32][]uint32 {
	minimalProofPath := make(map[uint32][]uint32)
	seenIndex := make(map[uint32]struct{})

	for i := uint32(0); i < m.leafCount; i++ {
		seenIndex[i] = struct{}{}
		path := m.ProofPathFromLeafIndex(i, false)
		minimalPath := make([]uint32, 0, len(path))
		for _, pIndex := range path {
			if _, ok := seenIndex[pIndex]; !ok {
				seenIndex[pIndex] = struct{}{}
				minimalPath = append(minimalPath, pIndex)
			}
		}
		if len(minimalPath) > 0 {
			minimalProofPath[i] = minimalPath
		}
		pathWithCalculatedNodes := m.ProofPathFromLeafIndex(i, true)
		for _, pIndx := range pathWithCalculatedNodes {
			if _, ok := seenIndex[pIndx]; !ok {
				seenIndex[pIndx] = struct{}{}
			}
		}
	}

	m.minimalProofPath = minimalProofPath
	return minimalProofPath
}

// leafHash calculates the hash of a leaf node's data.
func leafHash(node []byte) []byte {
	ret := sha256.Sum256(node)
	return ret[:]
}

// innerHash calculates the hash of two inner nodes combined.
func innerHash(lNode, rNode []byte) []byte {
	combined := make([]byte, 0, len(lNode)+len(rNode))
	combined = append(combined, lNode...)
	combined = append(combined, rNode...)
	ret := sha256.Sum256(combined)
	return ret[:]
}

// NewMT creates a new MerkleTree initialized with a given root hash.
// This is used when reconstructing a tree from a known root hash.
func NewMT(pieceSize, leafCount uint32, root []byte) (*MerkleTree, error) {
	return merkleTreeFromBytes(pieceSize, leafCount, root, false)
}

// DeriveMT creates a new MerkleTree from raw data.
// The tree structure is derived from the data and piece size.
func DeriveMT(pieceSize uint32, rawData []byte) (*MerkleTree, error) {
	return merkleTreeFromBytes(pieceSize, 0, rawData, true)
}

// validateInput validates the input parameters for merkle tree construction.
// It checks pieceSize, calculates the actual size and leaf count based on the input data.
// Returns:
//   - size: The actual size of the data in bytes
//   - leafCount: The number of leaves in the tree
//   - error: Any validation error encountered
func validateInput(pieceSize uint32, leafCount uint32, bytes []byte, fromRawData bool) (uint32, uint32, error) {
	if pieceSize == 0 {
		return 0, 0, errors.New("pieceSize can't be zero")
	}

	size := uint32(0)
	if fromRawData {
		tmp := len(bytes)
		if tmp == 0 || tmp > math.MaxUint32 {
			return 0, 0, fmt.Errorf("rawData size is invalid, size=%d", tmp)
		}
		size = uint32(tmp)
		calculatedLeafCount := size / pieceSize
		if size%pieceSize > 0 {
			calculatedLeafCount++
		}
		if leafCount > 0 && calculatedLeafCount != leafCount {
			return 0, 0, fmt.Errorf("input leafCount doesn't equals to the result calculated from rawData and pieceSize, leafCount%d, calculatedLeafCount:%d", leafCount, calculatedLeafCount)
		}
		if leafCount == 0 {
			leafCount = calculatedLeafCount
		}
	} else if len(bytes) != common.HashLength {
		return 0, 0, fmt.Errorf("rootHash must have length of %d, got:%d", common.HashLength, len(bytes))
	}

	return size, leafCount, nil
}

// createSingleLeafTree creates a merkle tree with a single leaf node.
// This is a special case when there's only one piece of data.
// Returns:
//   - *MerkleTree: A tree with a single node
//   - error: Any error encountered during creation
func createSingleLeafTree(pieceSize uint32, leafCount uint32, bytes []byte, fromRawData bool) (*MerkleTree, error) {
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

// buildTreeNodes constructs the merkle tree structure by building nodes layer by layer.
// It handles both raw data and hash-only cases, and collects pieces for the bottom layer.
// Returns:
//   - uint32: The index of the root node
//   - [][]byte: The pieces of data (only populated when fromRawData is true)
//   - error: Any error encountered during construction
func buildTreeNodes(t map[uint32]*Node, leafCount uint32, prevLayersCount uint32, bytes []byte, pieceSize uint32, size uint32, fromRawData bool) (uint32, [][]byte, error) {
	originalLeafCount := leafCount
	pSize := uint32(0)
	if fromRawData {
		pSize = leafCount
	}
	pieces := make([][]byte, 0, pSize)

	for leafCount > 1 {
		isBottomLayer := prevLayersCount == 0
		for i := uint32(0); i < leafCount; i += 2 {
			idx := i + prevLayersCount
			lNode, err := getOrCreateNode(t, idx, bytes, i, pieceSize, size, fromRawData)
			if err != nil {
				return 0, nil, err
			}

			if fromRawData && isBottomLayer {
				endIdx := min((i+1)*pieceSize, size)
				piece := bytes[i*pieceSize : endIdx]
				pieces = append(pieces, piece)
			}
			if i+1 < leafCount {
				rNode, err := getOrCreateNode(t, idx+1, bytes, i+1, pieceSize, size, fromRawData)
				if err != nil {
					return 0, nil, err
				}

				if fromRawData && isBottomLayer {
					endIdx := min((i+2)*pieceSize, size)
					piece := bytes[(i+1)*pieceSize : endIdx]
					pieces = append(pieces, piece)
				}
				linkNodes(lNode, rNode)
				parentIdx := i/2 + prevLayersCount + leafCount
				createParentNode(t, parentIdx, lNode, rNode, fromRawData)
			} else {
				if err := liftSingleNode(t, lNode, i, prevLayersCount, leafCount, originalLeafCount); err != nil {
					return 0, nil, err
				}
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

	return prevLayersCount, pieces, nil
}

// getOrCreateNode retrieves an existing node or creates a new one at the specified index.
// For raw data, it also calculates the hash of the piece.
// Returns:
//   - *Node: The existing or newly created node
//   - error: Any error encountered during node creation
func getOrCreateNode(t map[uint32]*Node, idx uint32, bytes []byte, i uint32, pieceSize uint32, size uint32, fromRawData bool) (*Node, error) {
	node := t[idx]
	if node == nil {
		node = &Node{index: idx}
		if fromRawData {
			if i*pieceSize >= size {
				return nil, errors.New("pieceSize is too large for the rawData")
			}
			endIdx := min((i+1)*pieceSize, size)
			piece := bytes[i*pieceSize : endIdx]
			node.hash = leafHash(piece)
		}
		t[idx] = node
	}
	return node, nil
}

// linkNodes establishes sibling relationships between two nodes.
// It sets the left and right pointers to create a bidirectional link.
func linkNodes(lNode, rNode *Node) {
	// Link siblings
	if lNode.right == nil {
		lNode.right = rNode
	}
	if rNode.left == nil {
		rNode.left = lNode
	}
}

// createParentNode creates a parent node for two sibling nodes and establishes parent-child relationships.
// For raw data, it also calculates the parent's hash from its children.
func createParentNode(t map[uint32]*Node, parentIdx uint32, lNode, rNode *Node, fromRawData bool) {
	parentNode := t[parentIdx]
	if parentNode == nil {
		parentNode = &Node{index: parentIdx}
		if fromRawData {
			parentNode.hash = innerHash(lNode.hash, rNode.hash)
		}
		t[parentIdx] = parentNode
	}
	// Set parent references
	if lNode.parent == nil {
		lNode.parent = parentNode
	}
	if rNode.parent == nil {
		rNode.parent = parentNode
	}
}

// liftSingleNode handles the case where a node doesn't have a sibling at its level.
// It moves the node up to the next level in the tree.
// Returns:
//   - error: Any error encountered during the lifting operation
func liftSingleNode(t map[uint32]*Node, lNode *Node, i uint32, prevLayersCount uint32, leafCount uint32, originalLeafCount uint32) error {
	liftedNodeIndex := i/2 + prevLayersCount + leafCount
	liftedNode := t[liftedNodeIndex]
	if liftedNode == nil {
		if lNode.index >= originalLeafCount {
			delete(t, lNode.index)
			lNode.index = liftedNodeIndex
		}
		t[liftedNodeIndex] = lNode
	} else {
		return errors.New("liftedNode must be nil when do lifting")
	}
	return nil
}

// merkleTreeFromBytes constructs a merkle tree from either raw data or a root hash.
// Parameters:
//   - pieceSize: The size of each piece of data
//   - leafCount: The number of leaves in the tree (0 for automatic calculation)
//   - bytes: Either the raw data or the root hash
//   - fromRawData: Whether bytes represents raw data (true) or root hash (false)
//
// Returns:
//   - *MerkleTree: The constructed merkle tree
//   - error: Any error encountered during construction
func merkleTreeFromBytes(pieceSize uint32, leafCount uint32, bytes []byte, fromRawData bool) (*MerkleTree, error) {
	size, leafCount, err := validateInput(pieceSize, leafCount, bytes, fromRawData)
	if err != nil {
		return nil, err
	}

	if leafCount == 1 {
		return createSingleLeafTree(pieceSize, leafCount, bytes, fromRawData)
	}

	t := make(map[uint32]*Node)
	prevLayersCount, pieces, err := buildTreeNodes(t, leafCount, 0, bytes, pieceSize, size, fromRawData)
	if err != nil {
		return nil, err
	}

	ret := &MerkleTree{
		rootIndex: prevLayersCount,
		t:         t,
		leafCount: leafCount,
		pieceSize: pieceSize,
	}

	if fromRawData {
		ret.root = t[prevLayersCount].hash
		ret.rawData = bytes
		ret.pieces = pieces
	} else {
		ret.root = bytes
		ret.t[prevLayersCount].hash = bytes
	}

	ret.calculateProofPaths()

	return ret, nil
}
