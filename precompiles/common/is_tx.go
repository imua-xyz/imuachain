package common

import (
	"bytes"
	"embed"
	"fmt"
	"sort"

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

	abiDerived, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return fmt.Errorf("error parsing the ABI %s", err)
	}

	// sort for determinism
	// although this method is only used during `init()` currently,
	// this is a form of future-proofing.
	methods := make([]string, 0, len(abiDerived.Methods))
	for _, method := range abiDerived.Methods {
		// `RunSetup` does not discriminate based on the method type
		// So we should not do so either.
		methods = append(methods, method.Name)
	}
	sort.Strings(methods)

	for _, method := range methods {
		var localErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					localErr = fmt.Errorf(
						"panic occurred while checking method %s: %v",
						method, r,
					)
				}
			}()
			// the result is not relevant, we are only checking that the
			// function does not panic for each known method.
			_ = isTx(method)
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
		// the result is not relevant, we are only checking that the
		// function panics for an unknown method.
		_ = isTx("unknownMethod")
	}()

	return err
}
