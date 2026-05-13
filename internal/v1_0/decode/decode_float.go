// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"math"
	"strconv"
	"unsafe"
)

// unsafeString converts []byte to string without allocation
// SAFETY: Only safe when the []byte is not modified after conversion
// and the string is used only during the function's lifetime
func unsafeString(b []byte) string {
	return unsafe.String(&b[0], len(b))
}

// decodeFloat validates and decodes IEEE 754 double precision floating point numbers
// Pattern: -?[0-9]+(.[0-9]+)?([eE][+-]?[0-9]+)? or special values
// Special values (case-insensitive): nan, inf, -inf
// NOTE: +inf is NOT allowed (only unsigned or negative)
// No leading plus sign allowed on the number itself
// Returns (value, true) on success, (0, false) on failure
// HOT_PATH -> zero allocations for validation, ASCII only, no []byte->string
func decodeFloat(value []byte) (float64, bool) {
	if len(value) == 0 {
		return 0, false
	}

	// Check for special values (case-insensitive)
	if len(value) == 3 {
		// "nan" or "inf"
		if (value[0]|0x20) == 'n' &&
			(value[1]|0x20) == 'a' &&
			(value[2]|0x20) == 'n' {
			return math.NaN(), true
		}
		if (value[0]|0x20) == 'i' &&
			(value[1]|0x20) == 'n' &&
			(value[2]|0x20) == 'f' {
			return math.Inf(1), true
		}
	}
	if len(value) == 4 && value[0] == '-' {
		// "-inf"
		if (value[1]|0x20) == 'i' &&
			(value[2]|0x20) == 'n' &&
			(value[3]|0x20) == 'f' {
			return math.Inf(-1), true
		}
	}

	// Parse numeric float: -?[0-9]+(.[0-9]+)?([eE][+-]?[0-9]+)?
	i := 0

	// Optional minus sign (no plus allowed)
	if value[0] == '-' {
		i++
		if i >= len(value) {
			return 0, false
		}
	}

	// Must have at least one digit before decimal or exponent
	if !isDigit[value[i]] {
		return 0, false
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
			return 0, false // Must have digit after decimal
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
			return 0, false // Must have exponent value
		}
		// Optional exponent sign
		if value[i] == '+' || value[i] == '-' {
			i++
			if i >= len(value) {
				return 0, false
			}
		}
		// Must have at least one exponent digit
		if !isDigit[value[i]] {
			return 0, false
		}
		i++
		for i < len(value) && isDigit[value[i]] {
			i++
		}
	}

	// Must have consumed entire input
	if i != len(value) || !hasDigit {
		return 0, false
	}

	// Use strconv.ParseFloat to convert the validated string to float64
	// PERF: unsafe conversion avoids allocation since value is read-only
	result, err := strconv.ParseFloat(unsafeString(value), 64)
	if err != nil {
		return 0, false
	}

	return result, true
}
