package oracle

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil/network"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	anvilMnemonic = "test test test test test test test test test test test junk"
	anvilPrivKey  = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

type XChainPriceFeederSuite struct {
	suite.Suite
	cfg     network.Config
	network *network.Network
}

func TestXChainPriceFeederIntegration(t *testing.T) {
	shouldSkipXChainPriceFeederIntegration(t)
	ensureXChainGatewayGenesis()
	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	cfg.CleanupDir = true
	cfg.EnableTMLogging = true
	oracleGen := PrepareXChainOracleGenesis(&cfg)
	cfg.OracleGenesisState = &oracleGen
	suite.Run(t, &XChainPriceFeederSuite{cfg: cfg})
}

func shouldSkipXChainPriceFeederIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_XCHAIN_PRICEFEEDER_IT") == "1" {
		return
	}
	if _, err := exec.LookPath("anvil"); err != nil {
		t.Skipf("anvil not found: %v (set RUN_XCHAIN_PRICEFEEDER_IT=1 to require)", err)
	}
	if _, err := exec.LookPath("forge"); err != nil {
		t.Skipf("forge not found: %v (set RUN_XCHAIN_PRICEFEEDER_IT=1 to require)", err)
	}
	if _, err := exec.LookPath("cast"); err != nil {
		t.Skipf("cast not found: %v (set RUN_XCHAIN_PRICEFEEDER_IT=1 to require)", err)
	}
	repoRoot := findRepoRoot(t)
	priceFeederHome := getenvDefault("PRICE_FEEDER_HOME", filepath.Join(repoRoot, "..", "price-feeder"))
	if _, err := os.Stat(priceFeederHome); err != nil {
		t.Skipf("price-feeder repo not found at %s: %v (set PRICE_FEEDER_HOME or RUN_XCHAIN_PRICEFEEDER_IT=1)", priceFeederHome, err)
	}
	contractsHome := getenvDefault("IMUA_CONTRACTS_HOME", filepath.Join(repoRoot, "..", "imua-contracts"))
	if _, err := os.Stat(contractsHome); err != nil {
		t.Skipf("imua-contracts repo not found at %s: %v (set IMUA_CONTRACTS_HOME or RUN_XCHAIN_PRICEFEEDER_IT=1)", contractsHome, err)
	}
}

func (s *XChainPriceFeederSuite) SetupSuite() {
	s.T().Log("setting up xchain price-feeder integration suite")
	var err error
	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)
	_, err = s.network.WaitForHeightWithTimeout(2, 20*time.Second)
	s.Require().NoError(err)
}

func (s *XChainPriceFeederSuite) TearDownSuite() {
	s.network.Cleanup()
}

func (s *XChainPriceFeederSuite) TestCrossChainPriceFeederDepositLST() {
	ctx := context.Background()
	paramsRes, err := s.network.QueryOracle().Params(ctx, &oracletypes.QueryParamsRequest{})
	s.Require().NoError(err)

	_, startBaseBlock, interval := s.mustGetXChainFeeder(paramsRes.Params)
	oracleCaller := common.BytesToAddress(authtypes.NewModuleAddress(oracletypes.ModuleName))
	gatewayAddr := s.deployOracleGateway(oracleCaller)
	s.Require().Equal(strings.ToLower(network.ExpectedOracleGatewayAddress().Hex()), strings.ToLower(gatewayAddr.Hex()))

	anvilCmd, rpcURL := startAnvil(s.T())
	s.T().Cleanup(func() { stopProcess(s.T(), anvilCmd) })

	repoRoot := findRepoRoot(s.T())
	priceFeederHome := getenvDefault("PRICE_FEEDER_HOME", filepath.Join(repoRoot, "..", "price-feeder"))
	contractsHome := getenvDefault("IMUA_CONTRACTS_HOME", filepath.Join(repoRoot, "..", "imua-contracts"))

	emitterAddr := deployEmitter(s.T(), contractsHome, rpcURL)

	stakerAddr := common.BytesToAddress(s.network.Validators[0].Address.Bytes())
	tokenAddr := common.HexToAddress(network.ETHAssetAddress)
	opAmount := sdkmath.NewInt(1000)

	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(
		network.TestEVMChainID,
		stakerAddr.Bytes(),
		tokenAddr.Bytes(),
	)
	before := s.queryStakerTotalDeposited(ctx, stakerID, assetID)

	tmpDir := s.T().TempDir()
	configPath, sourcesPath := writePriceFeederConfig(
		s.T(),
		tmpDir,
		priceFeederHome,
		s.network,
		rpcURL,
		emitterAddr,
		network.TestEVMChainID,
	)

	pfBin := buildPriceFeeder(s.T(), priceFeederHome, tmpDir)
	pfCmd := startPriceFeeder(s.T(), pfBin, configPath, sourcesPath)
	s.T().Cleanup(func() { stopProcess(s.T(), pfCmd) })

	payload := buildDepositLSTMessage(stakerAddr, tokenAddr, opAmount.BigInt())
	msgID := common.BytesToHash([]byte("xchain:deposit:0"))
	emitXChainMessage(s.T(), rpcURL, emitterAddr, 1, msgID, payload)

	currentHeight, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	baseBlock := s.nextBaseBlock(startBaseBlock, interval)
	if currentHeight < baseBlock {
		s.moveToAndCheck(baseBlock)
	}
	currentHeight, err = s.network.LatestStateHeight()
	s.Require().NoError(err)

	phaseTwoStart := baseBlock + int64(paramsRes.Params.MaxNonce)
	if currentHeight < phaseTwoStart {
		s.moveToAndCheck(phaseTwoStart)
	}
	s.moveNAndCheck(3)

	s.Require().Eventually(func() bool {
		seq := s.queryXChainLastExecutedSeq(network.TestEVMChainID)
		s.T().Logf("waiting for xchain execution: lastExecutedSeq=%d", seq)
		return seq >= 1
	}, 2*time.Minute, 2*time.Second)

	s.Require().Eventually(func() bool {
		after := s.queryStakerTotalDeposited(ctx, stakerID, assetID)
		delta := after.Sub(before)
		s.T().Logf("waiting for deposit: before=%s after=%s delta=%s target=%s", before, after, delta, opAmount)
		return delta.Equal(opAmount)
	}, 2*time.Minute, 2*time.Second)
}

func startAnvil(t *testing.T) (*exec.Cmd, string) {
	t.Helper()
	if _, err := exec.LookPath("anvil"); err != nil {
		t.Fatalf("anvil not found in PATH: %v", err)
	}
	port := freePort(t)
	cmd := exec.Command("anvil",
		"--chain-id", fmt.Sprintf("%d", network.TestEVMChainID),
		"--mnemonic", anvilMnemonic,
		"--port", fmt.Sprintf("%d", port),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	rpcURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	require.NoError(t, waitForPort(fmt.Sprintf("127.0.0.1:%d", port), 10*time.Second))
	return cmd, rpcURL
}

func deployEmitter(t *testing.T, contractsHome, rpcURL string) string {
	t.Helper()
	if _, err := exec.LookPath("forge"); err != nil {
		t.Fatalf("forge not found in PATH: %v", err)
	}
	contract := "src/test/XChainMessageEmitter.sol:XChainMessageEmitter"
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
	require.NoError(t, cmd.Run(), out.String())

	re := regexp.MustCompile(`Deployed to:\s*(0x[0-9a-fA-F]{40})`)
	match := re.FindStringSubmatch(out.String())
	require.Len(t, match, 2, "failed to parse deploy address: %s", out.String())
	return match[1]
}

func emitXChainMessage(t *testing.T, rpcURL, emitterAddr string, nonce uint64, msgID common.Hash, payload []byte) {
	t.Helper()
	if _, err := exec.LookPath("cast"); err != nil {
		t.Fatalf("cast not found in PATH: %v", err)
	}
	payloadHex := "0x" + hex.EncodeToString(payload)
	cmd := exec.Command(
		"cast", "send",
		emitterAddr,
		"emitMessage(uint64,bytes32,bytes)",
		fmt.Sprintf("%d", nonce),
		msgID.Hex(),
		payloadHex,
		"--rpc-url", rpcURL,
		"--private-key", anvilPrivKey,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

func writePriceFeederConfig(t *testing.T, tmpDir, priceFeederHome string, netw *network.Network, xchainRPC, gateway string, srcChainID uint64) (string, string) {
	t.Helper()
	sourcesPath := filepath.Join(tmpDir, "sources")
	require.NoError(t, os.MkdirAll(sourcesPath, 0o755))

	configPath := filepath.Join(tmpDir, "config.yaml")
	privPath := filepath.Join(netw.Validators[0].Dir, "imuad", "config")
	imuaRPC := toHTTP(netw.Validators[0].RPCAddress)
	wsURL := toWS(netw.Validators[0].RPCAddress)
	grpcAddr := toGRPC(netw.Validators[0].AppConfig.GRPC.Address)
	statusPort := freePort(t)

	config := fmt.Sprintf(`tokens:
  - token: xchain_%d
    sources: xchain
sender:
  path: %s
imua:
  chainid: %s
  appname: imua
  grpc: %s
  ws: %s
  rpc: %s
status:
  grpc: %d
log:
  level: info
`, srcChainID, privPath, netw.Config.ChainID, grpcAddr, wsURL, imuaRPC, statusPort)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	abiPath := filepath.Join(priceFeederHome, "fetcher/xchain/xchain_gateway_abi.json")
	xchainCfg := fmt.Sprintf(`abi_path: %s
event_name: XChainMessage
tokens:
  xchain_%d:
    rpc: %s
    gateway: %s
    src_chain_id: %d
    start_block: 0
    start_nonce: 1
    start_batch_seq: 1
    confirmations: 0
    max_messages: 200
    max_bytes: 2097152
    max_blocks: 2000
`, abiPath, srcChainID, xchainRPC, gateway, srcChainID)
	require.NoError(t, os.WriteFile(filepath.Join(sourcesPath, "oracle_env_xchain.yaml"), []byte(xchainCfg), 0o644))

	return configPath, sourcesPath
}

func buildPriceFeeder(t *testing.T, priceFeederHome, tmpDir string) string {
	t.Helper()
	binPath := filepath.Join(tmpDir, "price-feeder")
	cmd := exec.Command("go", "build", "-o", binPath, "./")
	cmd.Dir = priceFeederHome
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
	return binPath
}

func startPriceFeeder(t *testing.T, binPath, configPath, sourcesPath string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(binPath, "--config", configPath, "--sources_path", sourcesPath, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	return cmd
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

func toHTTP(rpcListen string) string {
	addr := strings.TrimPrefix(rpcListen, "tcp://")
	addr = strings.Replace(addr, "0.0.0.0", "127.0.0.1", 1)
	return "http://" + addr
}

func toWS(rpcListen string) string {
	addr := strings.TrimPrefix(rpcListen, "tcp://")
	addr = strings.Replace(addr, "0.0.0.0", "127.0.0.1", 1)
	return "ws://" + addr + "/websocket"
}

func toGRPC(grpcListen string) string {
	addr := strings.Replace(grpcListen, "0.0.0.0", "127.0.0.1", 1)
	return addr
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func limitOracleFeedersToXChain() {
	params := &network.DefaultGenStateOracle.Params
	var xchainFeeder *oracletypes.TokenFeeder
	for _, tf := range params.TokenFeeders {
		if tf == nil {
			continue
		}
		if int(tf.TokenID) >= len(params.Tokens) {
			continue
		}
		token := params.Tokens[tf.TokenID]
		if strings.HasPrefix(strings.ToLower(token.AssetID), oracletypes.XChainIDPrefix) {
			xchainFeeder = tf
			break
		}
	}
	if xchainFeeder != nil {
		params.TokenFeeders = []*oracletypes.TokenFeeder{{}, xchainFeeder}
	}
}

func (s *XChainPriceFeederSuite) moveToAndCheck(height int64) {
	_, err := s.network.WaitForStateHeightWithTimeout(height, 120*time.Second)
	s.Require().NoError(err)
}

func (s *XChainPriceFeederSuite) moveNAndCheck(n int64) {
	for i := int64(0); i < n; i++ {
		err := s.network.WaitForStateNextBlock()
		s.Require().NoError(err)
	}
}

func (s *XChainPriceFeederSuite) mustGetXChainFeeder(params oracletypes.Params) (uint64, uint64, uint64) {
	for feederID, feeder := range params.TokenFeeders {
		if feederID == 0 || feeder == nil {
			continue
		}
		if int(feeder.TokenID) >= len(params.Tokens) {
			continue
		}
		token := params.Tokens[feeder.TokenID]
		if strings.HasPrefix(strings.ToLower(token.AssetID), oracletypes.XChainIDPrefix) {
			return uint64(feederID), feeder.StartBaseBlock, feeder.Interval
		}
	}
	s.FailNow("xchain feeder not found in params")
	return 0, 0, 0
}

func (s *XChainPriceFeederSuite) nextBaseBlock(startBaseBlock uint64, interval uint64) int64 {
	height, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	if height <= int64(startBaseBlock) {
		return int64(startBaseBlock)
	}
	if interval == 0 {
		return height
	}
	delta := height - int64(startBaseBlock)
	rounds := delta / int64(interval)
	baseBlock := int64(startBaseBlock) + rounds*int64(interval)
	if baseBlock < height {
		baseBlock += int64(interval)
	}
	return baseBlock
}

func (s *XChainPriceFeederSuite) deployOracleGateway(oracleCaller common.Address) common.Address {
	addr, err := s.network.DeployOracleGatewayContract(oracleCaller)
	s.Require().NoError(err)
	s.moveNAndCheck(1)
	return addr
}

func (s *XChainPriceFeederSuite) queryStakerTotalDeposited(ctx context.Context, stakerID, assetID string) sdkmath.Int {
	res, err := s.network.QueryAssets().QueryStakerBalance(ctx, &assetstypes.QueryStakerBalanceRequest{
		StakerId: stakerID,
		AssetId:  assetID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(res.StakerBalance)
	return res.StakerBalance.TotalDeposited
}

func (s *XChainPriceFeederSuite) queryXChainLastExecutedSeq(srcChainID uint64) uint64 {
	value := s.queryOracleStore(oracletypes.XChainLastExecutedSeqKey(srcChainID))
	if len(value) == 0 {
		return 0
	}
	seq, err := oracletypes.BytesToUint64(value)
	s.Require().NoError(err)
	return seq
}

func (s *XChainPriceFeederSuite) queryOracleStore(key []byte) []byte {
	res, err := s.network.Validators[0].RPCClient.ABCIQuery(context.Background(), "/store/oracle/key", key)
	s.Require().NoError(err)
	return res.Response.Value
}
