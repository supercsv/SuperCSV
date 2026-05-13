// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateBool validates boolean values: true, false, 1, 0 (case-insensitive for text forms)
// Pattern: true|false|1|0
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateBool(value []byte) bool {
	switch len(value) {
	case 1:
		// '0' and '1' differ only in the lowest bit: '0' = 0x30, '1' = 0x31
		return (value[0] & 0xFE) == '0'
	case 4:
		// "true" or "TRUE" or mixed case
		// Use bitwise OR with 0x20 to convert to lowercase: 'T'|0x20 = 't', 'R'|0x20 = 'r'
		return (value[0]|0x20) == 't' &&
			(value[1]|0x20) == 'r' &&
			(value[2]|0x20) == 'u' &&
			(value[3]|0x20) == 'e'
	case 5:
		// "false" or "FALSE" or mixed case
		return (value[0]|0x20) == 'f' &&
			(value[1]|0x20) == 'a' &&
			(value[2]|0x20) == 'l' &&
			(value[3]|0x20) == 's' &&
			(value[4]|0x20) == 'e'
	default:
		return false
	}
}

func BoolError() string {
	return "bool must be true/false/1/0"
}
