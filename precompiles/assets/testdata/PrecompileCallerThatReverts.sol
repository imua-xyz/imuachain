// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "../IAssets.sol";
import "./ThirdPartyCallee.sol";

contract PrecompileCallerThatReverts {
    // Constants for the test chain
    uint8 public constant TEST_CHAIN_ID = 99;
    string public constant TEST_CHAIN_NAME = "TestChain";
    string public constant TEST_CHAIN_METADATA = "Test metadata";
    string public constant TEST_SIGNATURE_SCHEME = "TestSignature";
    uint8 public constant STAKER_ACCOUNT_LENGTH = 20;
    
    // Constants for the test token
    address public constant VIRTUAL_TOKEN_ADDRESS = 0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB;
    bytes public constant VIRTUAL_TOKEN = abi.encodePacked(VIRTUAL_TOKEN_ADDRESS);
    uint8 public constant TEST_TOKEN_DECIMALS = 8;
    string public constant TEST_TOKEN_NAME = "TestToken";
    string public constant TEST_TOKEN_METADATA = "Test token metadata";
    string public constant TEST_TOKEN_ORACLE_INFO = "TestChain,TestToken,8";
    bytes4 constant SELECTOR = this.callPrecompile.selector;

    // local state
    ThirdPartyCallee anotherReverter; // 0
    uint256 public nonce; // 1
    uint256 public successCount; // 2
    uint256 public failureCount; // 3

    constructor(address anotherReverter_) {
        anotherReverter = ThirdPartyCallee(anotherReverter_);
    }

    // Register client chain and token (similar to UTXOGateway.activateStakingForClientChain)
    function activateStakingForTestChain() external {
        _registerOrUpdateClientChain(
            TEST_CHAIN_ID, 
            STAKER_ACCOUNT_LENGTH, 
            TEST_CHAIN_NAME, 
            TEST_CHAIN_METADATA, 
            TEST_SIGNATURE_SCHEME
        );
        _registerOrUpdateToken(
            TEST_CHAIN_ID, 
            VIRTUAL_TOKEN, 
            TEST_TOKEN_DECIMALS, 
            TEST_TOKEN_NAME, 
            TEST_TOKEN_METADATA, 
            TEST_TOKEN_ORACLE_INFO
        );
    }

    function callPrecompileAndRevert(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external {
        nonce += 1;
        callPrecompile(clientChainID, token, staker, amount);
        anotherReverter.callMe{value: 1 ether}();
    }

    function callPrecompileAndNotRevert(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external {
        nonce += 1;
        callPrecompile(clientChainID, token, staker, amount);
    }

    function callPrecompileAndRevert2(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external {
        nonce += 1;
        callPrecompile2(clientChainID, token, staker, amount);
        anotherReverter.callMe{value: 1 ether}();
    }

    function callPrecompileAndNotRevert2(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external {
        nonce += 1;
        callPrecompile2(clientChainID, token, staker, amount);
    }

    function callPrecompile(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) public returns (bool) {
        // Call the precompile to deposit
        (bool success,) = ASSETS_CONTRACT.depositLST(
            clientChainID,
            token,
            staker,
            amount
        );

        return success;
    }

    function callPrecompileGasStarved(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount,
        uint256 gasLimit
    ) public returns (bool success) {
        nonce += 1;
        bytes memory data = abi.encodeWithSelector(
            ASSETS_CONTRACT.depositLST.selector,
            clientChainID,
            token,
            staker,
            amount
        );
        address target = address(ASSETS_CONTRACT);
        assembly {
            let ptr := add(data, 0x20)
            success := call(
                gasLimit, target, 0, add(data, 0x20), mload(data), ptr, 0x20
            )
            success := and(success, mload(ptr))
        }
        if (success) {
            successCount += 1;
        } else {
            failureCount += 1;
        }
    }

    function callPrecompile2(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) public returns (bool) {
        // Call the precompile to deposit
        (bool success,) = ASSETS_CONTRACT.withdrawLST(
            clientChainID,
            token,
            staker,
            amount
        );

        return success;
    }

    // Helper function to register a client chain
    function _registerOrUpdateClientChain(
        uint8 clientChainId,
        uint8 stakerAccountLength,
        string memory name,
        string memory metadata,
        string memory signatureScheme
    ) internal {
        (bool success,) = ASSETS_CONTRACT.registerOrUpdateClientChain(
            uint32(clientChainId), 
            stakerAccountLength, 
            name, 
            metadata, 
            signatureScheme
        );
        
        require(success, "Failed to register client chain");
    }

    // Helper function to register a token
    function _registerOrUpdateToken(
        uint8 clientChainId,
        bytes memory token,
        uint8 decimals,
        string memory name,
        string memory metadata,
        string memory oracleInfo
    ) internal {
        uint32 clientChainIdUint32 = uint32(clientChainId);
        bool registered = ASSETS_CONTRACT.registerToken(
            clientChainIdUint32, 
            token, 
            decimals, 
            name, 
            metadata, 
            oracleInfo
        );
        
        if (!registered) {
            bool updated = ASSETS_CONTRACT.updateToken(clientChainIdUint32, token, metadata);
            require(updated, "Failed to update token");
        }
    }
}