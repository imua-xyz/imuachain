package common

const (
	ErrContractInputParamOrType = "the contract input parameter type or value error,arg index:%d, expected type is:%s,value:%v"

	ErrContractCaller = "the caller doesn't have the permission to call this function"

	ErrInvalidAddrLength = "invalid length of staker or asset addr, actualLength:%d,min:%d"

	ErrInputOperatorAddrLength = "mismatched length of the input operator address,actual is:%d,expected:%v"

	ErrInvalidInputList = "the length of input list is invalid, field:%s, actualLength:%d, expected:%v"

	ErrInvalidMetaInfoLength = "nil meta info or too long for chain or token,value:%s,actualLength:%d,max:%d"

	ErrInvalidNameLength = "nil name or too long for chain or token,value:%s,actualLength:%d,max:%d"

	ErrInvalidEVMAddr = "the address is an invalid EVM address, addr:%s"

	ErrInvalidOracleInfo = "oracle info is invalid, need at least three fields not empty: token.Name, Chain.Name, token.Decimal"

	ErrEmptyGateways = "the gateways is empty"

	ErrIndexOutOfRange = "index out of range, index:%d, length:%d"
)
