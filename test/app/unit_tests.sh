#!/usr/bin/env bash
# Run app module integration tests.
# Usage: ./unit_tests.sh [test-name-filter]
# Examples:
#   ./unit_tests.sh                              # run all app tests (sequential)
#   ./unit_tests.sh TestInstall_AllAppsDefault   # run single test
#   ./unit_tests.sh TestContent                  # run all content tests
#   SKIP_deploy=true ./unit_tests.sh ...         # skip deploy stage (use saved state)
#   SKIP_cleanup=true ./unit_tests.sh ...        # skip cleanup (leave resources up)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

TEST_FILTER="${1:-}"
TIMEOUT="${TEST_TIMEOUT:-120m}"
PARALLELISM="${TEST_PARALLELISM:-1}"

echo "=== AWSO App Module Tests ==="
echo "Filter:      ${TEST_FILTER:-<all>}"
echo "Timeout:     ${TIMEOUT}"
echo "Parallelism: ${PARALLELISM}"
echo ""

echo "--- Building test binary ---"
go test -c ./test/app/ -o /dev/null
echo "Build OK"
echo ""

echo "--- Running tests ---"
if [[ -n "${TEST_FILTER}" ]]; then
    go test \
        -v \
        -run "${TEST_FILTER}" \
        -timeout "${TIMEOUT}" \
        -parallel "${PARALLELISM}" \
        ./test/app/ \
        2>&1 | tee "${SCRIPT_DIR}/test_output.log"
else
    go test \
        -v \
        -timeout "${TIMEOUT}" \
        -parallel "${PARALLELISM}" \
        ./test/app/ \
        2>&1 | tee "${SCRIPT_DIR}/test_output.log"
fi

echo ""
echo "=== Test run complete. Full output: ${SCRIPT_DIR}/test_output.log ==="
