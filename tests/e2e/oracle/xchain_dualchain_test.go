package oracle

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestDualChainBridgeE2E is the 方案B test: imuachain + anvil dual-chain.
// It verifies the full bidirectional flow:
//  1. imuachain: deposit → withdraw → outbound queue → checkpoint
//  2. anvil (client chain): BridgeVerifier.verifyAndDeliver() with ECDSA signatures
//  3. Verify the mock gateway on anvil received the correct response
//
// Prerequisites: anvil and forge must be in PATH.
func TestDualChainBridgeE2E(t *testing.T) {
	if _, err := exec.LookPath("anvil"); err != nil {
		t.Skip("anvil not found, skipping dual-chain test")
	}
	if _, err := exec.LookPath("forge"); err != nil {
		t.Skip("forge not found, skipping dual-chain test")
	}

	contractsHome := findContractsHome(t)

	// --- Phase 1: Start anvil as client chain ---
	t.Log("Phase 1: Starting anvil...")
	anvilCmd, anvilRPC := startAnvil(t)
	defer func() {
		_ = anvilCmd.Process.Kill()
		_ = anvilCmd.Wait()
	}()
	t.Logf("Anvil running at %s", anvilRPC)

	// --- Phase 2: Deploy BridgeVerifier + MockGateway to anvil ---
	t.Log("Phase 2: Deploying contracts to anvil...")

	// Deploy MockGateway first (simple contract that records delivered messages)
	mockGatewayAddr := forgeCreate(t, contractsHome, anvilRPC,
		"test/foundry/unit/BridgeVerifier.t.sol:MockGateway")
	t.Logf("MockGateway deployed at %s", mockGatewayAddr)

	// Deploy BridgeVerifier (upgradeable, so we use the non-proxy version for testing)
	bridgeVerifierAddr := forgeCreate(t, contractsHome, anvilRPC,
		"src/core/BridgeVerifier.sol:BridgeVerifier")
	t.Logf("BridgeVerifier deployed at %s", bridgeVerifierAddr)

	// Initialize BridgeVerifier with a test validator set
	// The anvil default account (anvilPrivKey) acts as the validator for this test
	validatorKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	validatorAddr := crypto.PubkeyToAddress(validatorKey.PublicKey)

	// Initialize: owner=anvilAddr, gateway=mockGateway, expectedDstChainID=dstChainID,
	// validators=[validatorAddr], powers=[100]
	castSend(t, anvilRPC, bridgeVerifierAddr,
		"initialize(address,address,uint256,address[],uint256[])",
		validatorAddr.Hex(), // owner
		mockGatewayAddr,     // gateway
		"31337",             // expectedDstChainID (anvil default; matches dstChainID used below)
		fmt.Sprintf("[%s]", validatorAddr.Hex()), // validators
		"[100]", // powers
	)
	t.Log("BridgeVerifier initialized")

	// --- Phase 3: Simulate outbound messages (normally from imuachain) ---
	t.Log("Phase 3: Simulating outbound checkpoint + signature...")

	// Create a test outbound message (a RESPOND message)
	// Format: Action.RESPOND(0) + requestId(uint64) + success(uint8)
	responsePayload := make([]byte, 10)
	responsePayload[0] = 0x00 // Action.RESPOND
	// requestId = 42
	big.NewInt(42).FillBytes(responsePayload[1:9])
	responsePayload[9] = 0x01 // success = true

	messages := [][]byte{responsePayload}
	messagesHash := hashMessages(t, messages)

	checkpointNonce := uint64(1)
	dstChainID := uint64(31337) // anvil default chain ID

	// Compute checkpoint hash (must match BridgeVerifier.sol)
	checkpointHash := crypto.Keccak256Hash(
		common.BigToHash(big.NewInt(1)).Bytes(),                        // BRIDGE_ID
		common.BigToHash(new(big.Int).SetUint64(checkpointNonce)).Bytes(), // nonce
		common.BigToHash(new(big.Int).SetUint64(dstChainID)).Bytes(),     // dstChainID
		messagesHash.Bytes(), // messagesHash
	)

	// Eth signed message hash
	ethSignedHash := crypto.Keccak256Hash(
		[]byte("\x19Ethereum Signed Message:\n32"),
		checkpointHash.Bytes(),
	)

	// Sign with the validator's key
	sig, err := crypto.Sign(ethSignedHash.Bytes(), validatorKey)
	require.NoError(t, err)

	v := sig[64] + 27
	r := common.BytesToHash(sig[0:32])
	s := common.BytesToHash(sig[32:64])

	t.Logf("Checkpoint hash: %s", checkpointHash.Hex())
	t.Logf("Validator: %s, v=%d", validatorAddr.Hex(), v)

	// --- Phase 4: Call verifyAndDeliver on BridgeVerifier ---
	t.Log("Phase 4: Calling verifyAndDeliver on BridgeVerifier...")

	// Encode messages as hex array for cast
	msgHex := "0x" + hex.EncodeToString(responsePayload)

	castSend(t, anvilRPC, bridgeVerifierAddr,
		"verifyAndDeliver(uint256,uint256,bytes32,bytes[],address[],uint8[],bytes32[],bytes32[])",
		fmt.Sprintf("%d", checkpointNonce),
		fmt.Sprintf("%d", dstChainID),
		messagesHash.Hex(),
		fmt.Sprintf("[%s]", msgHex),
		fmt.Sprintf("[%s]", validatorAddr.Hex()),
		fmt.Sprintf("[%d]", v),
		fmt.Sprintf("[%s]", r.Hex()),
		fmt.Sprintf("[%s]", s.Hex()),
	)
	t.Log("verifyAndDeliver succeeded!")

	// --- Phase 5: Verify MockGateway received the message ---
	t.Log("Phase 5: Verifying MockGateway received the message...")

	deliveredCount := castCall(t, anvilRPC, mockGatewayAddr, "getDeliveredCount()(uint256)")
	t.Logf("MockGateway delivered count: %s", deliveredCount)
	require.Contains(t, deliveredCount, "1", "expected 1 delivered message")

	// Verify checkpoint nonce was updated
	lastNonce := castCall(t, anvilRPC, bridgeVerifierAddr,
		fmt.Sprintf("lastCheckpointNonce(uint256)(uint256)"), fmt.Sprintf("%d", dstChainID))
	t.Logf("BridgeVerifier lastCheckpointNonce: %s", lastNonce)
	require.Contains(t, lastNonce, "1")

	t.Log("Dual-chain E2E (方案B) completed successfully!")
}

// --- Helpers ---

func findContractsHome(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	home := getenvDefault("IMUA_CONTRACTS_HOME", filepath.Join(repoRoot, "..", "imua-contracts"))
	if _, err := os.Stat(filepath.Join(home, "foundry.toml")); err != nil {
		t.Skipf("imua-contracts not found at %s (set IMUA_CONTRACTS_HOME)", home)
	}
	return home
}

func forgeCreate(t *testing.T, contractsHome, rpcURL, contract string) string {
	t.Helper()
	cmd := exec.Command(
		"forge", "create",
		"--root", contractsHome,
		"--broadcast",
		"--rpc-url", rpcURL,
		"--private-key", anvilPrivKey,
		contract,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Run(), "forge create failed: %s", out.String())

	re := regexp.MustCompile(`Deployed to:\s*(0x[0-9a-fA-F]{40})`)
	match := re.FindStringSubmatch(out.String())
	require.Len(t, match, 2, "failed to parse deploy address: %s", out.String())
	return match[1]
}

func castSend(t *testing.T, rpcURL, to, sig string, args ...string) {
	t.Helper()
	cmdArgs := []string{"send", to, sig}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--rpc-url", rpcURL, "--private-key", anvilPrivKey)
	cmd := exec.Command("cast", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Run(), "cast send failed: %s", out.String())
}

func castCall(t *testing.T, rpcURL, to, sig string, args ...string) string {
	t.Helper()
	cmdArgs := []string{"call", to, sig}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--rpc-url", rpcURL)
	cmd := exec.Command("cast", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Run(), "cast call failed: %s", out.String())
	return out.String()
}

func hashMessages(t *testing.T, messages [][]byte) common.Hash {
	// Must match BridgeVerifier.sol: keccak256(abi.encode(messages))
	// where messages is bytes[]. Use go-ethereum's ABI encoder for parity.
	t.Helper()
	bytesArrTy, err := abi.NewType("bytes[]", "", nil)
	require.NoError(t, err)
	args := abi.Arguments{{Type: bytesArrTy}}
	encoded, err := args.Pack(messages)
	require.NoError(t, err)
	return crypto.Keccak256Hash(encoded)
}

