#!/bin/bash

# This script runs all Go test files in the tests directory.

echo "Running all tests in the tests directory..."

go test ./tests/... -v

if [ $? -eq 0 ]; then
    echo "All tests passed successfully."
else
    echo "Some tests failed. Check the output above for details."
    exit 1
fi
