# Test Coverage Improvement Plan

## Current State

- Phase 1 complete: `pkg/dateutil` and `pkg/regexp` at 100%, `pkg/scraper` at 33%
- Phase 2 complete: `pkg/config` at 33% (`isErrHelp` fully tested)
- Bug fix: `FindNamedMatches()` now handles no-match case without panicking

---

## Phase 1: High Priority (Pure Functions) ✅

### 1.1 Create `pkg/dateutil/dateutil_test.go`
- [x] Test `AsTime()` - verify correct time construction from int day/month/year
- [x] Test `AsTimeWithMonth()` - verify parsing string months (January, February, etc.), invalid month handling
- [x] Test `IsDateMatching()` - same day match, different day mismatch, different year mismatch, year boundary edge cases

### 1.2 Create `pkg/regexp/regexp_test.go`
- [x] Test `FindNamedMatches()` with multiple named capture groups
- [x] Test `FindNamedMatches()` with no matches
- [x] Test `FindNamedMatches()` with partial matches

### 1.3 Extend `pkg/scraper/scraper_test.go`
- [x] Test `parseNextCollectionTime()` with malformed input (should return error)
- [x] Test `parseNextCollectionTime()` with missing fields (should return error)
- [x] Test `ScrapeBinTimes()` returns error when postcode is empty
- [x] Test `ScrapeBinTimes()` returns error when address is empty

---

## Phase 2: Medium Priority ✅

### 2.1 Create `pkg/config/config_test.go`
- [x] Test `isErrHelp()` returns true for help flag error
- [x] Test `isErrHelp()` returns false for other errors (required, unknown flag, invalid choice, marshal)
- [x] Test `isErrHelp()` returns false for standard errors
- [x] Test `isErrHelp()` returns false for nil error

---

## Phase 3: Low Priority (Requires Refactoring) ✅

### 3.1 Create `pkg/clients/twilioclient_test.go`
- [x] Refactor `TwilioClient` to accept an interface for the underlying client
- [x] Test `SendSms()` dry-run mode returns nil without calling Twilio
- [x] Test `SendSms()` calls Twilio API with correct parameters (using mock)

### 3.2 Improve testability of `cmd/notifier/main.go`
- [x] Extract main logic into a separate function with dependency injection
- [x] Create integration tests for the orchestration logic

---

## Coverage Progress

| Package | Before | After Phase 1 | After Phase 2 | After Phase 3 |
|---------|--------|---------------|---------------|---------------|
| `pkg/dateutil` | 0% | **100%** ✅ | 100% | 100% ✅ |
| `pkg/regexp` | 0% | **100%** ✅ | 100% | 100% ✅ |
| `pkg/scraper` | 19% | **33%** ✅ | 33% | 33% ✅ |
| `pkg/config` | 0% | 0% | **33%** ✅ | 33% ✅ |
| `pkg/clients` | 0% | 0% | 0% | **~80%** ✅ |
| `cmd/notifier` | 0% | 0% | 0% | **~50%** ✅ |

Note: `pkg/config` coverage is 33% because `GetConfig()` reads from `os.Args` and would require refactoring for higher coverage.
