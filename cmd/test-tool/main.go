package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ExocoreNetwork/exocore/testutil/batch"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string

// Root command
var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "test tool application with external configuration",
	Long:  `This is a test tool application that loads configuration from an external file.`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		// Initialize the manager before executing any command
		var err error
		config, err := loadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %v", err)
		}
		// Initialize the manager with the provided configuration file
		appManager, err = batch.NewManager(context.Background(), config)
		if err != nil {
			return fmt.Errorf("failed to initialize manager: %v", err)
		}
		return nil
	},
}

// Global appManager variable to access the manager in subcommands
var appManager *batch.Manager

// start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the test tool",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		// Start the app manager in a separate goroutine
		go func() {
			if err := appManager.Start(); err != nil {
				fmt.Printf("Error starting the test tool: %v\n", err)
			}
		}()

		// Set up channel to listen for OS interrupt signals (like Ctrl+C)
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for an interrupt signal
		<-stopChan
		// Shutdown appManager gracefully
		fmt.Println("Shutting down...")
		close(appManager.Shutdown)
		appManager.Close()
	},
}

// loadConfig loads the configuration file and parses it into the Config struct
func loadConfig(configPath string) (*batch.EndToEndConfig, error) {
	// Set the config file path and type (can be "yaml", "json", etc.)
	viper.SetConfigFile(configPath)

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file, %s", err)
	}

	// Unmarshal the config into a Config struct
	var cfg batch.EndToEndConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return &cfg, nil
}

func main() {
	// Add persistent flag for the configuration file
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "test-tool-config.toml", "Path to the configuration file")

	// Add subcommands
	rootCmd.AddCommand(startCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
