// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract ThirdPartyCallee {
    uint256 etherReceived;

    function callMe() external payable {
        if (msg.value == 0) {
            revert("No Zero Value");
        }
        etherReceived = msg.value;
    }
}