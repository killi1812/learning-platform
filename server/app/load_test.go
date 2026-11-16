package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSecrets(t *testing.T) {
	validKey1 := "12345678901234567890123456789012"
	validKey2 := "abcdefghijklmnopqrstuvwxyzabcdef"

	t.Run("Valid distinct 32-char keys pass", func(t *testing.T) {
		err := ValidateSecrets(validKey1, validKey2)
		assert.NoError(t, err)
	})

	t.Run("Empty access key fails", func(t *testing.T) {
		err := ValidateSecrets("", validKey2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ACCESS_KEY")
	})

	t.Run("Empty refresh key fails", func(t *testing.T) {
		err := ValidateSecrets(validKey1, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "REFRESH_KEY")
	})

	t.Run("Default placeholder values fail", func(t *testing.T) {
		err := ValidateSecrets("your-access-key-here", validKey2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder")

		err2 := ValidateSecrets(validKey1, "your-refresh-key-here")
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "placeholder")
	})

	t.Run("Short access key fails", func(t *testing.T) {
		err := ValidateSecrets("short-key", validKey2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 32 characters")
	})

	t.Run("Short refresh key fails", func(t *testing.T) {
		err := ValidateSecrets(validKey1, "short-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 32 characters")
	})

	t.Run("Identical access and refresh keys fail", func(t *testing.T) {
		err := ValidateSecrets(validKey1, validKey1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be different")
	})
}

func TestLoadBool(t *testing.T) {
	t.Run("unset defaults to false", func(t *testing.T) {
		t.Setenv("TEST_LOAD_BOOL", "")
		assert.False(t, loadBool("TEST_LOAD_BOOL"))
	})

	for _, truthy := range []string{"true", "True", "TRUE", "1", " true "} {
		t.Run("parses_"+truthy, func(t *testing.T) {
			t.Setenv("TEST_LOAD_BOOL", truthy)
			assert.True(t, loadBool("TEST_LOAD_BOOL"))
		})
	}

	for _, falsy := range []string{"false", "FALSE", "0", "not-a-bool"} {
		t.Run("falsy_"+falsy, func(t *testing.T) {
			t.Setenv("TEST_LOAD_BOOL", falsy)
			assert.False(t, loadBool("TEST_LOAD_BOOL"))
		})
	}
}
