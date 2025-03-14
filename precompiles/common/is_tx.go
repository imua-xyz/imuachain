package common

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func ValidateIsTx(fs embed.FS, isTx func(methodID string) bool) error {
	abiBz, err := fs.ReadFile("abi.json")
	if err != nil {
		return fmt.Errorf("error loading the deposit ABI %s", err)
	}

	abi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return fmt.Errorf("error parsing the deposit ABI %s", err)
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

	return nil
}
