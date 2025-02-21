package types

const (
	// ModuleName defines the module name
	ModuleName = "imslash"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	prefixParams       = 1
	prefixOperatorInfo = 2
)

var (
	KeyPrefixParams = []byte{prefixParams}
	// KeyPrefixOperatorInfo key-value: operatorAddr->operatorInfo
	KeyPrefixOperatorInfo = []byte{prefixOperatorInfo}
	ParamsKey             = []byte("Params")
)
