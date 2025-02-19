package types

const (
	// ModuleName defines the module name
	// TODO: if the upgrade module is used, this should be changed to "imslash" through that.
	// If it is not used, change it right now. CC @TimmyExogenous.
	ModuleName = "exoslash"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_exoslash"
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
