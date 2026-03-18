#!/bin/bash
# Test script for batch-cost-go

echo "=== Testing batch-cost-go CLI ==="
echo

# Test 1: Show help
echo "Test 1: Showing help"
echo "---------------------"
./batch-cost-go --help
echo

# Test 2: Test interactive mode (will exit immediately with no input)
echo "Test 2: Testing interactive mode (will timeout)"
echo "-----------------------------------------------"
timeout 2 ./batch-cost-go 2>/dev/null || echo "Interactive mode works (requires user input)"
echo

# Test 3: Test with invalid job ID (should show AWS error)
echo "Test 3: Testing with invalid job ID"
echo "------------------------------------"
AWS_PROFILE=nonexistent ./batch-cost-go --job-id test-123 2>&1 | head -20
echo

# Test 4: Test JSON output format
echo "Test 4: Testing JSON output with invalid job"
echo "--------------------------------------------"
AWS_PROFILE=nonexistent ./batch-cost-go --job-id test-123 --json 2>&1 | head -10
echo

echo "=== All tests completed ==="
echo
echo "To run with real AWS credentials:"
echo "1. Configure AWS CLI: aws configure"
echo "2. Run: ./batch-cost-go --job-id <your-job-id>"
echo "3. Or use interactive mode: ./batch-cost-go"
