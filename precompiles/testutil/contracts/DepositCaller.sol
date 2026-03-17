
pragma solidity >=0.8.17;

import "deposit/deposit.sol" as deposit;

contract DepositCaller {

    event CallDepositToResult(bool indexed success, uint256 indexed latestAssetState);
    event ErrorOccurred(string errorMessage);

    function testDepositTo(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        return
            deposit.DEPOSIT_CONTRACT.depositTo(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        );
    }

    function testCallDepositToAndEmitEvent(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        (bool success,uint256 latestAssetState) = deposit.DEPOSIT_CONTRACT.depositTo(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        );

        emit callDepositToResult(success, latestAssetState);
        return (success, latestAssetState);
    }

    function testCallDepositToWithTryCatch(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        try deposit.DEPOSIT_CONTRACT.depositTo(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        ) returns (bool success, uint256 latestAssetState){
            //call successfully
            emit callDepositToResult(success, latestAssetState);
            return (success, latestAssetState);
        }catch Error(string memory errorMessage){
            // An error occurred, handle it
            emit ErrorOccurred(errorMessage);
        }
        return (false,0);
    }
}
