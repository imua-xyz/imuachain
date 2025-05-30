// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract GatewayCallee {
    uint256 public etherReceived;

    function callMe() external payable {
        if (msg.value == 0) {
            revert("No Zero Value");
        }
        etherReceived = msg.value;
    }
}