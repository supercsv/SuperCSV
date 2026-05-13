// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateDecimal validates arbitrary-precision decimal literals
// Pattern: -?[0-9]+(\.[0-9]+)?
// No leading plus sign allowed
// No -0 or -0.0 allowed
// No leading zeros (007, 042.5, etc.)
// Maximum total length: 128 characters
// Maximum fractional digits: 64
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateDecimal(value []byte) bool {
	// Length check: min 1 char, max 128 characters
	if len(value) == 0 || len(value) > 128 {
		return false
	}

	i := 0
	negative := false

	// Optional minus sign (no plus allowed)
	if value[0] == '-' {
		negative = true
		i++
		if i >= len(value) {
			return false
		}
	}

	// Must have at least one digit
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if (value[i] - '0') > 9 {
		return false
	}

	// Fast path for negative zero detection and validation
	if negative && value[i] == '0' {
		// Exactly "-0"
		if len(value)-i == 1 {
			return false
		}
		// Check for "-0.XXX" pattern
		if len(value)-i > 2 && value[i+1] == '.' {
			j := i + 2
			foundNonZero := false
			// Validate fractional digits AND check for non-zero
			// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
			for j < len(value) {
				digit := value[j] - '0'
				if digit > 9 {
					return false // invalid character
				}
				if digit != 0 {
					foundNonZero = true
				}
				j++
			}
			if !foundNonZero {
				return false // -0.0, -0.00, etc.
			}
			return true // valid -0.XXX with at least one non-zero
		}
	}

	// Reject leading zeros (007, 042.5, etc.)
	// If first digit is 0 and there's more after, next must be '.' or reject
	if value[i] == '0' && len(value)-i > 1 && value[i+1] != '.' {
		return false
	}

	i++

	// Integer part
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	for i < len(value) && (value[i]-'0') <= 9 {
		i++
	}

	// Optional decimal point and fraction
	if i < len(value) && value[i] == '.' {
		i++
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		if i >= len(value) || (value[i]-'0') > 9 {
			return false // Must have digit after decimal
		}
		fracStart := i
		i++
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		for i < len(value) && (value[i]-'0') <= 9 {
			i++
		}
		// Maximum fractional digits: 64
		if i-fracStart > 64 {
			return false
		}
	}

	return i == len(value)
}

func DecimalError() string {
	return "invalid decimal literal"
}
