pragma solidity >=0.8.17;

/// @dev The delegation contract's address.
address constant DELEGATION_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000805;

/// @dev The delegation contract's instance.
IDelegation constant DELEGATION_CONTRACT = IDelegation(
    DELEGATION_PRECOMPILE_ADDRESS
);

/// @author Imuachain Team
/// @title delegation Precompile Contract
/// @dev The interface through which solidity contracts will interact with delegation
/// @custom:address 0x0000000000000000000000000000000000000805
interface IDelegation {
/// TRANSACTIONS
/// @dev delegate the client chain assets to the operator through client chain, that will change the states in delegation and assets module
/// Note that this address cannot be a module account.
/// @param clientChainID is the layerZero chainID if it is supported.
//  It might be allocated by Imuachain when the client chain isn't supported
//  by layerZero
/// @param lzNonce The cross chain tx layerZero nonce
/// @param assetsAddress The client chain asset Address
/// @param stakerAddress The staker address
/// @param operatorAddr  The operator address that wants to be delegated to
/// @param opAmount The delegation amount
    function delegate(
        uint32 clientChainID,
        uint64 lzNonce,
        bytes calldata assetsAddress,
        bytes calldata stakerAddress,
        bytes calldata operatorAddr,
        uint256 opAmount
    ) external returns (bool success);

/// @dev undelegate the client chain assets from the operator through client chain, that will change the states in delegation and assets module
/// Note that this address cannot be a module account.
/// @param clientChainID is the layerZero chainID if it is supported.
//  It might be allocated by Imuachain when the client chain isn't supported
//  by layerZero
/// @param lzNonce The cross chain tx layerZero nonce
/// @param assetsAddress The client chain asset Address
/// @param stakerAddress The staker address
/// @param operatorAddr  The operator address that wants to unDelegate from
/// @param opAmount The Undelegation amount
    function undelegate(
        uint32 clientChainID,
        uint64 lzNonce,
        bytes calldata assetsAddress,
        bytes calldata stakerAddress,
        bytes calldata operatorAddr,
        uint256 opAmount
    ) external returns (bool success);

/// @dev associate the staker as being owned by the specified operator
/// @param clientChainID is the layerZero chainID if it is supported.
//  It might be allocated by Imuachain when the client chain isn't supported
//  by layerZero
/// @param staker is the EVM address of the staker
/// @param operator is the address that is to be marked as the owner.
    function associateOperatorWithStaker(
        uint32 clientChainID,
        bytes calldata staker,
        bytes calldata operator
    ) external returns (bool success);

/// @dev dissociate the operator from staker
/// @param clientChainID is the layerZero chainID if it is supported.
//  It might be allocated by Imuachain when the client chain isn't supported
//  by layerZero
/// @param staker is the EVM address to remove the marking from.
    function dissociateOperatorFromStaker(
        uint32 clientChainID,
        bytes calldata staker
    ) external returns (bool success);
}

