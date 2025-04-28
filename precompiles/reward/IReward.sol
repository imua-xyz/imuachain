// SPDX-License-Identifier: MIT
pragma solidity >=0.8.17;

/// @dev The reward contract's address.
address constant REWARD_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000806;

/// @dev The reward contract's instance.
IReward constant REWARD_CONTRACT = IReward(
    REWARD_PRECOMPILE_ADDRESS
);

/// @dev The RewardCoin struct. it's equal to the DecCoin in cosmos-SDK
/// @param symbol The symbol of the reward coin, it will be used as the denomination in cosmos-SDK
/// @param amount The amount of the reward coin, it needs to be converted to decimal when using in native module.
    struct RewardCoin {
        string symbol;
        uint256 amount;
    }

/// @dev The OperatorRewardProportion struct. it's equal to the OperatorRewardProportion in distribution.proto
/// @param operator The operator address.
/// @param amount The amount of the reward coin, it needs to be converted to decimal when using in native module.
    struct OperatorRewardProportion {
        string operator;
        uint256 numerator;
        uint256 denominator;
    }

/// @dev The AVSRewardDistributionInfo struct. it's equal to the AVSRewardDistribution in distribution.proto
/// @param operator The operator address.
/// @param amount The amount of the reward coin, it needs to be converted to decimal when using in native module.
    struct AVSRewardDistributionInfo {
        RewardCoin[] rewardCoins;
        OperatorRewardProportion[] operatorRewardProportions;
    }

/// @author Imuachain Team
/// @title Reward Precompile Contract
/// @dev The interface through which solidity contracts will interact with ClaimReward
/// @custom:address 0x0000000000000000000000000000000000000806
interface IReward {

    /// The following transaction or query interfaces are used to claim and withdraw rewards for stakers and operators.
    ///
    /// TRANSACTIONS
    /// @dev claim the rewards earned from all AVSs to the staker.
    /// @dev This updates the outstanding reward state of the specified staker.
    /// Since we use F1 distribution for reward allocation, rewards are distributed lazily
    /// only when the delegation changes. This interface allows the staker to actively claim
    /// accumulated rewards, which are then recorded in the outstanding rewards for future withdrawal.
    /// Note that this address cannot be a module account.
    /// @param clientChainLzID The LzID of the client chain the staker originates from
    /// @param stakerAddress The address of the staker claiming the reward.
    function claimReward(
        uint32 clientChainLzID,
        bytes calldata stakerAddress
    ) external returns (bool success);

    /// @dev Withdraw the rewards earned from multiple AVSs (excluding the dogfood AVS) to the staker.
    /// This will update the outstanding reward state of the specified staker.
    /// Note that this address cannot be a module account.
    /// @param clientChainLzID The LzID of the client chain the staker originates from
    /// @param rewardAssetChainLzID The LzID of the chain the reward asset originates from
    /// @param assetAddress The reward asset Address
    /// @param stakerAddress The address of the staker withdrawing the reward.
    /// @param opAmount The reward amount
    function withdrawReward(
        uint32 clientChainLzID,
        uint32 rewardAssetChainLzID,
        bytes calldata assetAddress,
        bytes calldata stakerAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 actualWithdrawAmount);

    /// @dev Withdraws rewards in IMUA tokens earned from the dogfood AVS or from other AVSs that also distribute IMUA as rewards.
    /// Unlike `withdrawReward`, if the IMUA reward are from the dogfood AVS, this function sends the IMUA tokens directly
    /// to the staker within the native module. In such cases, the staker must provide a receipt address in case their
    /// staker address is incompatible with the IMUA chain.
    /// When other AVSs choose the IMUA token as the reward, the rewards vault will be managed by the contract,
    /// similar to how other reward tokens are handled. In this case, the actual token transfer will not occur within the native module.
    /// If the IMUA rewards originate from multiple AVSs, including the dogfood AVS, this interface will return both
    /// `actualWithdrawAmount` and `withdrawAmountFromDogfood`. The difference between them indicates the amount
    /// that remains to be claimed from the contract vault.
    /// Note that this address cannot be a module account.
    /// @param clientChainLzID The LzID of the client chain the staker originates from
    /// @param stakerAddress The address of the staker withdrawing the reward.
    /// @param receiptAddress The address to receive the IMUA reward. It should be an EVM address.
    /// @param opAmount The reward amount
    function withdrawIMUATokenReward(
        uint32 clientChainLzID,
        bytes calldata stakerAddress,
        bytes calldata receiptAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 actualWithdrawAmount, uint256 withdrawAmountFromDogfood);

    /// @dev Withdraw the commissions earned from multiple AVSs (excluding the dogfood AVS) to the operator.
    /// This will update the commission state of the specified operator.
    /// Note that this address cannot be a module account.
    /// @param rewardAssetChainLzID The LzID of the chain the commission asset originates from
    /// @param assetAddress The commission asset Address
    /// @param operatorAddress The address of the operator withdrawing the commission.
    /// @param opAmount The commission amount
    function withdrawCommission(
        uint32 rewardAssetChainLzID,
        bytes calldata assetAddress,
        bytes calldata operatorAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 actualWithdrawAmount);

    /// @dev Withdraws commissions in IMUA tokens earned from the dogfood AVS or from other AVSs that also
    /// distribute IMUA as rewards. The detailed logic is similar to `withdrawIMUATokenReward`.
    /// Note that this address cannot be a module account.
    /// @param operatorAddress The address of the operator withdrawing the commission.
    /// @param receiptAddress The address to receive the IMUA reward. It can be same as the operator address
    /// The recipient and operator addresses should be of EVM address type
    /// @param opAmount The commission amount
    function withdrawIMUATokenCommission(
        bytes calldata operatorAddress,
        bytes calldata receiptAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 actualWithdrawAmount, uint256 withdrawAmountFromDogfood);

    /// The following transaction or query interfaces are used to set distribution information for AVSs.
    ///
    /// TRANSACTIONS
    /// @dev register a token as the reward asset for an AVS.
    /// @param clientChainID is the identifier of the token's home chain (LZ or otherwise)
    /// @param token is the address of the token on the home chain
    /// @param decimals is the number of decimals of the token
    /// @param name is the name of the token
    /// @param symbol is the symbol of the token, used as its denomination in the Cosmos SDK.
    /// @param metaData is the arbitrary metadata of the token
    /// @return success if the token registration is successful
    /// The AVS address will be fetched from the contract caller instead of the input parameters,
    /// since these interfaces are intended to be called directly by the AVS itself.
    /// This design ensures proper authorization, as only the AVS address is allowed
    /// to invoke these transaction interfaces.
    function registerRewardToken(
        uint32 clientChainID,
        bytes calldata token,
        uint8 decimals,
        string calldata name,
        string calldata symbol,
        string calldata metaData
    ) external returns (bool success);

    /// @dev update the metaInfo for the reward token.
    /// @param clientChainID is the identifier of the token's home chain (LZ or otherwise)
    /// @param token is the address of the token on the home chain
    /// @param metaData is the arbitrary metadata of the token
    /// @return success if the token update is successful
    /// @dev The token must previously be registered before updating
    function updateRewardToken(uint32 clientChainID, bytes calldata token, string calldata metaData)
    external returns (bool success);

    /// @dev set the reward distribution information for an AVS
    /// @param rewardDistribution The reward distribution information, including the rewards
    /// and operator proportions for each epoch. This information will be used for each epoch
    /// until it is updated. It will invoke the function SetAVSRewardDistribution in feedistribution
    /// module.
    function setAVSRewardDistribution(AVSRewardDistributionInfo calldata rewardDistribution)
    external returns (bool success);

    /// @dev sets the epoch rewards exclusively for an AVS.
    /// @param epochRewards The total rewards for each epoch. This information will be used for each epoch
    /// until it is updated. It will invoke the function SetAVSEpochRewardExclusive in feedistribution
    /// module.
    /// In the current implementation, epoch rewards must be configured even if `isCustomRewardInflation` is false,
    /// because the default reward inflation logic is not yet implemented. As a result, the default function reads
    /// reward information directly from the KV store.
    function setAVSEpochReward(RewardCoin[] calldata epochRewards)
    external returns (bool success);

    /// @dev sets the operator reward proportions exclusively for an AVS.
    /// @param operatorRewardProportions The operator reward proportions for each epoch. This information will
    /// be used for each epoch until it is updated. It will invoke the function SetAVSRewardProportionsExclusive in
    /// feedistribution module.
    function setOperatorRewardProportions(OperatorRewardProportion[] calldata operatorRewardProportions)
    external returns (bool success);

    /// @dev sets the reward parameters for an AVS.
    /// @param isCustomRewardInflation The flag to determine whether customizing the reward inflation.
    /// @param isCustomOperatorRatio The flag to determine whether customizing the operator reward proportions.
    /// It will invoke the function SetAVSRewardParam in feedistribution module.
    /// The distribution information set by the three interfaces above will only be used
    /// when the corresponding flags are enabled through this interface.
    function setAVSRewardParams(bool isCustomRewardInflation, bool isCustomOperatorRatio)
    external returns (bool success);

    /// @dev This function funds rewards for an AVS. Unlike the other interfaces for AVS mentioned above,
    /// it should be called by the gateway contract, because the verification of funding to the reward
    /// vault is handled by the system contracts. This interface only updates the state recorded in the native module.
    /// @param rewardAssetChainLzID The LzID of the chain the reward asset originates from
    /// @param avsAddress The avs address
    /// @param assetAddress The reward asset Address
    /// @param opAmount The reward amount to fund
    function fundAVSReward(
        uint32 rewardAssetChainLzID,
        address avsAddress,
        bytes calldata assetAddress,
        uint256 opAmount
    ) external returns (bool success);
}
