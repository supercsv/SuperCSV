// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeBool validates and decodes boolean values: true, false, 1, 0 (case-insensitive for text forms)
// Pattern: true|false|1|0
// Returns (value, true) on success, (false, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeBool(value []byte) (bool, bool) {
	switch len(value) {
	case 1:
		// '0' and '1' differ only in the lowest bit: '0' = 0x30, '1' = 0x31
		if (value[0] & 0xFE) == '0' {
			return value[0] == '1', true
		}
		return false, false
	case 4:
		// "true" or "TRUE" or mixed case
		// Use bitwise OR with 0x20 to convert to lowercase: 'T'|0x20 = 't', 'R'|0x20 = 'r'
		if (value[0]|0x20) == 't' &&
			(value[1]|0x20) == 'r' &&
			(value[2]|0x20) == 'u' &&
			(value[3]|0x20) == 'e' {
			return true, true
		}
		return false, false
	case 5:
		// "false" or "FALSE" or mixed case
		if (value[0]|0x20) == 'f' &&
			(value[1]|0x20) == 'a' &&
			(value[2]|0x20) == 'l' &&
			(value[3]|0x20) == 's' &&
			(value[4]|0x20) == 'e' {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}
