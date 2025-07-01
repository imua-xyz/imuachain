# Imua Protocol Deployed Contracts Documentation

## Introduction

The Imua Protocol contracts form the foundation of a multi-chain restaking ecosystem that works in conjunction with Imuachain.
These contracts enable users across different blockchains to participate in restaking without moving their assets
to Imuachain directly.

Imuachain serves as the central coordination layer for the protocol, while these client-side contracts handle asset
custody and cross-chain communication. Together, they create a unified system where:

1. Assets remain secured on their native chains through specialized custody contracts
2. Staking operations and delegation decisions are coordinated by Imuachain
3. Cross-chain messaging enables seamless communication between client chains and Imuachain

This document details the contract addresses, architecture, and workflows that make omni-chain restaking possible
through the integration between client-side contracts and Imuachain.

## Deployed Contract Addresses

### Sepolia

- **🟢 Bootstrap/ClientChainGateway Proxy: 0x64B5B5A618072C1E4D137f91Af780e3B17A81f3f 🟢**
- Bootstrap Logic: 0xaa8901063317BF66dfed363468b01A9FD08D1a79
- ClientChainGateway Logic: 0xF1d902b81aEc0E4944FE6675c32FD21dfBe365ae
- Vault Beacon: 0xa188242ec78894840E20979B6F5087FeB56a9f19
- Vault Implementation: 0x8cdCC93823cCBed482cf20b22A78305CF0404DC4
- Capsule Beacon: 0x1F91eD5F48E5Ad897E71Fa4559Be8fA1851F0696
- Capsule Implementation: 0xE71771EeC8ba37c8383FaAe6F4D0de7521813442
- RewardVault Beacon: 0xB058fa6626Fcc56e44A1137a4CE2EFd8BD7f7711
- RewardVault Implementation: 0xcb6986a0ebF721e9a26C6F9427E58082e87E86F7

### Holesky

- **🟢 Bootstrap/ClientChainGateway Proxy: 0x38674073a3713dd2C46892f1d2C5Dadc5Bb14172 🟢**
- Bootstrap Logic: 0xCdA750418AFE2d2aD0f1052E8e997a52C2AA6861
- ClientChainGateway Logic: 0x7944cd287F2Fcd03F4Cdc0B17801e29F1317c386
- Vault Beacon: 0x9a7bc99d90D5B8D8a47Bd6F8DBb5407fE3b51668
- Vault Implementation: 0x9A6735c19f293500C24886cC829870cD725a98b6
- Capsule Beacon: 0xafc8e60Ab69735C5097c47e5E3A7ED5E63ce5001
- Capsule Implementation: 0x954B9BC52C4394984390065830500006805Db9C9
- RewardVault Beacon: 0x1b4B4D6abE71a49e7bb13774F3e1cb721a21Af8B
- RewardVault Implementation: 0x1283b551058bC8303c04B578c9AbA4727714808b

### Imuachain

- Imuachain Gateway: 0xdDf5218Dbff297ADdF17fB7977E2469D774545ED
- Imuachain Gateway Logic: 0x71C95911E9a5D330f4D621842EC243EE1343292e
- UTXO Gateway: 0x7c61a871071F91933a11567C14e056982aca93b4

## Contract Architecture

### Upgrade Pattern

Most contracts in the Imua protocol are upgradeable, following a proxy pattern:

- **Proxy Contract**: Entry point that users interact with
- **Logic Contract**: Contains the implementation code
- **Beacon Contract**: For beacon proxy pattern used by Vaults and Capsules

The proxy delegates calls to the logic contract while maintaining its own storage, allowing for upgrades without losing state.

## Key Contract Functionalities

### Bootstrap Contract

The initial entry point before Imuachain is spawned:

- Accepts validator registrations with commission settings
- Allows LST deposits and delegations to validators
- Manages NST (native ETH) staking setup
- Stores all information needed for Imuachain genesis
- **Important**: After spawn time, Bootstrap is upgraded in-place to ClientChainGateway

### ClientChainGateway

The upgraded version of Bootstrap after Imuachain is spawned:

- Inherits all state from Bootstrap
- Adds cross-chain messaging capabilities via LayerZero
- Serves as the primary interface for ongoing staking operations
- Manages both LST and NST restaking

#### LST Restaking Functions

- **deposit**: Locks tokens in token-specific Vault and sends cross-chain message
- **depositThenDelegateTo**: Combines deposit and delegation in one transaction
- **delegateTo**: Delegates previously deposited tokens to an operator
- **undelegateFrom**: Undelegates tokens from an operator, initiating unbonding
- **claimPrincipalFromImuachain**: Initiates withdrawal request to Imuachain to get approval and unlock tokens
- **withdrawPrincipal**: Transfers unlocked tokens from Vault to recipient

#### NST Restaking Functions

- **createImuaCapsule**: Creates a Capsule contract for staker's ETH
- **stake**: Stakes 32 ETH to Ethereum beacon chain using Capsule as withdrawal credentials
- **verifyAndDepositNativeStake**: Verifies beacon chain proof and registers validator
- **processBeaconChainWithdrawal**: Processes beacon chain withdrawals with proofs
- **withdrawNonBeaconChainETHFromCapsule**: Withdraws ETH not related to beacon chain staking

### Vault

Each whitelisted LST has its own dedicated Vault:

- Tracks user balances and enforces TVL limits
- Handles token deposits and withdrawals
- Manages unlocking of principal
- Implemented as beacon proxies sharing the same implementation

### RewardVault

A single RewardVault per client chain handles all reward tokens:

- Tracks reward balances by token and AVS
- Handles deposit and withdrawal of rewards
- Manages unlocking of reward tokens

### ImuaCapsule

Contract for managing ETH staked on the beacon chain:

- Verifies deposit and withdrawal proofs
- Manages withdrawal credentials for validators
- Handles unlocking and withdrawing ETH

### ImuachainGateway

Deployed on Imuachain to handle cross-chain messaging:

- Registers client chains and their metadata
- Manages whitelisted tokens
- Processes messages from client chains

## Staking Workflows

### Before Imuachain Spawn (Bootstrap Phase)

1. **Validator Registration**:
   - Validators register with commission settings and consensus keys
   - Users deposit LSTs to Bootstrap contract
   - Users delegate tokens to registered validators
   - Native ETH stakers create Capsules and stake ETH

2. **Genesis Generation**:
   - After spawn time, Bootstrap is upgraded to ClientChainGateway
   - All registrations, deposits, and delegations form Imuachain genesis

### After Imuachain Spawn (ClientChainGateway Phase)

#### LST Restaking Workflow

1. **Deposit**: Call `deposit` or `depositThenDelegateTo` with token address, amount, and relay fee
2. **Delegate**: If using separate calls, call `delegateTo` with operator name, token, and amount
3. **Withdrawal**:
   - Call `claimPrincipalFromImuachain` with token and amount
   - Wait for cross-chain confirmation
   - Call `withdrawPrincipal` to receive tokens

#### NST Restaking Workflow

1. **Setup**:
   - Call `createImuaCapsule` to create withdrawal credentials
   - Call `stake` with validator details and 32 ETH
2. **Activation**:
   - Call `verifyAndDepositNativeStake` with validator proofs
3. **Withdrawal**:
   - Initiate withdrawal on beacon chain
   - Call `processBeaconChainWithdrawal` with proofs
   - ETH is unlocked in Capsule
   - Call `withdrawPrincipal` to receive ETH
   - Call `withdrawNonBeaconChainETHFromCapsule` to withdraw ETH received via direct transfer

### Reward Management (Support coming soon)

## Non-EVM Chain Support

Imua protocol is designed to support both EVM and non-EVM chains:

- **Bitcoin**: Staking support in testing phase
- **Solana**: Integration under testing phase
- **XRP**: Integration under testing phase

These non-EVM integrations will allow users to stake assets from multiple blockchain ecosystems through the Imua protocol,
expanding the cross-chain restaking capabilities.

## Important Notes

- **Same Address, Different Functionality**: Bootstrap and ClientChainGateway share the same proxy address but switch
  functionality at spawn time
- **Upgradeable Architecture**: All core contracts use proxy patterns for future upgrades
- **Token-Specific Vaults**: Each LST has its own dedicated Vault contract
- **Single Reward Vault**: All reward tokens share a single RewardVault per client chain
- **Cross-Chain Operations**: All operations after spawn time require LayerZero fees in ETH
- **TVL Limits**: Each token vault has configurable TVL limits for risk management

## References

- https://github.com/imua-xyz/imua-contracts/blob/main/script/deployments/deployedContracts.json
