package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	pricefeeder "github.com/imua-xyz/price-feeder/external"
	"github.com/spf13/cobra"

	"github.com/cometbft/cometbft/libs/log"
)

const (
	flagOracle         = "oracle"
	flagFeederLogPath  = "feeder_log_path"
	flagFeederMnemonic = "feeder_mnemonic"
	flagFeederBinPath  = "feeder_bin"

	flagSourcesConfPath = "sources_path"
	flagConfFile        = "config"
	confOracle          = "oracle_feeder.yaml"
)

var feederPIDFile = filepath.Join(os.TempDir(), "feeder.pid")

func externalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external",
		Short: "External commands",
	}

	cmd.AddCommand(feederCommand())
	cmd.AddCommand(feederStopCommand())
	return cmd
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
			// TODO: refactor logger ?
			logger := sdkserver.GetServerContextFromCmd(cmd).Logger.With("module", "price-feeder")
			ok := pricefeeder.StartPriceFeeder(configFile, mnemonic, sourcesConfPath, logPath, logger)
			if !ok {
				return fmt.Errorf("failed to start price feeder")
			}
			return nil
		},
	}
	feederCmd.Flags().String(flagConfFile, "", "file of feeder config")
	feederCmd.Flags().String(flagSourcesConfPath, "", "path to sources config")
	feederCmd.Flags().String(flagFeederLogPath, "", "path to feeder logs")
	feederCmd.Flags().String(flagFeederMnemonic, "", "Oracle mnemonic")

	_ = feederCmd.MarkFlagRequired(flagConfFile)
	_ = feederCmd.MarkFlagRequired(flagSourcesConfPath)
	return feederCmd
}

func feederStopCommand() *cobra.Command {
	feederStopCmd := &cobra.Command{
		Use:   "feeder-stop",
		Short: "Stop the feeder subprocess started by --oracle",
		RunE: func(_ *cobra.Command, _ []string) error {
			pidData, err := os.ReadFile(feederPIDFile)
			if err != nil {
				return fmt.Errorf("failed to read feeder PID file: %w", err)
			}

			pidStr := strings.TrimSpace(string(pidData))
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				return fmt.Errorf("invalid PID in file: %s", pidStr)
			}

			proc, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("failed to find process: %w", err)
			}

			fmt.Printf("Sending SIGTERM to feeder (PID %d)...\n", pid)
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("failed to send SIGTERM: %w", err)
			}

			_ = os.Remove(feederPIDFile)
			fmt.Println("Feeder stopped successfully.")
			return nil
		},
	}
	return feederStopCmd
}

func launchFeeder(configFile, sourcesConfPath, binPath, mnemonic string, logger log.Logger, logPath string) {
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

			cmd := exec.Command(selfPath,
				"external",
				"feeder",
				fmt.Sprintf("--%s", flagConfFile), configFile,
				fmt.Sprintf("--%s", flagSourcesConfPath), sourcesConfPath,
				fmt.Sprintf("--%s", flagFeederLogPath), logPath,
				fmt.Sprintf("--%s", flagFeederMnemonic), mnemonic,
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				logger.Error("failed to start feeder subprocess", "err", err)
				return
			}

			logger.Info("feeder subprocess started", "pid", cmd.Process.Pid)

			_ = os.WriteFile(
				feederPIDFile,
				[]byte(fmt.Sprintf("%d", cmd.Process.Pid)),
				0o600,
			)

			go func() {
				err := cmd.Wait()
				logger.Error("feeder subprocess exited", "err", err)
			}()
		}
	}()
}

func startExternalFeeder(binPath, configFile, sourcesConfPath string, logger log.Logger, logPath string) {
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
		//		}

		if err := cmd.Start(); err != nil {
			logger.Error("failed to start external feeder", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info("external feeder started", "pid", cmd.Process.Pid)

		// write PID file
		_ = os.WriteFile(feederPIDFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0600)

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
