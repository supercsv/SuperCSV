// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeInt validates and decodes signed 64-bit integers with overflow checking
// Pattern: -?[0-9]+
// Range: -9223372036854775808 to 9223372036854775807
// No leading plus sign allowed
// No leading zeros except for "0" itself
// Negative zero "-0" is not allowed
// Returns (value, true) on success, (0, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeInt(value []byte) (int64, bool) {
	// Length check: min 1 char, max 20 chars ("-9223372036854775808")
	if len(value) == 0 || len(value) > 20 {
		return 0, false
	}

	i := 0
	negative := false

	// Handle optional minus sign (no plus allowed)
	if value[0] == '-' {
		if len(value) == 1 {
			return 0, false // Just a sign with no digits
		}
		negative = true
		i = 1
	}

	// Reject -0
	if negative && value[i] == '0' && len(value)-i == 1 {
		return 0, false
	}

	// Handle no leading zeros
	// After handling optional '-'
	if value[i] == '0' && len(value)-i > 1 {
		return 0, false
	}

	// Must have at least one digit
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if i >= len(value) || (value[i]-'0') > 9 {
		return 0, false
	}

	// Fast path: numbers with < 19 digits can't overflow int64
	// Max int64 is 9223372036854775807 (19 digits)
	// Min int64 is -9223372036854775808 (19 digits + sign)
	numDigits := len(value) - i
	if numDigits < 19 {
		// No overflow possible, parse the value
		var val int64
		for ; i < len(value); i++ {
			// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
			if (value[i] - '0') > 9 {
				return 0, false
			}
			digit := int64(value[i] - '0')
			if negative {
				val = val*10 - digit
			} else {
				val = val*10 + digit
			}
		}
		return val, true
	}

	// Slow path: 19+ digits, need overflow checking
	var val int64
	for ; i < len(value); i++ {
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		if (value[i] - '0') > 9 {
			return 0, false
		}

		digit := int64(value[i] - '0')

		if negative {
			// Check for overflow: value * 10 - digit < minInt64
			if val < -922337203685477580 || (val == -922337203685477580 && digit > 8) {
				return 0, false
			}
			val = val*10 - digit
		} else {
			// Check for overflow: value * 10 + digit > maxInt64
			if val > 922337203685477580 || (val == 922337203685477580 && digit > 7) {
				return 0, false
			}
			val = val*10 + digit
		}
	}

	return val, true
}
