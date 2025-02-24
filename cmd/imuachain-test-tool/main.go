package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/imua-xyz/imuachain/testutil/batch"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	homePath string
	sqliteDB *gorm.DB
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "test tool application with external configuration",
	Long:  `This is a test tool application that loads configuration from an external file.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Check the subcommand
		switch cmd.Name() {
		case initCmd.Name():
			return nil
		case QueryTxRecordCmd.Name(),
			QueryTestObjectsCmd.Name(),
			QueryHelperRecordCmd.Name():
			// only initialize the sqlite db
			// open the sqlite db
			dsn := "file:" + filepath.Join(homePath, batch.SqliteDBFileName) + "?cache=shared&_journal_mode=WAL"
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			if err != nil {
				return xerrors.Errorf("can't open sqlite db, err:%s", err)
			}
			// SQLite waits for 600000 milliseconds (10 minute) when encountering a lock conflict.
			db.Exec("PRAGMA busy_timeout = 600000;")
			sqliteDB = db
			return nil
		}
		// Initialize the manager before executing any tx commands
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

// init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init the default config for the test tool",
	Long: "init the default config for the test tool, using test-tool-config.toml " +
		"as the default name of the config file",
	Example: "imuachain-test-tool init --home .",
	Args:    cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		configFilePath := filepath.Join(homePath, batch.ConfigFileName)
		// Create or open the configuration file
		file, err := os.Create(configFilePath)
		if err != nil {
			fmt.Printf("failed to create config file: %s\r\n", err)
			return
		}
		defer file.Close()

		// Serialize the default configuration to TOML format
		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(batch.DefaultTestToolConfig); err != nil {
			fmt.Printf("failed to encode config to TOML: %s\r\n", err)
			return
		}
	},
}

// start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the test tool",
	Long: "Start the testing tool to automatically perform preparation steps " +
		"and batch tests for multiple message types.",
	Example: "imuachain-test-tool start --home .",
	Args:    cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		// Start the app manager in a separate goroutine
		go func() {
			if err := appManager.Start(); err != nil {
				fmt.Printf("Error starting the test tool: %s\r\n", err)
			}
		}()
		appManager.WaitForShuttingDown()
	},
}

// prepare command
var prepareCmd = &cobra.Command{
	Use:     "prepare",
	Short:   "prepare for the batch test",
	Long:    "prepare the test objects, funding, registration and opting-in for the test tool",
	Example: "imuachain-test-tool prepare --home .",
	Args:    cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		if err := appManager.Prepare(); err != nil {
			fmt.Printf("failed to prepare for the test tool: %s\r\n", err)
		}
		appManager.Close()
	},
}

// batch test command
var batchTestCmd = &cobra.Command{
	Use:   "batch-test <msgType>",
	Short: "batch test",
	Long: "batch test the multiple functions, the msgType should be: \r\n" +
		"depositLST,delegate,undelegate and withdrawLST",
	Example: "imuachain-test-tool batch-test depositLST --home .",
	Args:    cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		// Start the app manager in a separate goroutine
		go func() {
			if err := appManager.ExecuteBatchTestForType(args[0]); err != nil {
				fmt.Printf("failed to execute the batch test for the specified message type,"+
					"msgType:%s,err: %s\r\n", args[0], err)
			}
		}()
		appManager.WaitForShuttingDown()
	},
}

// QueryHelperRecordCmd query the helper record info, the current batch id can be queried by this command.
var QueryHelperRecordCmd = &cobra.Command{
	Use:     "query-helper-record",
	Short:   "query the helper record info",
	Long:    "query the helper record info, the info includes: current-batch-id",
	Example: "imuachain-test-tool query-helper-record --home .",
	Args:    cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		helperRecord, err := batch.LoadObjectByID[batch.HelperRecord](sqliteDB, batch.SqliteDefaultStartID)
		if err != nil {
			fmt.Printf("failed to load the helper record, err:%s\r\n", err)
			return
		}
		err = batch.PrintObject(helperRecord)
		if err != nil {
			fmt.Printf("failed to print the helper record, err:%s\r\n", err)
		}
	},
}

// QueryTestObjectsCmd query the test objects that have been created.
var QueryTestObjectsCmd = &cobra.Command{
	Use:     "query-test-objects <object>",
	Short:   "query the specified test objects",
	Long:    "query the specified test objects, the object type is: asset, staker, operator and AVS",
	Example: "imuachain-test-tool query-test-objects staker --home .",
	Args:    cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "asset":
			opFunc := func(id uint, objectNumber int64, obj batch.Asset) error {
				if id == uint(1) {
					fmt.Printf("the number of %s is: %d\r\n", args[0], objectNumber)
				}
				return batch.PrintObject(obj)
			}
			err = batch.IterateObjects(sqliteDB, batch.Asset{}, opFunc)
		case "staker":
			opFunc := func(id uint, objectNumber int64, obj batch.Staker) error {
				if id == uint(1) {
					fmt.Printf("the number of %s is: %d\r\n", args[0], objectNumber)
				}
				return batch.PrintObject(obj)
			}
			err = batch.IterateObjects(sqliteDB, batch.Staker{}, opFunc)
		case "operator":
			opFunc := func(id uint, objectNumber int64, obj batch.Operator) error {
				if id == uint(1) {
					fmt.Printf("the number of %s is: %d\r\n", args[0], objectNumber)
				}
				return batch.PrintObject(obj)
			}
			err = batch.IterateObjects(sqliteDB, batch.Operator{}, opFunc)
		case "AVS":
			opFunc := func(id uint, objectNumber int64, obj batch.AVS) error {
				if id == uint(1) {
					fmt.Printf("the number of %s is: %d\r\n", args[0], objectNumber)
				}
				return batch.PrintObject(obj)
			}
			err = batch.IterateObjects(sqliteDB, batch.AVS{}, opFunc)
		default:
			fmt.Printf("invalid object type:%s\r\n", args[0])
			return
		}
		if err != nil {
			fmt.Printf("failed to iterate the test objects, type:%s,err:%s\r\n", args[0], err)
		}
	},
}

// QueryTxRecordCmd queries the transaction record for the specified batch ID and status.
// This can be used to check the status of the batch test.
var QueryTxRecordCmd = &cobra.Command{
	Use:   "query-tx-record <msgType> <batch-id> <status>",
	Short: "query the transaction record",
	Long: "query the transaction record according to the specified batch ID and status\r\n" +
		"the status includes: \r\n" +
		"0: Queued\r\n" +
		"1: pending\r\n" +
		"2: OnChainButFailed\r\n" +
		"3: OnChainAndSuccessful",
	Example: "imuachain-test-tool query-tx-record depositLST 1 1 --home .",
	Args:    cobra.ExactArgs(3),
	Run: func(_ *cobra.Command, args []string) {
		batchID, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			fmt.Printf("invalid batch id,input:%s,err:%s", args[1], err)
			return
		}
		helperRecord, err := batch.LoadObjectByID[batch.HelperRecord](sqliteDB, batch.SqliteDefaultStartID)
		if err != nil {
			fmt.Printf("failed to load the helper record, err:%s\r\n", err)
			return
		}
		if uint(batchID) > helperRecord.CurrentBatchID {
			fmt.Printf("invalid batch id, inputBatchID:%d,currentBatchID:%d\r\n", batchID, helperRecord.CurrentBatchID)
			return
		}
		status, err := strconv.ParseInt(args[2], 10, 32)
		if err != nil {
			fmt.Printf("invalid status,input:%s,err:%s", args[2], err)
			return
		}
		if int(status) > batch.OnChainAndSuccessful || status < 0 {
			fmt.Printf("invalid status,status:%d", status)
			return
		}
		txIDs, count, err := batch.GetTxIDsByBatchTypeAndStatus(sqliteDB, uint(batchID), args[0], int(status))
		if err != nil {
			fmt.Printf("failed to get tx IDs,err:%s", err)
			return
		}
		fmt.Println("the number of tx record is:", count)
		for _, txID := range txIDs {
			txRecord, err := batch.LoadObjectByID[batch.Transaction](sqliteDB, txID)
			if err != nil {
				fmt.Printf("failed to load the tx record,txID:%d, err:%s\r\n", txID, err)
			}
			err = batch.PrintObject(txRecord)
			if err != nil {
				fmt.Printf("failed to print the tx record, err:%s\r\n", err)
			}
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
	rootCmd.AddCommand(
		// txs
		startCmd,
		initCmd,
		prepareCmd,
		batchTestCmd,
		// queries
		QueryTestObjectsCmd,
		QueryHelperRecordCmd,
		QueryTxRecordCmd,
	)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
