// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract GatewayCallee {
    uint256 public success;

    function reverter(bool shouldRevert) public {
        if (shouldRevert) {
            revert("Intentional revert");
        }
        ++success;
    }
}