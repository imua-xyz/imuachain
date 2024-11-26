package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"

	"github.com/ExocoreNetwork/exocore/testutil/batch"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var homePath string

// Root command
var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "test tool application with external configuration",
	Long:  `This is a test tool application that loads configuration from an external file.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Check if the subcommand is "init"
		if cmd.Name() == initCmd.Name() {
			// Skip manager initialization for the "init" command
			return nil
		}
		// Initialize the manager before executing any command
		var err error
		config, err := loadConfig(filepath.Join(homePath, batch.ConfigFileName))
		if err != nil {
			return fmt.Errorf("failed to load config: %v", err)
		}

		// Initialize the manager with the provided configuration file
		appManager, err = batch.NewManager(context.Background(), homePath, config)
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

// init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init the default config for the test tool",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		configFilePath := filepath.Join(homePath, batch.ConfigFileName)
		// Create or open the configuration file
		file, err := os.Create(configFilePath)
		if err != nil {
			fmt.Printf("failed to create config file: %s", err)
		}
		defer file.Close()

		// Serialize the default configuration to TOML format
		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(batch.DefaultTestToolConfig); err != nil {
			fmt.Printf("failed to encode config to TOML: %err", err)
		}
	},
}

// loadConfig loads the configuration file and parses it into the Config struct
func loadConfig(configPath string) (*batch.TestToolConfig, error) {
	// Set the config file path and type (can be "yaml", "json", etc.)
	viper.SetConfigFile(configPath)

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file, %s", err)
	}

	// Unmarshal the config into a Config struct
	var cfg batch.TestToolConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return &cfg, nil
}

func main() {
	// Add persistent flag for the configuration file
	rootCmd.PersistentFlags().StringVar(&homePath, "home", ".", "Path to the config, db and keyRing file")
	// Add subcommands
	rootCmd.AddCommand(startCmd, initCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
