// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "./Gateway.sol";
import "../IAssets.sol";

contract GatewayCaller {
    address public gateway;
    constructor(address _gateway) {
        gateway = _gateway;
    }

    function depositLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata stakerAddress,
        uint256 opAmount
    ) public returns (bool success, uint256 latestAssetState) {
        return Gateway(gateway).depositLST(clientChainID, assetsAddress, stakerAddress, opAmount);
    }

    function withdrawLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) public returns (bool success, uint256 latestAssetState) {
        return Gateway(gateway).withdrawLST(clientChainID, assetsAddress, withdrawAddress, opAmount);
    }

    function withdrawLSTAndThenRevert(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount,
        // see if we can exceed 7 and then what happens
        uint256 count
    ) public {
        for (uint256 i = 0; i < count; i++) {
            try Gateway(gateway).withdrawLSTAndThenRevert(clientChainID, assetsAddress, withdrawAddress, opAmount) {
                // do nothing
            } catch {
                // do nothing
            }
        }
    }

}