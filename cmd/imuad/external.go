package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	pricefeeder "github.com/imua-xyz/price-feeder/external"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/cometbft/cometbft/libs/log"
)

const (
	flagOracle                 = "oracle"
	flagFeederLogPath          = "feeder_log_path"
	flagFeederMnemonic         = "feeder_mnemonic"
	flagFeederBinPath          = "feeder_bin"
	flagFeederStatusGRPCAddr   = "grpc_addr"
	flagFeederStatusListenPort = "status_port"

	flagSourcesConfPath = "sources_path"
	flagConfFile        = "config"
	confOracle          = "oracle_feeder.yaml"

	defaultStatusGRPCAddr = "localhost:50052"
)

var mnemonicWordRegexp = regexp.MustCompile(`^[a-z]+$`)

func externalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external",
		Short: "External commands",
	}

	cmd.AddCommand(feederCommand())
	cmd.AddCommand(feederStatusCommand())
	return cmd
}

func feederStatusCommand() *cobra.Command {
	feederStatusCmd := &cobra.Command{
		Use:   "feeder-status",
		Short: "Check tokens status of the embedded oracle price feeder",
		RunE: func(cmd *cobra.Command, _ []string) error {
			grpcAddr, _ := cmd.Flags().GetString(flagFeederStatusGRPCAddr)
			if grpcAddr == "" {
				grpcAddr = defaultStatusGRPCAddr
			}
			res, err := pricefeeder.GetAllTokens(grpcAddr)
			if err != nil {
				message, err := pricefeeder.FilterErrors(err)
				if err != nil {
					return err
				}
				fmt.Println("fetching token status:", message)
				return nil
			}
			printProto(res)
			return nil
		},
	}
	feederStatusCmd.Flags().String(flagFeederStatusGRPCAddr, "", "gRPC address to connect to the price feeder for status check")
	return feederStatusCmd
}

func feederCommand() *cobra.Command {
	feederCmd := &cobra.Command{
		Use:   "feeder",
		Short: "Start the embedded oracle price feeder as a subprocess",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFile, _ := cmd.Flags().GetString(flagConfFile)
			sourcesConfPath, _ := cmd.Flags().GetString(flagSourcesConfPath)
			logPath, _ := cmd.Flags().GetString(flagFeederLogPath)
			mnemonic, _ := cmd.Flags().GetString(flagFeederMnemonic)
			statusPort, _ := cmd.Flags().GetInt(flagFeederStatusListenPort)
			// TODO: refactor logger ?
			logger := sdkserver.GetServerContextFromCmd(cmd).Logger.With("module", "price-feeder")
			err := pricefeeder.StartPriceFeeder(configFile, mnemonic, sourcesConfPath, logPath, statusPort, logger)
			if err != nil {
				return fmt.Errorf("failed to start price feeder, err: %w", err)
			}
			return nil
		},
	}
	feederCmd.Flags().String(flagConfFile, "", "file of feeder config")
	feederCmd.Flags().String(flagSourcesConfPath, "", "path to sources config")
	feederCmd.Flags().String(flagFeederLogPath, "", "path to feeder logs")
	feederCmd.Flags().String(flagFeederMnemonic, "", "Oracle mnemonic")
	feederCmd.Flags().Int(flagFeederStatusListenPort, 0, "Port for the feeder status gRPC server")

	_ = feederCmd.MarkFlagRequired(flagConfFile)
	_ = feederCmd.MarkFlagRequired(flagSourcesConfPath)
	return feederCmd
}

func launchFeeder(configFile, sourcesConfPath, binPath, mnemonic string, logger log.Logger, logPath string, statusPort int) {
	if binPath != "" {
		if err := validatePath(binPath, true); err != nil {
			logger.Error("invalid feeder binary path", "path", binPath, "err", err)
			return
		}
		if err := isSafeExecutable(binPath); err != nil {
			logger.Error("feeder binary is not safe to execute", "path", binPath, "err", err)
			return
		}
	} else if err := validateFeederInputs(configFile, sourcesConfPath, logPath, mnemonic, strconv.Itoa(statusPort)); err != nil {
		logger.Error("invalid feeder inputs", "err", err)
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("feeder panic recovered", "err", r)
			}
		}()
		if binPath != "" {
			logger.Info("starting external feeder binary", "path", binPath)
			startExternalFeeder(binPath, configFile, sourcesConfPath, logger, logPath)
		} else {
			logger.Info("starting feeder subprocess via CLI command")
			selfPath, err := os.Executable()
			if err != nil {
				logger.Error("cannot determine self binary path", "err", err)
				return
			}
			statusPortStr := strconv.Itoa(statusPort)
			// nosemgrep
			cmd := exec.Command(selfPath,
				"external",
				"feeder",
				"--"+flagConfFile, configFile,
				"--"+flagSourcesConfPath, sourcesConfPath,
				"--"+flagFeederLogPath, logPath,
				"--"+flagFeederMnemonic, mnemonic,
				"--"+flagFeederStatusListenPort, statusPortStr,
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sig
				if cmd.Process != nil {
					if err := cmd.Process.Kill(); err != nil {
						logger.Error("failed to kill process", "pid", cmd.Process.Pid, "err", err)
					} else {
						logger.Info("feeder subprocess terminated by signal", "pid", cmd.Process.Pid)
					}
				}
			}()
			if err := cmd.Start(); err != nil {
				logger.Error("failed to start feeder subprocess", "err", err)
				return
			}

			logger.Info("feeder subprocess started", "pid", cmd.Process.Pid)

			go func() {
				err := cmd.Wait()
				if err != nil {
					logger.Error("feeder subprocess exited with error", "err", err)
				} else {
					logger.Info("feeder subprocess exited normally")
				}
			}()
		}
	}()
}

func startExternalFeeder(binPath, configFile, sourcesConfPath string, logger log.Logger, logPath string) {
	// validate binary path
	if _, err := os.Stat(binPath); err != nil {
		logger.Error("feeder binary not found", "path", binPath, "err", err)
		return
	}
	for retry := 0; ; retry++ {
		if err := validatePath(configFile, true); err != nil {
			logger.Error("invalid config file path", "path", configFile, "err", err)
			return
		}
		if err := validatePath(sourcesConfPath, true); err != nil {
			logger.Error("invalid sources config path", "path", sourcesConfPath, "err", err)
			return
		}
		args := []string{
			"--config", configFile,
			"--sources_path", sourcesConfPath,
		}
		if len(logPath) > 0 {
			if err := validatePath(logPath, true); err != nil {
				logger.Error("invalid log path", "path", logPath, "err", err)
				return
			}
			args = append(args, "--log_path", logPath)
		} else {
			args = append(args, "--log_imua_format=true")
		}

		args = append(args, "start")
		// nosemgrep
		cmd := exec.Command(binPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			logger.Error("failed to start external feeder", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info("external feeder started", "pid", cmd.Process.Pid)

		err := cmd.Wait()
		logger.Error("external feeder exited", "err", err)

		time.Sleep(backoff(retry))
	}
}

func backoff(retry int) time.Duration {
	d := time.Duration(5+retry*2) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func printProto(m proto.Message) {
	if m == nil {
		fmt.Println("nil proto message")
		return
	}
	marshaled, err := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}.Marshal(m)
	if err != nil {
		fmt.Printf("failed to print proto message, error:%v", err)
	}
	fmt.Println(string(marshaled))
}

func validatePath(p string, required bool) error {
	if !required && len(p) == 0 {
		return nil
	}
	if len(p) == 0 {
		return errors.New("path is empty")
	}
	if strings.ContainsAny(p, "&|;$<>`\\\"'*?[]{}()~") {
		return fmt.Errorf("invalid characters in path:%s", p)
	}
	return nil
}

func validateMnemonic(m string, required bool) error {
	if !required && len(m) == 0 {
		return nil
	}
	words := strings.Fields(m)
	if len(words) != 12 && len(words) != 24 {
		return fmt.Errorf("invalid mnemonic length: %d, expected 12 or 24 words", len(words))
	}
	for _, word := range words {
		if !mnemonicWordRegexp.MatchString(word) {
			return fmt.Errorf("invalid mnemonic word: %s", word)
		}
	}
	return nil
}

func validatePort(portStr string, required bool) error {
	if !required && (len(portStr) == 0 || portStr == "0") {
		return nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s, must be a number", portStr)
	}

	if port < 1024 || port > 65535 {
		return fmt.Errorf("invalid port: %d, must be in range 1024-65535", port)
	}
	return nil
}

func validateFeederInputs(configFile, sourcesConfPath, logPath, mnemonic, statusPortStr string) error {
	validators := []struct {
		name string
		err  error
	}{
		{"configFile", validatePath(configFile, true)},
		{"sourcesConfPath", validatePath(sourcesConfPath, true)},
		{"logPath", validatePath(logPath, false)},
		{"mnemonic", validateMnemonic(mnemonic, false)},
		{"statusPortStr", validatePort(statusPortStr, false)},
	}

	for _, v := range validators {
		if v.err != nil {
			return fmt.Errorf("invalid %s: %w", v.name, v.err)
		}
	}
	return nil
}

func isSafeExecutable(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	if !fi.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	if fi.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("file is not executable")
	}

	if fi.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("file is world-writable")
	}

	return nil
}
