// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateInt validates signed 64-bit integers with overflow checking
// Pattern: -?[0-9]+
// Range: -9223372036854775808 to 9223372036854775807
// No leading plus sign allowed
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateInt(value []byte) bool {
	// Length check: min 1 char, max 20 chars ("-9223372036854775808")
	if len(value) == 0 || len(value) > 20 {
		return false
	}

	i := 0
	negative := false

	// Handle optional minus sign (no plus allowed)
	if value[0] == '-' {
		if len(value) == 1 {
			return false // Just a sign with no digits
		}
		negative = true
		i = 1
	}

	// reject -0
	if negative && value[i] == '0' && len(value)-i == 1 {
		return false
	}

	// Handle no leading zeros
	// After handling optional '-'
	if value[i] == '0' && len(value)-i > 1 {
		return false
	}

	// Must have at least one digit
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if i >= len(value) || (value[i]-'0') > 9 {
		return false
	}

	// Fast path: numbers with < 19 digits can't overflow int64
	// Max int64 is 9223372036854775807 (19 digits)
	// Min int64 is -9223372036854775808 (19 digits + sign)
	numDigits := len(value) - i
	if numDigits < 19 {
		// No overflow possible, just validate remaining digits
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		for i++; i < len(value); i++ {
			if (value[i] - '0') > 9 {
				return false
			}
		}
		return true
	}

	// Slow path: 19+ digits, need overflow checking
	var val int64
	for ; i < len(value); i++ {
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		if (value[i] - '0') > 9 {
			return false
		}

		digit := int64(value[i] - '0')

		if negative {
			// Check for overflow: value * 10 - digit < minInt64
			if val < -922337203685477580 || (val == -922337203685477580 && digit > 8) {
				return false
			}
			val = val*10 - digit
		} else {
			// Check for overflow: value * 10 + digit > maxInt64
			if val > 922337203685477580 || (val == 922337203685477580 && digit > 7) {
				return false
			}
			val = val*10 + digit
		}
	}

	return true
}

// IntErrorFor classifies an invalid int literal as either a format error or an overflow.
// This runs only on invalid inputs after ValidateInt has already returned false, so it does
// not affect the valid-value hot path.
// The classifier is intentionally simple: it uses a cheap length/sign pre-check to choose
// message type rather than re-running full numeric overflow validation.
func IntErrorFor(value []byte) string {
	if inputHasTooManyCharsForInt(value) {
		return "int overflow: input has too many chars for int"
	}
	return "invalid int format"
}

func inputHasTooManyCharsForInt(value []byte) bool {
	if len(value) == 0 {
		return false
	}

	if value[0] == '+' {
		return false
	}

	i := 0
	if value[0] == '-' {
		if len(value) == 1 {
			return false
		}
		i = 1
	}

	return len(value)-i >= 19
}

func IntError() string {
	return "invalid int format or overflow"
}
