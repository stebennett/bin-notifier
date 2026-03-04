# Env Var Phone Numbers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow `from_number` and `to_number` to be set via `BN_FROM_NUMBER` and `BN_TO_NUMBER` environment variables, with env vars taking precedence over YAML config values.

**Architecture:** Add env var override logic in `LoadConfig()` between YAML unmarshal and validation. No changes to `validate()`, `main()`, or CLI flags.

**Tech Stack:** Go stdlib (`os.Getenv`), testify, existing config package patterns.

---

### Task 1: Add tests for env var overrides of phone numbers

**Files:**
- Modify: `pkg/config/config_test.go`

**Step 1: Write the failing tests**

Add three test functions at the end of `pkg/config/config_test.go`:

```go
func TestLoadConfig_FromNumberFromEnv(t *testing.T) {
	t.Setenv("BN_FROM_NUMBER", "+440000000000")
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+440000000000", cfg.FromNumber)
	assert.Equal(t, "+449876543210", cfg.ToNumber)
}

func TestLoadConfig_ToNumberFromEnv(t *testing.T) {
	t.Setenv("BN_TO_NUMBER", "+440000000000")
	path := writeConfigFile(t, `
from_number: "+441234567890"
to_number: "+449876543210"
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+441234567890", cfg.FromNumber)
	assert.Equal(t, "+440000000000", cfg.ToNumber)
}

func TestLoadConfig_PhoneNumbersFromEnvOverrideEmptyYAML(t *testing.T) {
	t.Setenv("BN_FROM_NUMBER", "+440000000000")
	t.Setenv("BN_TO_NUMBER", "+441111111111")
	path := writeConfigFile(t, `
locations:
  - label: Home
    scraper: bracknell
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
`)
	cfg, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "+440000000000", cfg.FromNumber)
	assert.Equal(t, "+441111111111", cfg.ToNumber)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run "TestLoadConfig_(FromNumberFromEnv|ToNumberFromEnv|PhoneNumbersFromEnvOverrideEmptyYAML)" ./pkg/config/ -v`

Expected: `TestLoadConfig_FromNumberFromEnv` FAIL (env var ignored, YAML value used). `TestLoadConfig_PhoneNumbersFromEnvOverrideEmptyYAML` FAIL (validation error "from_number is required").

**Step 3: Commit**

```bash
git add pkg/config/config_test.go
git commit -m "test: add failing tests for env var phone number overrides"
```

---

### Task 2: Implement env var overrides in LoadConfig

**Files:**
- Modify: `pkg/config/config.go` (lines 69-85, the `LoadConfig` function)

**Step 1: Add env var override logic**

In `pkg/config/config.go`, modify `LoadConfig()` to add env var checks between the YAML unmarshal and the `validate()` call. Replace the current function body:

```go
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if v := os.Getenv("BN_FROM_NUMBER"); v != "" {
		cfg.FromNumber = v
	}
	if v := os.Getenv("BN_TO_NUMBER"); v != "" {
		cfg.ToNumber = v
	}

	if err := validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
```

**Step 2: Run all tests to verify they pass**

Run: `go test ./pkg/config/ -v`

Expected: All tests PASS including the three new ones and all existing tests.

**Step 3: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat: support BN_FROM_NUMBER and BN_TO_NUMBER env vars"
```

---

### Task 3: Update documentation

**Files:**
- Modify: `CLAUDE.md` (Environment Variables section, around line 20)

**Step 1: Add new env vars to the documentation**

In `CLAUDE.md`, find the "Environment Variables" section. Add the new variables under a **Phone numbers** subheading before the existing **Application config** subheading:

```markdown
## Environment Variables

**Twilio (required):**
- `TWILIO_ACCOUNT_SID` - Twilio account SID
- `TWILIO_AUTH_TOKEN` - Twilio auth token

**Phone numbers (optional, overrides config file):**
- `BN_FROM_NUMBER` - Twilio "from" phone number (overrides `from_number` in config)
- `BN_TO_NUMBER` - Destination phone number (overrides `to_number` in config)

**Application config (alternative to CLI flags):**
- `BN_CONFIG_FILE` - Path to YAML config file
- `BN_DRY_RUN` - Set to `true` for dry-run mode
- `BN_TODAY_DATE` - Override today's date (YYYY-MM-DD)
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document BN_FROM_NUMBER and BN_TO_NUMBER env vars"
```

---

### Task 4: Run full test suite

**Step 1: Run all tests**

Run: `go test ./... -v`

Expected: All tests PASS.

**Step 2: Build to verify compilation**

Run: `go build -o bin-notifier ./cmd/notifier`

Expected: Build succeeds with no errors.
