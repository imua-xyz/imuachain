package types

import (
	"bytes"
)

// RemoveFromBytesList removes an address from a list of addresses
// or a byte slice from a list of byte slices.
func RemoveFromBytesList(list [][]byte, addr []byte) [][]byte {
	for i, a := range list {
		if bytes.Equal(a, addr) {
			return append(list[:i], list[i+1:]...)
		}
	}
	panic("address not found in list")
}
