
rem exocore compile on windows
rem install golang , gcc, sed for windows
rem 1. install msys2 : https://www.msys2.org/
rem 2. pacman -S mingw-w64-x86_64-toolchain
rem    pacman -S sed
rem    pacman -S mingw-w64-x86_64-jq
rem 3. add path C:\msys64\mingw64\bin  
rem             C:\msys64\usr\bin

set KEY="dev0"
# TODO: exocore testnet chainid is still under consideration and need to be finalized later
set CHAINID="exocorelocalnet_232-1"
set MONIKER="localtestnet"
set KEYRING="test"
set ALGO="eth_secp256k1"
set LOGLEVEL="info"
# to trace evm
#TRACE="--trace"
set TRACE=""
set HOME=%USERPROFILE%\.exocored
echo %HOME%
set ETHCONFIG=%HOME%\config\config.toml
set GENESIS=%HOME%\config\genesis.json
set TMPGENESIS=%HOME%\config\tmp_genesis.json

@echo build binary
go build .\cmd\exocored


@echo clear home folder
del /s /q %HOME%

exocored config keyring-backend %KEYRING%
exocored config chain-id %CHAINID%

exocored keys add %KEY% --keyring-backend %KEYRING% --algo %ALGO%

rem Set moniker and chain-id for exocore (Moniker can be anything, chain-id must be an integer)
exocored init %MONIKER% --chain-id %CHAINID% 

rem Change parameter token denominations to hua
cat %GENESIS% | jq ".app_state[\"staking\"][\"params\"][\"bond_denom\"]=\"hua\""   >   %TMPGENESIS% && move %TMPGENESIS% %GENESIS%
cat %GENESIS% | jq ".app_state[\"crisis\"][\"constant_fee\"][\"denom\"]=\"hua\"" > %TMPGENESIS% && move %TMPGENESIS% %GENESIS%
cat %GENESIS% | jq ".app_state[\"gov\"][\"deposit_params\"][\"min_deposit\"][0][\"denom\"]=\"hua\"" > %TMPGENESIS% && move %TMPGENESIS% %GENESIS%
cat %GENESIS% | jq ".app_state[\"mint\"][\"params\"][\"mint_denom\"]=\"hua\"" > %TMPGENESIS% && move %TMPGENESIS% %GENESIS%

rem increase block time (?)
cat %GENESIS% | jq ".consensus_params[\"block\"][\"time_iota_ms\"]=\"30000\"" > %TMPGENESIS% && move %TMPGENESIS% %GENESIS%

rem gas limit in genesis
cat %GENESIS% | jq ".consensus_params[\"block\"][\"max_gas\"]=\"10000000\"" > %TMPGENESIS% && move %TMPGENESIS% %GENESIS%

rem setup
sed -i "s/create_empty_blocks = true/create_empty_blocks = false/g" %ETHCONFIG%

rem Allocate genesis accounts (cosmos formatted addresses)
exocored add-genesis-account %KEY% 100000000000000000000000000hua --keyring-backend %KEYRING%

rem Sign genesis transaction
exocored gentx %KEY% 1000000000000000000000hua --keyring-backend %KEYRING% --chain-id %CHAINID%

rem Collect genesis tx
exocored collect-gentxs

rem Run this to ensure everything worked and that the genesis file is setup correctly
exocored validate-genesis



rem Start the node (remove the --pruning=nothing flag if historical queries are not needed)
exocored start --pruning=nothing %TRACE% --log_level %LOGLEVEL% --minimum-gas-prices=0.0001hua