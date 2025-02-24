pragma solidity >=0.8.17 .0;

/// @dev The SLASH contract's address.
address constant SLASH_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000807;

/// @dev The SLASH contract's instance.
ISlash constant SLASH_CONTRACT = ISlash(
    SLASH_PRECOMPILE_ADDRESS
);

/// @author Imuachain Team
/// @title Slash Precompile Contract
/// @dev The interface through which solidity contracts will interact with Slash
/// @custom:address 0x0000000000000000000000000000000000000807
interface ISlash {
/// TRANSACTIONS
/// @dev Slash the oprator, that will change the state in Slash module
/// Note that this address cannot be a module account.
/// @param clientChainLzID The lzId of client chain
/// @param assetsAddress The client chain asset Address
/// @param opAmount The Slash amount
/// @param operatorAddress The Slashed OperatorAddress
/// @param middlewareContractAddress The middleware address
/// @param proportion The Slash proportion
/// @param proof The Slash proof

    function submitSlash(
        uint32 clientChainLzID,
        bytes memory assetsAddress,
        bytes memory stakerAddress,
        uint256 opAmount,
        bytes memory operatorAddress,
        bytes memory middlewareContractAddress,
        string memory proportion,
        string memory proof
    ) external returns (bool success);
}
