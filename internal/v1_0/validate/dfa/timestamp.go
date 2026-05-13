// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// Days in each month (non-leap year)
var daysInMonth = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// ValidateTimestamp validates ISO 8601 timestamp using position-based checks.
// Format: YYYY-MM-DDTHH:MM:SS[.fff][Z|+/-HH:MM]
// Validates actual calendar dates (e.g., rejects Feb 30, Nov 31, non-leap Feb 29)
func ValidateTimestamp(b []byte) bool {
	n := len(b)

	// Length check: minimum 19 (YYYY-MM-DDTHH:MM:SS), maximum 35
	if n < 19 || n > 35 {
		return false
	}

	// Fixed positions validation
	// Position 4: date separator (- or /)
	dateSep := b[4]
	if !isDateSep[dateSep] {
		return false
	}

	// Position 7: same date separator
	if b[7] != dateSep {
		return false
	}

	// Position 10: datetime separator (T or space)
	if !isDateTimeSep[b[10]] {
		return false
	}

	// Position 13, 16: time colons
	if b[13] != ':' || b[16] != ':' {
		return false
	}

	// Validate year (positions 0-3): any 4 digits
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if (b[0]-'0') > 9 || (b[1]-'0') > 9 || (b[2]-'0') > 9 || (b[3]-'0') > 9 {
		return false
	}
	year := int(b[0]-'0')*1000 + int(b[1]-'0')*100 + int(b[2]-'0')*10 + int(b[3]-'0')

	// Validate month (positions 5-6): 01-12
	if !validMonth[b[5]][b[6]] {
		return false
	}
	month := int(b[5]-'0')*10 + int(b[6]-'0')

	// Validate day (positions 8-9): format check 01-31
	if !validDay[b[8]][b[9]] {
		return false
	}
	day := int(b[8]-'0')*10 + int(b[9]-'0')

	// Calendar validation: check actual days in month
	maxDay := daysInMonth[month]
	if month == 2 && (year&3 == 0) && (year%100 != 0 || year%400 == 0) {
		maxDay = 29
	}
	if day > maxDay {
		return false
	}

	// Validate hour (positions 11-12): 00-23
	if !validHour[b[11]][b[12]] {
		return false
	}

	// Validate minute (positions 14-15): 00-59
	// isDigit0to5 check: (b - '0') <= 5 validates digits 0-5
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if (b[14]-'0') > 5 || (b[15]-'0') > 9 {
		return false
	}

	// Validate second (positions 17-18): 00-59
	// isDigit0to5 check: (b - '0') <= 5 validates digits 0-5
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if (b[17]-'0') > 5 || (b[18]-'0') > 9 {
		return false
	}

	// Position 19+: optional fraction, timezone
	if n == 19 {
		return true // YYYY-MM-DDTHH:MM:SS
	}

	pos := 19

	// Optional fraction: .ddd...
	if b[pos] == '.' {
		pos++
		if pos >= n {
			return false // dot without digits
		}
		// Consume fraction digits
		// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
		for pos < n && (b[pos]-'0') <= 9 {
			pos++
		}
		if pos == 20 {
			return false // dot without fraction digits
		}
	}

	// Position should be at end or timezone
	if pos == n {
		return true // no timezone
	}

	// Timezone: Z or +/-HH:MM
	if b[pos] == 'Z' {
		return pos+1 == n // Z must be last character
	}

	// Timezone offset: +/-HH:MM (6 characters remaining)
	if !isTZSign[b[pos]] {
		return false
	}
	pos++

	// Need exactly 5 more characters: HH:MM
	if n-pos != 5 {
		return false
	}

	// Validate TZ hour: 00-23
	if !validHour[b[pos]][b[pos+1]] {
		return false
	}

	// Validate TZ colon
	if b[pos+2] != ':' {
		return false
	}

	// Validate TZ minute: 00-59
	// isDigit0to5 check: (b - '0') <= 5 validates digits 0-5
	// isDigit check: (b - '0') <= 9 validates ASCII digits 0x30-0x39
	if (b[pos+3]-'0') > 5 || (b[pos+4]-'0') > 9 {
		return false
	}

	return true
}

func TimestampError() string {
	return "invalid timestamp format (expected ISO 8601)"
}
