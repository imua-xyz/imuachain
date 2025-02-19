pragma solidity >=0.8.17 .0;

/// @dev The claimReward contract's address.
address constant CLAIM_REWARD_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000806;

/// @dev The claimReward contract's instance.
IClaimReward constant CLAIM_REWARD_CONTRACT = IClaimReward(
    CLAIM_REWARD_PRECOMPILE_ADDRESS
);

/// @author Imuachain Team
/// @title ClaimReward Precompile Contract
/// @dev The interface through which solidity contracts will interact with ClaimReward
/// @custom:address 0x0000000000000000000000000000000000000806
interface IClaimReward {
/// TRANSACTIONS
/// @dev ClaimReward To the staker, that will change the state in reward module
/// Note that this address cannot be a module account.
/// @param clientChainLzID The LzID of client chain
/// @param assetsAddress The client chain asset Address
/// @param withdrawRewardAddress The claim reward address
/// @param opAmount The reward amount
    function claimReward(
    uint32 clientChainLzID,
    bytes memory assetsAddress,
    bytes memory withdrawRewardAddress,
    uint256 opAmount
    ) external returns (bool success,uint256 latestAssetState);
}
