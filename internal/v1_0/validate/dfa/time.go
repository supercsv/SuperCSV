// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateTime validates ISO 8601 time: HH:MM:SS or HH:MM:SS.fffffffff
// Timezone offsets are NOT permitted
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateTime(value []byte) bool {
	// Length check: HH:MM:SS (8 chars min), HH:MM:SS.fffffffff (18 chars max)
	if len(value) < 8 || len(value) > 18 {
		return false
	}

	// Check structure: HH:MM:SS
	if value[2] != ':' || value[5] != ':' {
		return false
	}

	// Validate HH:MM:SS using 2-char lookup tables
	if !validHour[value[0]][value[1]] || !validMinSec[value[3]][value[4]] || !validMinSec[value[6]][value[7]] {
		return false
	}

	// Optional fractional seconds
	if len(value) > 8 {
		if value[8] != '.' {
			return false
		}
		// Must have at least one digit after decimal
		if len(value) == 9 {
			return false
		}
		// All remaining characters must be digits (up to 9 for nanoseconds)
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		for i := 9; i < len(value); i++ {
			if (value[i] - '0') > 9 {
				return false
			}
		}
	}

	return true
}

func TimeError() string {
	return "invalid time format (expected HH:MM:SS or HH:MM:SS.fffffffff)"
}
