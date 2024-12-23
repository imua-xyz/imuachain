// SPDX-License-Identifier: MIT
pragma solidity >=0.8.17;

/// @dev The Assets contract's address.
address constant ASSETS_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000804;

/// @dev The Assets contract's instance.
IAssets constant ASSETS_CONTRACT = IAssets(ASSETS_PRECOMPILE_ADDRESS);

/// @dev The TokenInfo struct.
/// @param name The name of the token
/// @param symbol The symbol of the token
/// @param clientChainID The client chain ID
/// @param tokenID The token ID, typically the token address encoded in bytes
/// @param decimals The number of decimals of the token
/// @param totalStaked The total staked amount of the token
struct TokenInfo {
    string name;
    string symbol;
    uint32 clientChainID;
    bytes tokenID;
    uint8 decimals;
    uint256 totalStaked;
}

/// @dev The StakerBalance struct.
/// @param clientChainID The client chain ID
/// @param stakerAddress The staker address, typically the staker's address encoded in bytes
/// @param tokenID The token ID, typically the token address encoded in bytes
/// @param balance The balance of the staker, balance = withdrawable + delegated + pendingUndelegated
/// @param withdrawable The withdrawable balance
/// @param delegated The delegated balance
/// @param pendingUndelegated The pending undelegated balance, during the unboding period and would become withdrawable after the unboding period
/// @param totalDeposited The total deposited balance
struct StakerBalance {
    uint32 clientChainID;
    bytes stakerAddress;
    bytes tokenID;
    uint256 balance;
    uint256 withdrawable;
    uint256 delegated;
    uint256 pendingUndelegated;
    uint256 totalDeposited;
}

/// @author Exocore Team
/// @title Assets Precompile Contract
/// @dev The interface through which solidity contracts will interact with assets module
/// @custom:address 0x0000000000000000000000000000000000000804
interface IAssets {

    /// TRANSACTIONS
    /// @dev deposit the client chain assets, only LSTs, for the staker,
    /// that will change the state in assets module
    /// Note that this address cannot be a module account.
    /// @param clientChainID is the layerZero chainID if it is supported.
    //  It might be allocated by Exocore when the client chain isn't supported
    //  by layerZero
    /// @param assetsAddress The client chain asset address
    /// @param stakerAddress The staker address
    /// @param opAmount The amount to deposit
    function depositLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata stakerAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 latestAssetState);

    /// @dev deposit the client chain assets, native staking tokens, for the staker,
    /// that will change the state in assets module
    /// Note that this address cannot be a module account.
    /// @param clientChainID is the layerZero chainID if it is supported.
    //  It might be allocated by Exocore when the client chain isn't supported
    //  by layerZero
    /// @param validatorID The validator's identifier (index or pubkey)
    /// @param stakerAddress The staker address
    /// @param opAmount The amount to deposit
    function depositNST(
        uint32 clientChainID,
        bytes calldata validatorID,
        bytes calldata stakerAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 latestAssetState);

    /// @dev withdraw LST To the staker, that will change the state in assets module
    /// Note that this address cannot be a module account.
    /// @param clientChainID is the layerZero chainID if it is supported.
    //  It might be allocated by Exocore when the client chain isn't supported
    //  by layerZero
    /// @param assetsAddress The client chain asset Address
    /// @param withdrawAddress The withdraw address
    /// @param opAmount The withdraw amount
    function withdrawLST(
        uint32 clientChainID,
        bytes calldata assetsAddress,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 latestAssetState);

    /// @dev withdraw NST To the staker, that will change the state in assets module
    /// Note that this address cannot be a module account.
    /// @param clientChainID is the layerZero chainID if it is supported.
    //  It might be allocated by Exocore when the client chain isn't supported
    //  by layerZero
    /// @param validatorID The validator's identifier (index or pubkey)
    /// @param withdrawAddress The withdraw address
    /// @param opAmount The withdraw amount
    function withdrawNST(
        uint32 clientChainID,
        bytes calldata validatorID,
        bytes calldata withdrawAddress,
        uint256 opAmount
    ) external returns (bool success, uint256 latestAssetState);

    /// @dev registers or updates a client chain to allow deposits / staking, etc.
    /// from that chain.
    /// @param clientChainID is the layerZero chainID if it is supported.
    //  It might be allocated by Exocore when the client chain isn't supported
    //  by layerZero
    function registerOrUpdateClientChain(
        uint32 clientChainID,
        uint8 addressLength,
        string calldata name,
        string calldata metaInfo,
        string calldata signatureType
    ) external returns (bool success, bool updated);

    /// @dev register a token to allow deposits / staking, etc.
    /// @dev note that there is no way to delete a token. If a token is to be removed,
    /// the TVL limit should be set to 0 on the client chain.
    /// @param clientChainId is the identifier of the token's home chain (LZ or otherwise)
    /// @param token is the address of the token on the home chain
    /// @param decimals is the number of decimals of the token
    /// @param name is the name of the token
    /// @param metaData is the arbitrary metadata of the token
    /// @param oracleInfo is the oracle information of the token
    /// @return success if the token registration is successful
    function registerToken(
        uint32 clientChainId,
        bytes calldata token,
        uint8 decimals,
        string calldata name,
        string calldata metaData,
        string calldata oracleInfo
    ) external returns (bool success);

    /// @dev update a token to allow deposits / staking, etc.
    /// @param clientChainId is the identifier of the token's home chain (LZ or otherwise)
    /// @param token is the address of the token on the home chain
    /// @param metaData is the arbitrary metadata of the token
    /// @return success if the token update is successful
    /// @dev The token must previously be registered before updating
    function updateToken(uint32 clientChainId, bytes calldata token, string calldata metaData)
        external
        returns (bool success);
    
    /// @dev update the authorized gateways, only the authorized gateways can call precompile functions
    /// @dev If it is the mainnet, only the authority can call this function
    /// @param gateways the authorized gateways
    /// @return success if the update is successful
    function updateAuthorizedGateways(address[] calldata gateways) external returns (bool success);

    /// QUERIES
    /// @dev Returns the chain indices of the client chains.
    function getClientChains() external view returns (bool, uint32[] memory);

    /// @dev Checks if the client chain is registered, given the chain ID.
    /// @param clientChainID is the layerZero chainID if it is supported.
    /// @return success true if the query is successful
    /// @return isRegistered true if the client chain is registered
    function isRegisteredClientChain(uint32 clientChainID) external view returns (bool success, bool isRegistered);

    /// @dev Checks if the gateway is authorized, given the gateway address.
    /// @param gateway is the address of the gateway
    /// @return success true if the query is successful
    /// @return isAuthorized true if the gateway is authorized
    function isAuthorizedGateway(address gateway) external view returns (bool success, bool isAuthorized);

    /// @dev Returns the asset info for a given asset ID.
    /// @param clientChainId is the ID of the client chain
    /// @param tokenId is the ID of the token, typically the token address
    /// @return success true if the query is successful
    /// @return assetInfo the asset info
    function getTokenInfo(uint32 clientChainId, bytes calldata tokenId) external view returns (bool success, TokenInfo memory assetInfo);

    /// @dev Returns the staker's balance for a given token.
    /// @param clientChainId is the ID of the client chain
    /// @param tokenId is the ID of the token, typically the token address
    /// @return success true if the query is successful
    /// @return stakerBalance the staker's balance
    function getStakerBalanceByToken(uint32 clientChainId, bytes calldata stakerAddress, bytes calldata tokenId) external view returns (bool success, StakerBalance memory stakerBalance);
}


