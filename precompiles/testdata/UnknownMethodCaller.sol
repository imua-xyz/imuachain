// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract UnknownMethodCaller {
    event UnknownMethodResult(bool success);

    function callUnknownMethod(address target) public {
        (bool success, ) = target.call(abi.encodeWithSelector(bytes4(keccak256("unknownMethod"))));
        emit UnknownMethodResult(success);
    }
}
