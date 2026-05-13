// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateFloat validates IEEE 754 double precision floating point numbers using lookup tables
// Pattern: -?[0-9]+(.[0-9]+)?([eE][+-]?[0-9]+)? or special values
// Special values (case-insensitive): nan, inf, -inf
// NOTE: +inf is NOT allowed (only unsigned or negative)
// No leading plus sign allowed on the number itself
// This version uses isDigit lookup tables for optimal performance
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateFloat(value []byte) bool {
	if len(value) == 0 {
		return false
	}

	// Check for special values (case-insensitive)
	if len(value) == 3 {
		// "nan" or "inf"
		if (value[0]|0x20) == 'n' &&
			(value[1]|0x20) == 'a' &&
			(value[2]|0x20) == 'n' {
			return true
		}
		if (value[0]|0x20) == 'i' &&
			(value[1]|0x20) == 'n' &&
			(value[2]|0x20) == 'f' {
			return true
		}
	}
	if len(value) == 4 && value[0] == '-' {
		// "-inf"
		if (value[1]|0x20) == 'i' &&
			(value[2]|0x20) == 'n' &&
			(value[3]|0x20) == 'f' {
			return true
		}
	}

	// Parse numeric float: -?[0-9]+(.[0-9]+)?([eE][+-]?[0-9]+)?
	i := 0

	// Optional minus sign (no plus allowed)
	if value[0] == '-' {
		i++
		if i >= len(value) {
			return false
		}
	}

	// Must have at least one digit before decimal or exponent
	if !isDigit[value[i]] {
		return false
	}
	hasDigit := true
	i++

	// Integer part
	for i < len(value) && isDigit[value[i]] {
		i++
	}

	// Optional decimal point and fraction
	if i < len(value) && value[i] == '.' {
		i++
		if i >= len(value) || !isDigit[value[i]] {
			return false // Must have digit after decimal
		}
		i++
		for i < len(value) && isDigit[value[i]] {
			i++
		}
	}

	// Optional exponent
	if i < len(value) && (value[i]|0x20) == 'e' {
		i++
		if i >= len(value) {
			return false // Must have exponent value
		}
		// Optional exponent sign
		if value[i] == '+' || value[i] == '-' {
			i++
			if i >= len(value) {
				return false
			}
		}
		// Must have at least one exponent digit
		if !isDigit[value[i]] {
			return false
		}
		i++
		for i < len(value) && isDigit[value[i]] {
			i++
		}
	}

	// Must have consumed entire input
	return i == len(value) && hasDigit
}

func FloatError() string {
	return "invalid float format"
}
