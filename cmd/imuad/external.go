package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	pricefeeder "github.com/imua-xyz/price-feeder/external"
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

			if statusPort <= 0 || statusPort > 65535 {
				logger.Error("invalid status port", "statusPort", statusPort)
				return
			}
			statusPortStr := strconv.Itoa(statusPort)

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

			if err := cmd.Start(); err != nil {
				logger.Error("failed to start feeder subprocess", "err", err)
				return
			}

			logger.Info("feeder subprocess started", "pid", cmd.Process.Pid)

			go func() {
				err := cmd.Wait()
				logger.Error("feeder subprocess exited", "err", err)
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
		args := []string{
			"--config", configFile,
			"--sources_path", sourcesConfPath,
		}
		if len(logPath) > 0 {
			args = append(args, "--log_path", logPath)
		} else {
			args = append(args, "--log_imua_format=true")
		}

		args = append(args, "start")
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
