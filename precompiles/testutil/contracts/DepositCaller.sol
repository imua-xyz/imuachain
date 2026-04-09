
pragma solidity >=0.8.17;

import "assets/IAssets.sol";

contract DepositCaller {

    event CallDepositLSTResult(bool indexed success, uint256 indexed latestAssetState);
    event ErrorOccurred(string errorMessage);

    function testDepositLST(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        return
            ASSETS_CONTRACT.depositLST(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        );
    }

    function testCallDepositLSTAndEmitEvent(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        (bool success,uint256 latestAssetState) = ASSETS_CONTRACT.depositLST(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        );

        emit CallDepositLSTResult(success, latestAssetState);
        return (success, latestAssetState);
    }

    function testCallDepositLSTWithTryCatch(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount
    ) public returns (bool, uint256) {
        try ASSETS_CONTRACT.depositLST(
            clientChainLzID,
            assetsAddress,
            stakerAddress,
            opAmount
        ) returns (bool success, uint256 latestAssetState){
            //call successfully
            emit CallDepositLSTResult(success, latestAssetState);
            return (success, latestAssetState);
        }catch Error(string memory errorMessage){
            // An error occurred, handle it
            emit ErrorOccurred(errorMessage);
        }
        return (false,0);
    }
}
