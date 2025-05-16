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

    function withdrawLSTXTimes(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount,
        uint256 count
    ) public {
        for (uint256 i = 0; i < count; i++) {
            Gateway(gateway).withdrawLST(clientChainID, assetsAddress, withdrawAddress, opAmount);
        }
    }

    function withdrawLSTXTimesInTryCatch(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount,
        uint256 count
    ) public {
        for (uint256 i = 0; i < count; i++) {
            try Gateway(gateway).withdrawLST(clientChainID, assetsAddress, withdrawAddress, opAmount) {
                // do nothing
            } catch {
                // do nothing
            }
        }
    }

    function withdrawLSTAndThenRevertXTimes(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount,
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

    function callWithTryCatch(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) external returns (bool callSucceeded, string memory errorMessage) {
        try Gateway(gateway).callPrecompileAndRevert(
            clientChainID,
            assetsAddress,
            withdrawAddress,
            opAmount
        ) {
            // This will never execute since the called function always reverts
            return (true, "");
        } catch Error(string memory reason) {
            // Catch the revert but let the transaction complete successfully
            return (false, reason);
        } catch (bytes memory) {
            // Catch any other type of revert
            return (false, "Low-level revert");
        }
    }

    function callPrecompileWithDataInsideTryCatch(
        bytes memory data
    ) public payable {
        try Gateway(gateway).callPrecompileWithData{value: msg.value}(data) {
            // do nothing
        } catch {
            // do nothing
        }
    }

}