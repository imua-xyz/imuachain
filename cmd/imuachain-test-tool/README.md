# imuachain-test-tool

This is a custom tool designed to batch-send test transactions to the Imuachain network. It can be used for stress testing
or routine automated testing of the Imuachain network.

Currently, all test transactions are executed by directly calling precompiles and are signed using automatically
generated private keys. Therefore, a customized Imuachain node, which disables the caller address check on precompiles,
is required for use. The branch of customized Imuachain is as below:
https://github.com/imua-xyz/imuachain/tree/pressure-test

When using the test tool to batch-send test transactions, you can dynamically adjust the number of test objects and the
transaction sending rate in the configuration file to control the test volume. This allows for routine automated testing
and stress testing of the Imuachain network.

## functionalities

The current implementation primarily provides the following functionalities:

1. Creates the required test objects, including assets, stakers, operators, and AVS, based on parameters specified in
   the configuration file, and saves them to the local SQLite database.
2. Transfers a specified amount of IMUA tokens to the created test objects, the amount has been defined in the
   configuration file, which will be used to sign subsequent test transactions.
3. Registers the created test objects onto the Imuachain network.
4. Saves the existing assets and AVS (dogfood) information on the Imuachain network to the local SQLite database.
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

The specific processes for the two types of testing are as follows:
A local node needs to be started through `local_node.sh` before testing.

1. automated testing:
    Step 1: Initialize the configuration file in the current directory

    ```shell
    imuachain-test-tool init --home
    ```

    Step 2: Start the test tool

    ```shell
    imuachain-test-tool start --home .
    ```

2. manual testing:

    Step 1: Initialize the configuration file in the current directory

    ```shell
    imuachain-test-tool init --home
    ```

    Step 2: Prepare the test environment

    ```shell
     imuachain-test-tool prepare --home .
    ```

    Step 3: Run batch tests

    ```shell
    imuachain-test-tool batch-test depositLST --home .
    ```

We can query transactions for a specific batch and status using the following command, either during or after the test:

```shell
imuachain-test-tool query-tx-record <msgType> <batch-id> <status>
```

## go binding generation

In the implementation of the test tool, since it needs to directly call precompiled contracts, the Go binding file for
the contract will be used. This binding file is automatically generated using abigen. For example, the Go binding for
the asset precompile can be generated using the following command:

```shell
abigen --abi abi.json --pkg assets --out assets_binding.go  
```

We need to regenerate the Go bindings using the above command when the ABI has been changed.

## todo

* Feed prices for all test assets. Currently, the test does not enable the oracle. We might consider providing a fake
  price or enabling price feeding for the test assets to evaluate the distribution function.
* Provide batch testing for IMUA tokens.
