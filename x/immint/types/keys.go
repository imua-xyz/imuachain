package types

const (
	// ModuleName defines the module name
	ModuleName = "immint"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_immint"
)

const (
	// bytePrefixParams is the single byte prefix for the params store.
	bytePrefixParams byte = iota + 1
)

// KeyPrefixParams is the prefix for the params store, as a byte array.
func KeyPrefixParams() []byte {
	return []byte{bytePrefixParams}
}
