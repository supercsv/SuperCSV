// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeTime validates and decodes ISO 8601 time: HH:MM:SS or HH:MM:SS.fffffffff
// Timezone offsets are NOT permitted
// Returns (Time, true) on success, (Time{}, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeTime(value []byte) (Time, bool) {
	// Length check: HH:MM:SS (8 chars min), HH:MM:SS.fffffffff (18 chars max)
	if len(value) < 8 || len(value) > 18 {
		return Time{}, false
	}

	// Check structure: HH:MM:SS
	if value[2] != ':' || value[5] != ':' {
		return Time{}, false
	}

	// Validate HH:MM:SS using 2-char lookup tables
	if !validHour[value[0]][value[1]] || !validMinSec[value[3]][value[4]] || !validMinSec[value[6]][value[7]] {
		return Time{}, false
	}

	// Extract hour, minute, second
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	second := int(value[6]-'0')*10 + int(value[7]-'0')
	nanos := 0

	// Optional fractional seconds
	if len(value) > 8 {
		if value[8] != '.' {
			return Time{}, false
		}
		// Must have at least one digit after decimal
		if len(value) == 9 {
			return Time{}, false
		}
		// Parse fractional seconds (up to 9 digits for nanoseconds)
		fracLen := len(value) - 9
		if fracLen > 9 {
			fracLen = 9 // truncate to nanosecond precision
		}

		for i := 9; i < len(value); i++ {
			// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
			if (value[i] - '0') > 9 {
				return Time{}, false
			}
			// Build nanoseconds value (pad with zeros if < 9 digits)
			if i-9 < 9 {
				nanos = nanos*10 + int(value[i]-'0')
			}
		}
		// Pad to nanoseconds if fewer than 9 digits
		for i := fracLen; i < 9; i++ {
			nanos *= 10
		}
	}

	return Time{Hour: hour, Minute: minute, Second: second, Nanos: nanos}, true
}
