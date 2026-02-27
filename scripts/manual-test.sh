#!/usr/bin/env bash
#
# Manual test script for multi-schedule collection days feature.
# Builds a Docker image and runs 8 test scenarios in dry-run mode.
#
# Usage: ./scripts/manual-test.sh -p "RG12 1AB" -a 12345
#
set -euo pipefail

usage() {
  echo "Usage: $0 -p <postcode> -a <address_code>"
  echo ""
  echo "  -p  Bracknell postcode (e.g. \"RG12 1AB\")"
  echo "  -a  Address code (numeric ID from council site)"
  exit 1
}

POSTCODE=""
ADDRESS_CODE=""

while getopts ":p:a:" opt; do
  case ${opt} in
    p) POSTCODE="${OPTARG}" ;;
    a) ADDRESS_CODE="${OPTARG}" ;;
    :) echo "Error: -${OPTARG} requires an argument" >&2; usage ;;
    *) usage ;;
  esac
done

if [[ -z "${POSTCODE}" || -z "${ADDRESS_CODE}" ]]; then
  echo "Error: both -p and -a are required" >&2
  usage
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOGS_DIR="${PROJECT_ROOT}/logs"

IMAGE_NAME="bin-notifier-test"
PASS=0
FAIL=0
TMPDIR_BASE=""

# ── Colours ──────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Colour

# ── Helpers ──────────────────────────────────────────────────────────

cleanup() {
  if [[ -n "${TMPDIR_BASE}" && -d "${TMPDIR_BASE}" ]]; then
    rm -rf "${TMPDIR_BASE}"
  fi
}
trap cleanup EXIT

pass() {
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}PASS${NC} $1"
}

fail() {
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}FAIL${NC} $1"
  if [[ -n "${2:-}" ]]; then
    echo -e "       ${RED}Output:${NC}"
    echo "$2" | head -20 | sed 's/^/       /'
  fi
}

run_scenario() {
  local num="$1" title="$2" config_file="$3" extra_flags="$4" expect_exit="$5" expect_grep="$6"

  echo -e "\n${CYAN}Scenario ${num}: ${title}${NC}"

  local output exit_code
  output=$(docker run --rm \
    -e TWILIO_ACCOUNT_SID=test \
    -e TWILIO_AUTH_TOKEN=test \
    -v "${config_file}:/config.yaml:ro" \
    "${IMAGE_NAME}" -c /config.yaml ${extra_flags} 2>&1) && exit_code=0 || exit_code=$?

  # Check exit code
  if [[ "${expect_exit}" == "nonzero" ]]; then
    if [[ ${exit_code} -ne 0 ]]; then
      pass "exit code non-zero (${exit_code})"
    else
      fail "expected non-zero exit but got 0" "${output}"
    fi
  else
    if [[ ${exit_code} -eq 0 ]]; then
      pass "exit code 0"
    else
      fail "expected exit 0 but got ${exit_code}" "${output}"
    fi
  fi

  # Check grep pattern in output
  if [[ -n "${expect_grep}" ]]; then
    if echo "${output}" | grep -qiE "${expect_grep}"; then
      pass "output matches: ${expect_grep}"
    else
      fail "output missing pattern: ${expect_grep}" "${output}"
    fi
  fi

  echo "${output}" > "${LOGS_DIR}/scenario-${num}.log"
}

write_config() {
  local file="$1"
  shift
  cat > "${file}" "$@"
}

# ── date helpers ─────────────────────────────────────────────────────

# Return the day-before date string for a given weekday name.
# Finds the next occurrence of that weekday from a base date and returns the day before.
day_before_weekday() {
  local weekday_name="$1" base_date="$2"
  # weekday_name e.g. "Tuesday", base_date e.g. "2026-03-01"
  local target_num
  case "${weekday_name,,}" in
    sunday)    target_num=0 ;;
    monday)    target_num=1 ;;
    tuesday)   target_num=2 ;;
    wednesday) target_num=3 ;;
    thursday)  target_num=4 ;;
    friday)    target_num=5 ;;
    saturday)  target_num=6 ;;
    *) echo "unknown weekday: ${weekday_name}" >&2; return 1 ;;
  esac

  # Find the next occurrence of the target weekday from base_date
  for i in $(seq 0 6); do
    local candidate
    candidate=$(date -j -v "+${i}d" -f "%Y-%m-%d" "${base_date}" "+%Y-%m-%d" 2>/dev/null || \
                date -d "${base_date} + ${i} days" "+%Y-%m-%d" 2>/dev/null)
    local candidate_dow
    candidate_dow=$(date -j -f "%Y-%m-%d" "${candidate}" "+%u" 2>/dev/null || \
                    date -d "${candidate}" "+%u" 2>/dev/null)
    # %u: Monday=1 ... Sunday=7; convert to 0-based with Sunday=0
    local candidate_num=$(( candidate_dow % 7 ))
    if [[ ${candidate_num} -eq ${target_num} ]]; then
      # Return the day before this date
      date -j -v "-1d" -f "%Y-%m-%d" "${candidate}" "+%Y-%m-%d" 2>/dev/null || \
        date -d "${candidate} - 1 day" "+%Y-%m-%d" 2>/dev/null
      return 0
    fi
  done
  echo "could not find weekday" >&2
  return 1
}

weekday_name_lc() {
  local d="$1"
  date -j -f "%Y-%m-%d" "${d}" "+%A" 2>/dev/null | tr '[:upper:]' '[:lower:]' || \
    date -d "${d}" "+%A" 2>/dev/null | tr '[:upper:]' '[:lower:]'
}

# Return a date N weeks offset from a reference date (same weekday)
offset_weeks() {
  local ref="$1" weeks="$2"
  date -j -v "+${weeks}w" -f "%Y-%m-%d" "${ref}" "+%Y-%m-%d" 2>/dev/null || \
    date -d "${ref} + $((weeks * 7)) days" "+%Y-%m-%d" 2>/dev/null
}

# ── Setup ────────────────────────────────────────────────────────────
echo -e "${BOLD}Multi-Schedule Collection Days — Manual Test Script${NC}"
echo -e "Postcode:     ${POSTCODE}"
echo -e "Address code: ${ADDRESS_CODE}"
echo ""
TMPDIR_BASE=$(mktemp -d)
rm -rf "${LOGS_DIR}"
mkdir -p "${LOGS_DIR}"
echo -e "${YELLOW}Logs directory:${NC} ${LOGS_DIR}"

# Build Docker image
echo -e "\n${BOLD}Building Docker image '${IMAGE_NAME}'...${NC}"
docker build -t "${IMAGE_NAME}" . || { echo -e "${RED}Docker build failed${NC}"; exit 1; }
echo -e "${GREEN}Build complete.${NC}"

# ═════════════════════════════════════════════════════════════════════
# GROUP 1: Config validation (no scraping needed)
# ═════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}═══ Group 1: Config Validation ═══${NC}"

# Scenario 1: Missing collection_days rejected
CFG1="${TMPDIR_BASE}/cfg1.yaml"
write_config "${CFG1}" <<'YAML'
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "Test"
    scraper: "bracknell"
    postcode: "RG12 1AB"
    address_code: "12345"
YAML
run_scenario 1 "Missing collection_days rejected" "${CFG1}" "-x" "nonzero" "collection_days must have at least one entry"

# Scenario 2: Empty types rejected
CFG2="${TMPDIR_BASE}/cfg2.yaml"
write_config "${CFG2}" <<'YAML'
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "Test"
    scraper: "bracknell"
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: []
YAML
run_scenario 2 "Empty types rejected" "${CFG2}" "-x" "nonzero" "types must have at least one entry"

# Scenario 3: Missing reference_date for fortnightly
CFG3="${TMPDIR_BASE}/cfg3.yaml"
write_config "${CFG3}" <<'YAML'
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "Test"
    scraper: "bracknell"
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
        every_n_weeks: 2
YAML
run_scenario 3 "Missing reference_date for fortnightly" "${CFG3}" "-x" "nonzero" "reference_date is required"

# Scenario 4: Invalid reference_date weekday
CFG4="${TMPDIR_BASE}/cfg4.yaml"
write_config "${CFG4}" <<'YAML'
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "Test"
    scraper: "bracknell"
    postcode: "RG12 1AB"
    address_code: "12345"
    collection_days:
      - day: tuesday
        types: ["General Waste"]
        every_n_weeks: 2
        reference_date: "2026-03-04"
YAML
# 2026-03-04 is a Wednesday, not a Tuesday
run_scenario 4 "Invalid reference_date weekday" "${CFG4}" "-x" "nonzero" "reference_date must fall on"

# ═════════════════════════════════════════════════════════════════════
# DISCOVERY: Scrape real data to learn actual collection dates
# ═════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}═══ Discovery: Scraping real collection dates ═══${NC}"
echo -e "Running a discovery scrape to find actual collection days..."

DISC_CFG="${TMPDIR_BASE}/discovery.yaml"
cat > "${DISC_CFG}" <<YAML
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "Discovery"
    scraper: "bracknell"
    postcode: "${POSTCODE}"
    address_code: "${ADDRESS_CODE}"
    collection_days:
      - day: monday
        types: ["General Waste"]
YAML

# Run discovery — we use a Monday date so it won't match and we just see the log output
DISC_OUTPUT=$(docker run --rm \
  -e TWILIO_ACCOUNT_SID=test \
  -e TWILIO_AUTH_TOKEN=test \
  -v "${DISC_CFG}:/config.yaml:ro" \
  "${IMAGE_NAME}" -c /config.yaml -x -d 2026-03-01 2>&1) || true

echo "${DISC_OUTPUT}" > "${LOGS_DIR}/discovery.log"
echo -e "${YELLOW}Discovery output:${NC}"
echo "${DISC_OUTPUT}" | grep -i "Next collection" | sed 's/^/  /' || echo "  (no collection lines found)"

# Extract a real collection date from the log
# Lines look like: [Discovery] Next collection for General Waste is 2026-03-10 00:00:00 +0000 UTC
FIRST_COLLECTION_DATE=$(echo "${DISC_OUTPUT}" | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}' | head -1 || true)

if [[ -z "${FIRST_COLLECTION_DATE}" ]]; then
  echo -e "${RED}Could not extract any collection dates from discovery run.${NC}"
  echo -e "Check ${LOGS_DIR}/discovery.log for details."
  echo -e "Skipping Group 2 scenarios."
  echo -e "\n${BOLD}═══ Results ═══${NC}"
  echo -e "  ${GREEN}Passed: ${PASS}${NC}"
  echo -e "  ${RED}Failed: ${FAIL}${NC}"
  [[ ${FAIL} -eq 0 ]] && exit 0 || exit 1
fi

echo -e "\n${YELLOW}First collection date found:${NC} ${FIRST_COLLECTION_DATE}"

# The day before the collection is when -d should be set so "tomorrow" = collection day
DAY_BEFORE=$(date -j -v "-1d" -f "%Y-%m-%d" "${FIRST_COLLECTION_DATE}" "+%Y-%m-%d" 2>/dev/null || \
             date -d "${FIRST_COLLECTION_DATE} - 1 day" "+%Y-%m-%d" 2>/dev/null)
COLLECTION_WEEKDAY=$(weekday_name_lc "${FIRST_COLLECTION_DATE}")

echo -e "${YELLOW}Collection weekday:${NC} ${COLLECTION_WEEKDAY}"
echo -e "${YELLOW}Will use -d date:${NC} ${DAY_BEFORE} (so tomorrow = ${FIRST_COLLECTION_DATE})"

# Find a second weekday that is different from the first collection day for scenario 8
SECOND_COLLECTION_DATE=$(echo "${DISC_OUTPUT}" | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}' | sort -u | while read -r d; do
  dw=$(weekday_name_lc "$d")
  if [[ "${dw}" != "${COLLECTION_WEEKDAY}" ]]; then
    echo "$d"
    break
  fi
done || true)

# ═════════════════════════════════════════════════════════════════════
# GROUP 2: Real scraping with notification logic
# ═════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}═══ Group 2: Real Scraping ═══${NC}"

# Scenario 5: Scraping works with new config format
CFG5="${TMPDIR_BASE}/cfg5.yaml"
cat > "${CFG5}" <<YAML
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "TestHome"
    scraper: "bracknell"
    postcode: "${POSTCODE}"
    address_code: "${ADDRESS_CODE}"
    collection_days:
      - day: ${COLLECTION_WEEKDAY}
        types: ["General Waste", "Recycling"]
YAML
run_scenario 5 "Scraping works with new config format" "${CFG5}" "-x -d ${DAY_BEFORE}" "0" "Tomorrows bin collections|Expected.*collection tomorrow|Next collection"

# Scenario 6: Fortnightly on-week shows expected warning
# Use the real collection date as the reference date (it's on-week by definition)
CFG6="${TMPDIR_BASE}/cfg6.yaml"
cat > "${CFG6}" <<YAML
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "TestHome"
    scraper: "bracknell"
    postcode: "${POSTCODE}"
    address_code: "${ADDRESS_CODE}"
    collection_days:
      - day: ${COLLECTION_WEEKDAY}
        types: ["General Waste"]
        every_n_weeks: 2
        reference_date: "${FIRST_COLLECTION_DATE}"
YAML
run_scenario 6 "Fortnightly on-week shows expected warning" "${CFG6}" "-x -d ${DAY_BEFORE}" "0" "Tomorrows bin collections|Expected.*collection tomorrow|Next collection"

# Scenario 7: Fortnightly off-week is silent
# We need tomorrow to be the same weekday as the collection but on a week where
# no real collection occurs, so the scraper results won't match tomorrow.
# Shift the -d date forward by 1 week — tomorrow will be the same weekday but
# the scraped dates won't match, so the app falls through to schedule logic.
# Then set the reference_date so this week is the OFF-week.
OFF_WEEK_DAY_BEFORE=$(offset_weeks "${DAY_BEFORE}" 1)
OFF_WEEK_TOMORROW=$(offset_weeks "${FIRST_COLLECTION_DATE}" 1)
CFG7="${TMPDIR_BASE}/cfg7.yaml"
cat > "${CFG7}" <<YAML
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "TestHome"
    scraper: "bracknell"
    postcode: "${POSTCODE}"
    address_code: "${ADDRESS_CODE}"
    collection_days:
      - day: ${COLLECTION_WEEKDAY}
        types: ["General Waste"]
        every_n_weeks: 2
        reference_date: "${FIRST_COLLECTION_DATE}"
YAML
# reference_date = real collection week, but -d puts us 1 week later (off-week)
run_scenario 7 "Fortnightly off-week is silent" "${CFG7}" "-x -d ${OFF_WEEK_DAY_BEFORE}" "0" "No collections tomorrow and not an expected collection day"

# Scenario 8: Multiple collection days
# Pick a different weekday for the second entry
if [[ -n "${SECOND_COLLECTION_DATE:-}" ]]; then
  SECOND_WEEKDAY=$(weekday_name_lc "${SECOND_COLLECTION_DATE}")
else
  # Fall back: pick a weekday two days after the collection day
  SECOND_WEEKDAY=$(weekday_name_lc "$(offset_weeks "${FIRST_COLLECTION_DATE}" 0 | \
    xargs -I{} date -j -v "+2d" -f "%Y-%m-%d" {} "+%Y-%m-%d" 2>/dev/null || \
    date -d "${FIRST_COLLECTION_DATE} + 2 days" "+%Y-%m-%d" 2>/dev/null)")
fi

CFG8="${TMPDIR_BASE}/cfg8.yaml"
cat > "${CFG8}" <<YAML
from_number: "+10000000000"
to_number: "+10000000001"
locations:
  - label: "TestHome"
    scraper: "bracknell"
    postcode: "${POSTCODE}"
    address_code: "${ADDRESS_CODE}"
    collection_days:
      - day: ${COLLECTION_WEEKDAY}
        types: ["General Waste"]
      - day: ${SECOND_WEEKDAY}
        types: ["Recycling"]
YAML
run_scenario 8 "Multiple collection days" "${CFG8}" "-x -d ${DAY_BEFORE}" "0" "Next collection"

# ═════════════════════════════════════════════════════════════════════
# Summary
# ═════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}═══ Results ═══${NC}"
echo -e "  ${GREEN}Passed: ${PASS}${NC}"
echo -e "  ${RED}Failed: ${FAIL}${NC}"
echo -e "  Logs saved in: ${LOGS_DIR}"

[[ ${FAIL} -eq 0 ]] && exit 0 || exit 1
