# how to write a test to call the precompile contract from a contract

## cmd to generate abi and bin

`solc --base-path ./ --include-path ./../.. --evm-version paris --combined-json abi,bin ./DepositCaller.sol > /tmp/DepositCaller.combined.json`

Then convert the combined output to `DepositCaller.json` using this format:
- `abi` must be a JSON string
- `bin` must be a hex string

You can refer to `assets_integrate_test.go` for calling the Assets precompile from a contract.
