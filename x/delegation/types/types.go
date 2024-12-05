package types

type DelegationOpFunc func(keys *SingleDelegationInfoReq, amounts *DelegationAmounts) (bool, error)
