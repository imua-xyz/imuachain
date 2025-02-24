package types_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

const (
	invalidEpochIDByType = uint64(0)
	invalidEpochIDByName = " "
	validEpochID         = " hello "
	nullEpochIDByName    = types.NullEpochIdentifier
)

var (
	validEpoch            = types.NewEpoch(1, validEpochID)
	invalidEpochInterface = struct{}{}
	invalidEpochString    = types.NewEpoch(1, invalidEpochIDByName)
	invalidEpochNumber    = types.NewEpoch(0, validEpochID)
)

func TestValidateEpochIdentifierInterface(t *testing.T) {
	t.Run("invalid epoch identifier by type", func(t *testing.T) {
		err := types.ValidateEpochIdentifierInterface(invalidEpochIDByType)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid epoch identifier by name", func(t *testing.T) {
		err := types.ValidateEpochIdentifierInterface(invalidEpochIDByName)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("null epoch identifier", func(t *testing.T) {
		err := types.ValidateEpochIdentifierInterface(nullEpochIDByName)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid epoch identifier", func(t *testing.T) {
		err := types.ValidateEpochIdentifierInterface(validEpochID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateEpochIdentifierString(t *testing.T) {
	t.Run("invalid epoch identifier by name", func(t *testing.T) {
		err := types.ValidateEpochIdentifierString(invalidEpochIDByName)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("null epoch identifier", func(t *testing.T) {
		err := types.ValidateEpochIdentifierString(nullEpochIDByName)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid epoch identifier", func(t *testing.T) {
		err := types.ValidateEpochIdentifierString(validEpochID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateEpochInterface(t *testing.T) {
	t.Run("invalid epoch by type", func(t *testing.T) {
		err := types.ValidateEpochInterface(invalidEpochInterface)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid epoch by string", func(t *testing.T) {
		err := types.ValidateEpochInterface(invalidEpochString)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid epoch by number", func(t *testing.T) {
		err := types.ValidateEpochInterface(invalidEpochNumber)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid epoch", func(t *testing.T) {
		err := types.ValidateEpochInterface(validEpoch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateEpoch(t *testing.T) {
	t.Run("invalid epoch by string", func(t *testing.T) {
		err := types.ValidateEpoch(invalidEpochString)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid epoch by number", func(t *testing.T) {
		err := types.ValidateEpoch(invalidEpochNumber)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid epoch", func(t *testing.T) {
		err := types.ValidateEpoch(validEpoch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
