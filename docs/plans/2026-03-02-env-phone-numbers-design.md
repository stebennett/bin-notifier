# Design: Environment Variable Support for Phone Numbers

**Date**: 2026-03-02
**Status**: Approved

## Problem

Phone numbers (`from_number`, `to_number`) can only be configured via the YAML config file. For deployment scenarios (Docker, CI, secrets management), it's useful to set these via environment variables.

## Design

### New Environment Variables

- `BN_FROM_NUMBER` -- Twilio "from" phone number
- `BN_TO_NUMBER` -- Destination phone number

Follows the existing `BN_` prefix convention.

### Precedence

Environment variable > YAML config file. If both are set, the env var wins. If neither is set, validation fails with the existing error message.

### Implementation

In `pkg/config/config.go`'s `LoadConfig()`, after YAML unmarshal and before calling `validate()`, apply env var overrides:

```go
if v := os.Getenv("BN_FROM_NUMBER"); v != "" {
    cfg.FromNumber = v
}
if v := os.Getenv("BN_TO_NUMBER"); v != "" {
    cfg.ToNumber = v
}
```

### Files Changed

1. `pkg/config/config.go` -- Add env var override logic in `LoadConfig()`
2. `pkg/config/config_test.go` -- Tests for env var override behavior
3. `CLAUDE.md` -- Document the new env vars

### What Doesn't Change

- `validate()` function unchanged
- `main()` unchanged
- No CLI flags for phone numbers
- Config file remains required (locations must be defined there)
