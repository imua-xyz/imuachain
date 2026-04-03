// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "../IAssets.sol";

/// @title OracleGateway
/// @notice Minimal gateway for oracle-bridge e2e tests.
/// @dev Accepts oracle module calls and executes deposit/withdraw payloads.
///      Emits OutboundResponse when a handler produces a response (e.g., withdraw).
contract OracleGateway {
    address public immutable oracleCaller;

    event OracleReceived(uint32 srcChainId, uint64 nonce, bytes message);
    event DepositLST(uint32 srcChainId, bytes staker, bytes token, uint256 amount, bool success);
    event WithdrawLST(uint32 srcChainId, bytes staker, bytes token, uint256 amount, bool success);
    /// @notice Emitted when a handler produces a response for outbound relay.
    event OutboundResponse(uint32 indexed dstChainId, uint64 indexed requestNonce, bytes payload);

    constructor(address oracleCaller_) {
        require(oracleCaller_ != address(0), "oracle caller is zero");
        oracleCaller = oracleCaller_;
    }

    function oracleReceive(uint32 srcChainId, uint64 nonce, bytes calldata message) external {
        require(msg.sender == oracleCaller, "unauthorized caller");
        emit OracleReceived(srcChainId, nonce, message);

        require(message.length >= 1, "empty message");
        uint8 action = uint8(message[0]);
        bytes calldata payload = message[1:];

        if (action == 2) {
            // DepositLST payload: staker(32) | amount(32) | token(32)
            require(payload.length >= 96, "invalid deposit payload");
            bytes calldata staker = payload[:32];
            uint256 amount = uint256(bytes32(payload[32:64]));
            bytes calldata token = payload[64:96];
            (bool success, ) = ASSETS_CONTRACT.depositLST(srcChainId, token, staker, amount);
            emit DepositLST(srcChainId, staker, token, amount, success);
            require(success, "deposit failed");
        } else if (action == 4) {
            // WithdrawLST payload: staker(32) | amount(32) | token(32)
            // Returns response: Action.RESPOND(0) | nonce(8) | success(1)
            require(payload.length >= 96, "invalid withdraw payload");
            bytes calldata staker = payload[:32];
            uint256 amount = uint256(bytes32(payload[32:64]));
            bytes calldata token = payload[64:96];
            (bool success, ) = ASSETS_CONTRACT.withdrawLST(srcChainId, token, staker, amount);
            emit WithdrawLST(srcChainId, staker, token, amount, success);
            // Emit outbound response for oracle bridge to pick up.
            bytes memory responsePayload = abi.encodePacked(
                uint8(0),          // Action.RESPOND
                uint64(nonce),     // requestId = inbound nonce
                success ? uint8(1) : uint8(0)
            );
            emit OutboundResponse(srcChainId, nonce, responsePayload);
        } else {
            revert("unsupported action");
        }
    }
}
