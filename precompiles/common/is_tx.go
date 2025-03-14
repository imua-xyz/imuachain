package common

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ValidateIsTx loads the ABI from the given embed.FS and checks that the given
// isTx function
// (1) does not panic for known methods, and
// (2) panics for unknown methods.
// Ideally, it should be called in the init() for each precompile.
func ValidateIsTx(fs embed.FS, isTx func(methodName string) bool) error {
	abiBz, err := fs.ReadFile("abi.json")
	if err != nil {
		return fmt.Errorf("error loading the ABI %s", err)
	}

	abi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return fmt.Errorf("error parsing the ABI %s", err)
	}

	for _, method := range abi.Methods {
		var localErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					localErr = fmt.Errorf(
						"panic occurred while checking method %s: %v",
						method.Name, r,
					)
				}
			}()
			_ = isTx(method.Name)
		}()
		if localErr != nil {
			return localErr
		}
	}

	// lastly, check that unknown methods _do_ panic.
	err = fmt.Errorf("IsTx did not panic for unknown method")
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = nil
			}
		}()
		_ = isTx("unknownMethod")
	}()

	return err
}
