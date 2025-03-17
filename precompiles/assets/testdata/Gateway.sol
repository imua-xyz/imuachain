// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "../IAssets.sol";

import "./GatewayCallee.sol";

contract Gateway {
    event UnknownMethodResult(bool success);

    address public callee;

    constructor(address callee_) {
        callee = callee_;
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
        GatewayCallee(callee).reverter(true);
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

    function callUnknownMethod() public {
        address assetsPrecompile = address(ASSETS_CONTRACT);
        (bool success, ) = assetsPrecompile.call(abi.encodeWithSelector(bytes4(keccak256("unknownMethod"))));
        emit UnknownMethodResult(success);
    }
}