#!/bin/bash
# CLI smoke test for SuperCSV validator
# Tests all spec examples and flag combinations

echo "Building supercsv-validate..."
go build -o supercsv-validate ./cmd/supercsv-validate

echo ""
echo "Checking version..."
version_output=$(./supercsv-validate --version)
echo "  $version_output"

echo ""
echo "Testing spec examples..."
spec_dir="internal/v1_0/spec/examples"
passed=0
failed=0

for file in "$spec_dir"/*.supr; do
    echo -n "  Testing $(basename "$file")..."
    if ./supercsv-validate "$file" > /dev/null 2>&1; then
        echo " OK"
        ((passed++))
    else
        echo " FAIL"
        ((failed++))
    fi
done

echo ""
echo "Testing output formats..."

# Test JSON output
echo -n "  Testing --json..."
if ./supercsv-validate --json "internal/v1_0/spec/examples/people.supr" > /dev/null 2>&1; then
    echo " OK"
    ((passed++))
else
    echo " FAIL"
    ((failed++))
fi

# Test default SUPR output format
echo -n "  Testing default supr format..."
output=$(./supercsv-validate "internal/v1_0/spec/examples/people.supr" 2>&1 || true)
if [[ $output == *"Line:int,ErrorSection:string,ErrorMsg:string"* ]]; then
    echo " OK"
    ((passed++))
else
    echo " FAIL"
    ((failed++))
fi

# Test quiet mode
echo -n "  Testing --quiet..."
if ./supercsv-validate --quiet "internal/v1_0/spec/examples/people.supr" > /dev/null 2>&1; then
    echo " OK"
    ((passed++))
else
    echo " FAIL"
    ((failed++))
fi

# Test strict version (should fail on file without version directive)
echo -n "  Testing --strict-version 1.0 (expect failure)..."
if ! ./supercsv-validate --strict-version 1.0 "internal/v1_0/spec/examples/people.supr" > /dev/null 2>&1; then
    echo " OK"
    ((passed++))
else
    echo " FAIL (expected failure, got success)"
    ((failed++))
fi

# Clean up
rm -f supercsv-validate

echo ""
echo "========================================="
echo "SMOKE TEST RESULTS"
echo "========================================="
echo "Passed: $passed"
echo "Failed: $failed"
echo "Total:  $((passed + failed))"
echo "========================================="

if [ $failed -gt 0 ]; then
    exit 1
fi

echo ""
echo "All smoke tests passed!"
exit 0
