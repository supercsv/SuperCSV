// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateDuration validates ISO 8601 duration strings using lookup tables
// Format: P[n]D[T[n]H[n]M[n]S]
// - Must start with 'P'
// - Date part: optional nD (days)
// - Time part: optional T followed by optional nH (hours), nM (minutes), nS (seconds)
// - At least one component (day, hour, minute, or second) must be present
// - Seconds can have fractional part (e.g., 0.5S)
// - All numeric values must be non-negative
// This version uses isDigit lookup tables for optimal performance
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateDuration(value []byte) bool {
	if len(value) < 2 {
		return false
	}
	if value[0] != 'P' {
		return false
	}

	i := 1
	hasComponent := false

	// Date part: optional digits followed by 'D'
	digitStart := i
	for i < len(value) && isDigit[value[i]] {
		i++
	}
	if i > digitStart && i < len(value) && value[i] == 'D' {
		hasComponent = true
		i++
		if i >= len(value) {
			return hasComponent // Valid: P5D
		}
	}

	// Time part: must start with 'T'
	if i < len(value) && value[i] == 'T' {
		i++
		if i >= len(value) {
			return false // 'T' with nothing after is invalid
		}

		// Parse hours, minutes, seconds
		for i < len(value) {
			digitStart = i
			for i < len(value) && isDigit[value[i]] {
				i++
			}

			if i == digitStart {
				return false // Expected digits
			}

			if i >= len(value) {
				return false // Expected unit marker
			}

			unit := value[i]
			i++

			switch unit {
			case 'H', 'M':
				hasComponent = true
			case 'S':
				hasComponent = true
				// Check for fractional seconds
				if i < len(value) {
					return false // 'S' must be last
				}
			case '.':
				// Fractional seconds: the outer loop already read digits (whole part),
				// then read '.' as the unit marker and advanced i past it.
				// So i already points to the first fractional digit - no backtracking needed.

				if i >= len(value) || !isDigit[value[i]] {
					return false // Must have digit after decimal
				}
				fracStart := i
				for i < len(value) && isDigit[value[i]] {
					i++
				}
				if i-fracStart > 9 {
					return false // Max 9 fractional digits (nanosecond precision)
				}
				if i >= len(value) || value[i] != 'S' {
					return false // Must end with 'S'
				}
				i++
				if i < len(value) {
					return false // 'S' must be last
				}
				hasComponent = true
			default:
				return false // Invalid unit
			}

			if i >= len(value) {
				break
			}
		}
	}

	return i == len(value) && hasComponent
}

func DurationError() string {
	return "invalid duration format"
}
