package config

import (
	"errors"
	"os"
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

func TestConfig_EnvironmentVariableFallback(t *testing.T) {
	// Save original env vars and restore after test
	envVars := []string{
		"BN_POSTCODE",
		"BN_ADDRESS_CODE",
		"BN_REGULAR_COLLECTION_DAY",
		"BN_FROM_NUMBER",
		"BN_TO_NUMBER",
		"BN_DRY_RUN",
		"BN_TODAY_DATE",
	}
	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
	}
	defer func() {
		for _, env := range envVars {
			if originalValues[env] == "" {
				os.Unsetenv(env)
			} else {
				os.Setenv(env, originalValues[env])
			}
		}
	}()

	// Set environment variables
	os.Setenv("BN_POSTCODE", "RG12 1AB")
	os.Setenv("BN_ADDRESS_CODE", "123456")
	os.Setenv("BN_REGULAR_COLLECTION_DAY", "2")
	os.Setenv("BN_FROM_NUMBER", "+441234567890")
	os.Setenv("BN_TO_NUMBER", "+447123456789")
	os.Setenv("BN_DRY_RUN", "true")
	os.Setenv("BN_TODAY_DATE", "2024-01-15")

	// Parse config using env vars (no CLI args)
	var c Config
	parser := flags.NewParser(&c, flags.IgnoreUnknown)
	_, err := parser.ParseArgs([]string{})

	assert.NoError(t, err)
	assert.Equal(t, "RG12 1AB", c.PostCode)
	assert.Equal(t, "123456", c.AddressCode)
	assert.Equal(t, 2, c.RegularCollectionDay)
	assert.Equal(t, "+441234567890", c.FromNumber)
	assert.Equal(t, "+447123456789", c.ToNumber)
	assert.True(t, c.DryRun)
	assert.Equal(t, "2024-01-15", c.TodayDate)
}

func TestConfig_CLIFlagsTakePrecedenceOverEnvVars(t *testing.T) {
	// Save original env vars and restore after test
	envVars := []string{
		"BN_POSTCODE",
		"BN_ADDRESS_CODE",
		"BN_REGULAR_COLLECTION_DAY",
		"BN_FROM_NUMBER",
		"BN_TO_NUMBER",
	}
	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
	}
	defer func() {
		for _, env := range envVars {
			if originalValues[env] == "" {
				os.Unsetenv(env)
			} else {
				os.Setenv(env, originalValues[env])
			}
		}
	}()

	// Set all required env vars
	os.Setenv("BN_POSTCODE", "ENV_POSTCODE")
	os.Setenv("BN_ADDRESS_CODE", "123456")
	os.Setenv("BN_REGULAR_COLLECTION_DAY", "2")
	os.Setenv("BN_FROM_NUMBER", "+441234567890")
	os.Setenv("BN_TO_NUMBER", "+447123456789")

	// Parse with CLI flag overriding postcode
	var c Config
	parser := flags.NewParser(&c, flags.IgnoreUnknown)
	_, err := parser.ParseArgs([]string{"-p", "CLI_POSTCODE"})

	assert.NoError(t, err)
	assert.Equal(t, "CLI_POSTCODE", c.PostCode, "CLI flag should take precedence over env var")
	assert.Equal(t, "123456", c.AddressCode, "Env var should be used when no CLI flag provided")
}
