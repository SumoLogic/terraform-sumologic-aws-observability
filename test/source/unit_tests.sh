#!/usr/bin/env bash
# Run source module integration tests.
# Usage: ./unit_tests.sh [test-name-filter]
# Examples:
#   ./unit_tests.sh                            # run all source tests (sequential)
#   ./unit_tests.sh TestBasic_AllDefaults      # run single test
#   ./unit_tests.sh TestAutoEnable             # run all autoenable tests
#   SKIP_deploy=true ./unit_tests.sh ...       # skip deploy stage (use saved state)
#   SKIP_cleanup=true ./unit_tests.sh ...      # skip cleanup (leave infra up for debugging)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

# Default test filter (run all if not specified)
TEST_FILTER="${1:-}"
TIMEOUT="${TEST_TIMEOUT:-180m}"
PARALLELISM="${TEST_PARALLELISM:-1}"

echo "=== AWSO Source Module Tests ==="
echo "Filter:      ${TEST_FILTER:-<all>}"
echo "Timeout:     ${TIMEOUT}"
echo "Parallelism: ${PARALLELISM}"
echo ""

# Build the test binary first to catch compilation errors early
echo "--- Building test binary ---"
go test -c ./test/source/ -o /dev/null
echo "Build OK"
echo ""

# Run tests
echo "--- Running tests ---"
if [[ -n "${TEST_FILTER}" ]]; then
    go test \
        -v \
        -run "${TEST_FILTER}" \
        -timeout "${TIMEOUT}" \
        -parallel "${PARALLELISM}" \
        ./test/source/ \
        2>&1 | tee "${SCRIPT_DIR}/test_output.log"
else
    go test \
        -v \
        -timeout "${TIMEOUT}" \
        -parallel "${PARALLELISM}" \
        ./test/source/ \
        2>&1 | tee "${SCRIPT_DIR}/test_output.log"
fi

echo ""
echo "=== Test run complete. Full output: ${SCRIPT_DIR}/test_output.log ==="
