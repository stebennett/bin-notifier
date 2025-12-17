package config

import (
	"errors"
	"testing"

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
)

func TestIsErrHelp_WithHelpError(t *testing.T) {
	helpErr := &flags.Error{
		Type: flags.ErrHelp,
	}

	result := isErrHelp(helpErr)

	assert.True(t, result)
}

func TestIsErrHelp_WithOtherFlagsError(t *testing.T) {
	tests := []struct {
		name    string
		errType flags.ErrorType
	}{
		{
			name:    "required flag error",
			errType: flags.ErrRequired,
		},
		{
			name:    "unknown flag error",
			errType: flags.ErrUnknownFlag,
		},
		{
			name:    "invalid choice error",
			errType: flags.ErrInvalidChoice,
		},
		{
			name:    "marshal error",
			errType: flags.ErrMarshal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flagsErr := &flags.Error{
				Type: test.errType,
			}

			result := isErrHelp(flagsErr)

			assert.False(t, result)
		})
	}
}

func TestIsErrHelp_WithStandardError(t *testing.T) {
	standardErr := errors.New("some standard error")

	result := isErrHelp(standardErr)

	assert.False(t, result)
}

func TestIsErrHelp_WithNilError(t *testing.T) {
	result := isErrHelp(nil)

	assert.False(t, result)
}
