// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "../IAssets.sol";

import "./GatewayCallee.sol";

contract Gateway {
    address public callee;
    uint256 public counter;

    constructor(address callee_) {
        callee = callee_;
        counter = 1;
    }

    // Deposit LST
    function depositLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata stakerAddress,
        uint256 opAmount
    ) public returns (bool success, uint256 latestAssetState) {
        // Call the precompile
        (success, latestAssetState) = ASSETS_CONTRACT.depositLST(
            clientChainID,
            assetsAddress,
            stakerAddress,
            opAmount
        );

        return (success, latestAssetState);
    }

    // Withdraw LST
    function withdrawLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) public returns (bool success, uint256 latestAssetState) {
        counter++;
        // Call the precompile
        (success, latestAssetState) = ASSETS_CONTRACT.withdrawLST(
            clientChainID,
            assetsAddress,
            withdrawAddress,
            opAmount
        );

        return (success, latestAssetState);
    }

    function withdrawLSTAndThenRevert(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) public returns (bool success, uint256 latestAssetState) {
        (success, latestAssetState) = withdrawLST(clientChainID, assetsAddress, withdrawAddress, opAmount);
        GatewayCallee(callee).callMe{value: 1 ether}();
    }

    // Query staker balance
    function getStakerBalance(
        uint32 clientChainID,
        bytes calldata stakerAddress,
        bytes calldata tokenID
    ) public view returns (bool success, StakerBalance memory stakerBalance) {
        return ASSETS_CONTRACT.getStakerBalanceByToken(
            clientChainID,
            stakerAddress,
            tokenID
        );
    }

     function callPrecompileAndRevert(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external {
        counter += 1;
        callPrecompile(clientChainID, token, staker, amount);
        GatewayCallee(callee).callMe{value: address(this).balance + 1}();
    }

    function callPrecompile(
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) public returns (bool) {
        (bool success,) = ASSETS_CONTRACT.withdrawLST(
            clientChainID,
            token,
            staker,
            amount
        );

        return success;
    }
}