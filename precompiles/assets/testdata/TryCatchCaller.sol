// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "./PrecompileCallerThatReverts.sol";

// Contract that uses try/catch to call the reverting contract
contract TryCatchCaller {
    // slot 0
    uint256 public successCount;
    // slot 1
    uint256 public errorCount;
    // slot 2
    uint256 public lowLevelRevertCount;

    function callWithTryCatch(
        address reverterContract,
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external returns (bool callSucceeded, string memory errorMessage) {
        try PrecompileCallerThatReverts(reverterContract).callPrecompileAndRevert(
            clientChainID,
            token,
            staker,
            amount
        ) {
            // This will never execute since the called function always reverts
            successCount++;
            return (true, "");
        } catch Error(string memory reason) {
            // Catch the revert but let the transaction complete successfully
            errorCount++;
            return (false, reason);
        } catch (bytes memory) {
            // Catch any other type of revert
            lowLevelRevertCount++;
            return (false, "Low-level revert");
        }
    }

    function callWithTryCatch2(
        address reverterContract,
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external returns (bool callSucceeded, string memory errorMessage) {
        try PrecompileCallerThatReverts(reverterContract).callPrecompileAndRevert2(
            clientChainID,
            token,
            staker,
            amount
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

    function callWithTryCatch3(
        address reverterContract,
        uint32 clientChainID,
        bytes calldata token,
        bytes calldata staker,
        uint256 amount
    ) external returns (bool callSucceeded, string memory errorMessage) {
        try PrecompileCallerThatReverts(reverterContract).callPrecompileAndNotRevert2(
            clientChainID,
            token,
            staker,
            amount
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
}