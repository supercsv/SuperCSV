// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeDuration validates and decodes ISO 8601 duration strings
// Format: P[n]D[T[n]H[n]M[n]S]
// - Must start with 'P'
// - Date part: optional nD (days)
// - Time part: optional T followed by optional nH (hours), nM (minutes), nS (seconds)
// - At least one component (day, hour, minute, or second) must be present
// - Seconds can have fractional part (e.g., 0.5S)
// - All numeric values must be non-negative
// Returns (Duration, true) on success, (Duration{}, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeDuration(value []byte) (Duration, bool) {
	if len(value) < 2 {
		return Duration{}, false
	}
	if value[0] != 'P' {
		return Duration{}, false
	}

	i := 1
	hasComponent := false
	result := Duration{}

	// Date part: optional digits followed by 'D'
	digitStart := i
	for i < len(value) && isDigit[value[i]] {
		i++
	}
	if i > digitStart && i < len(value) && value[i] == 'D' {
		// Parse days
		days := 0
		for j := digitStart; j < i; j++ {
			days = days*10 + int(value[j]-'0')
		}
		result.Days = days
		hasComponent = true
		i++
		if i >= len(value) {
			return result, hasComponent // Valid: P5D
		}
	}

	// Time part: must start with 'T'
	if i < len(value) && value[i] == 'T' {
		i++
		if i >= len(value) {
			return Duration{}, false // 'T' with nothing after is invalid
		}

		// Parse hours, minutes, seconds
		for i < len(value) {
			digitStart = i
			for i < len(value) && isDigit[value[i]] {
				i++
			}

			if i == digitStart {
				return Duration{}, false // Expected digits
			}

			// Parse the integer value
			val := 0
			for j := digitStart; j < i; j++ {
				val = val*10 + int(value[j]-'0')
			}

			if i >= len(value) {
				return Duration{}, false // Expected unit marker
			}

			unit := value[i]
			i++

			switch unit {
			case 'H':
				result.Hours = val
				hasComponent = true
			case 'M':
				result.Minutes = val
				hasComponent = true
			case 'S':
				result.Seconds = val
				hasComponent = true
				// Check for fractional seconds
				if i < len(value) {
					return Duration{}, false // 'S' must be last
				}
			case '.':
				// Fractional seconds: we're in the middle of seconds
				// The value we just parsed is the whole part
				result.Seconds = val
				// Parse fractional part
				nanos := 0
				fracStart := i
				for i < len(value) && isDigit[value[i]] {
					if i-fracStart < 9 {
						nanos = nanos*10 + int(value[i]-'0')
					}
					i++
				}
				if fracStart == i {
					return Duration{}, false // Must have digit after decimal
				}
				// Pad to nanoseconds
				for j := i - fracStart; j < 9; j++ {
					nanos *= 10
				}
				result.Nanos = nanos

				if i >= len(value) || value[i] != 'S' {
					return Duration{}, false // Must end with 'S'
				}
				i++
				if i < len(value) {
					return Duration{}, false // 'S' must be last
				}
				hasComponent = true
			default:
				return Duration{}, false // Invalid unit
			}

			if i >= len(value) {
				break
			}
		}
	}

	if i != len(value) || !hasComponent {
		return Duration{}, false
	}

	return result, true
}
