// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateBytesHex validates hexadecimal byte strings
// Format: [0-9A-Fa-f]+ with even length
// NO 0x prefix, case-insensitive
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateBytesHex(value []byte) bool {
	if len(value) == 0 {
		return false
	}

	// All characters must be hex digits
	for _, b := range value {
		if !isHexChar[b] {
			return false
		}
	}

	// Must be even length (each pair = one byte)
	if len(value)%2 != 0 {
		return false
	}

	return true
}

func BytesHexError() string {
	return "invalid bytes<hex> format (expected even-length hexadecimal)"
}
