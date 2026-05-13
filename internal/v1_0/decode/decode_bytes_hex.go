// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeBytesHex validates and decodes hexadecimal byte strings
// Format: [0-9A-Fa-f]+ with even length
// NO 0x prefix, case-insensitive
// Returns ([]byte, true) on success, (nil, false) on failure
// HOT_PATH -> minimal allocations, ASCII only
func decodeBytesHex(value []byte) ([]byte, bool) {
	if len(value) == 0 {
		return nil, false
	}

	// All characters must be hex digits
	for _, b := range value {
		if !isHexChar[b] {
			return nil, false
		}
	}

	// Must be even length (each pair = one byte)
	if len(value)%2 != 0 {
		return nil, false
	}

	// Decode hex to bytes
	result := make([]byte, len(value)/2)
	for i := 0; i < len(value); i += 2 {
		high := hexCharToNibble(value[i])
		low := hexCharToNibble(value[i+1])
		result[i/2] = (high << 4) | low
	}

	return result, true
}
