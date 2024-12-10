# test-tool

This is a custom tool designed to batch-send test transactions to the Exocore chain. It can be used for stress testing
or routine automated testing of the Exocore chain.

Currently, all test transactions are executed by directly calling precompiles and are signed using automatically
generated private keys. Therefore, a customized Exocore node is required for use, with the node configured to disable
the precompile's gateway contract address check.

When using the test tool to batch-send test transactions, you can dynamically adjust the number of test objects and the
transaction sending rate in the configuration file to control the test volume. This allows for routine automated testing
and stress testing of the Exocore chain.

## functionalities

The current implementation primarily provides the following functionalities:

1. Creates the required test objects, including assets, stakers, operators, and AVS, based on parameters specified in
   the configuration file, and saves them to the local SQLite database.
2. Transfers a specified amount of Exo tokens to the created test objects, the amount has been defined in the
   configuration file, which will be used to sign subsequent test transactions.
3. Registers the created test objects onto the Exocore chain.
4. Saves the existing assets and AVS (dogfood) information on the Exocore chain to the local SQLite database.
5. Adds the newly created test assets to the asset list supported by the dogfood AVS.
6. Opts in the newly created test operators into all test-created AVS and the dogfood AVS.
7. Uses the private keys of all created stakers to batch-send deposit test transactions. Each asset initiates a test
   deposit transaction. After sending, the tool verifies the on-chain status and asset state of all transactions
   according to the configuration file parameters. The relevant transaction information and verification results are
   saved to the local SQLite database.
8. Uses the private keys of all stakers to batch-send delegation transactions. Each staker delegates their deposited
   assets equally to all operators. After sending, the tool verifies the on-chain status and asset state of all
   transactions according to the configuration file parameters. The relevant transaction information and verification
   results are saved to the local SQLite database.
9. Uses the private keys of all stakers to batch-send undelegation transactions. Each staker undelegates all delegations
   performed in the previous step. After sending, the tool verifies the on-chain status and asset state of all
   transactions according to the configuration file parameters. The relevant transaction information and verification
   results are saved to the local SQLite database.
10. Uses the private keys of all created stakers to batch-send withdrawal test transactions. Each asset initiates a test
    withdrawal transaction. After sending, the tool verifies the on-chain status and asset state of all transactions
    according to the configuration file parameters. The relevant transaction information and verification results are
    saved to the local SQLite database.
11. After waiting for a specified period as defined in the configuration file, the tool repeatedly performs the tests
    described in steps 7, 8, 9, and 10 in a loop.

## commands

* The `init` command can be used to create a default configuration file.
* The `start` command provides all the testing functionalities mentioned above.
* The `query` commands are used to retrieve information about test objects and the status of test transactions.
* The `prepare` and `batch-test` commands break down the functionalities provided by the start command. The `prepare`
  command handles the creation, funding, registration, and opting-in of test objects, while the `batch-test` command
  allows for manual batch testing of different transaction types.

## todo

* Feed prices for all test assets. Currently, the test does not enable the oracle. We might consider providing a fake
  price or enabling price feeding for the test assets to evaluate the distribution function.
* Provide batch testing for Exo tokens.
