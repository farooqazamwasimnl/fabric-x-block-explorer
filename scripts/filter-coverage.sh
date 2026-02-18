#!/bin/bash
# Copyright IBM Corp. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Filter coverage output to exclude:
# - main.go files (application entry points)
# - *_test.go files (test files)
# - test_exports.go files (test helper exports)
# - .pb.go files (generated protobuf code)
# - sqlc generated files (db/sqlc/*.go)

set -e

if [ $# -ne 2 ]; then
    echo "Usage: $0 <input-coverage-file> <output-coverage-file>"
    exit 1
fi

INPUT_FILE=$1
OUTPUT_FILE=$2

if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file '$INPUT_FILE' not found"
    exit 1
fi

# Extract the first line (mode line)
MODE_LINE=$(head -n 1 "$INPUT_FILE")
echo "$MODE_LINE" > "$OUTPUT_FILE"

# Filter out unwanted files (skip the first line which is the mode)
tail -n +2 "$INPUT_FILE" | grep -v -E '(main\.go:|_test\.go:|test_exports\.go:|\.pb\.go:|/sqlc/.*\.go:)' >> "$OUTPUT_FILE" || true

echo "Filtered coverage written to: $OUTPUT_FILE"
