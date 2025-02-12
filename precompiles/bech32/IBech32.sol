// SPDX-License-Identifier: MIT
pragma solidity >=0.8.17;

/// @dev The IBech32 contract's address.
address constant BECH32_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000400;

IBech32 constant BECH32_CONTRACT = IBech32(BECH32_PRECOMPILE_ADDRESS);

/// @author ExocoreNetwork
/// @title Bech32 Precompiled Contract
/// @dev This contract can be used by Solidity devs to convert from `string bech32Addr` to
///      `address 0xAddr` and vice versa. The bech32-prefix used is the chain's prefix, via
///      `sdk.Config#SetBech32PrefixForAccount`.
/// @custom:address 0x0000000000000000000000000000000000000400
interface IBech32 {

    /// @dev Defines a method for converting a hex formatted address to bech32.
    /// @param addr The hex address to be converted.
    /// @param prefix The human readable prefix (HRP) of the bech32 address.
    /// @return bech32Address The address in bech32 format.
    function hexToBech32(address addr, string memory prefix) external view returns (string memory bech32Address);

    /// @dev Defines a method for converting a bech32 formatted address to hex.
    /// @param bech32Address The bech32 address to be converted.
    /// @return addr The address in hex format.
    function bech32ToHex(string memory bech32Address) external view returns (address addr);

}
