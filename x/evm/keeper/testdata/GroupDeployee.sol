// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

// This contract is used to verify that x/group can deploy a contract,
// and later call a function on it.
contract GroupDeployee {
    uint256 public value;
    address public owner;

    constructor() {
        value = 1;
        // simple case
        owner = msg.sender;
    }

    function setValue(uint256 _value) external {
        require(msg.sender == owner, "Only owner can set value");
        value = _value;
    }

    function failingFunction() external {
        // to avoid marking it as pure, add a state change op.
        value += 1;
        revert("This function is failing");
    }

    function setValueWithAmount(uint256 _value) external payable {
        require(msg.sender == owner, "Only owner can set value");
        value = _value;
    }

}