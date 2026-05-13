// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// Days in each month (non-leap year) for timestamp validation
var timestampDaysInMonth = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// decodeTimestamp validates and decodes ISO 8601 timestamp using position-based checks.
// Format: YYYY-MM-DDTHH:MM:SS[.fff][Z|+/-HH:MM]
// Validates actual calendar dates (e.g., rejects Feb 30, Nov 31, non-leap Feb 29)
// Returns (Timestamp, true) on success, (Timestamp{}, false) on failure
func decodeTimestamp(b []byte) (Timestamp, bool) {
	n := len(b)

	// Length check: minimum 19 (YYYY-MM-DDTHH:MM:SS), maximum 35
	if n < 19 || n > 35 {
		return Timestamp{}, false
	}

	// Fixed positions validation
	// Position 4: date separator (- or /)
	dateSep := b[4]
	if !isDateSep[dateSep] {
		return Timestamp{}, false
	}

	// Position 7: same date separator
	if b[7] != dateSep {
		return Timestamp{}, false
	}

	// Position 10: datetime separator (T or space)
	if !isDateTimeSep[b[10]] {
		return Timestamp{}, false
	}

	// Position 13, 16: time colons
	if b[13] != ':' || b[16] != ':' {
		return Timestamp{}, false
	}

	// Validate year (positions 0-3): any 4 digits
	if (b[0]-'0') > 9 || (b[1]-'0') > 9 || (b[2]-'0') > 9 || (b[3]-'0') > 9 {
		return Timestamp{}, false
	}
	year := int(b[0]-'0')*1000 + int(b[1]-'0')*100 + int(b[2]-'0')*10 + int(b[3]-'0')

	// Validate month (positions 5-6): 01-12
	if !validMonth[b[5]][b[6]] {
		return Timestamp{}, false
	}
	month := int(b[5]-'0')*10 + int(b[6]-'0')

	// Validate day (positions 8-9): format check 01-31
	if !validDay[b[8]][b[9]] {
		return Timestamp{}, false
	}
	day := int(b[8]-'0')*10 + int(b[9]-'0')

	// Calendar validation: check actual days in month
	maxDay := timestampDaysInMonth[month]
	if month == 2 && (year&3 == 0) && (year%100 != 0 || year%400 == 0) {
		maxDay = 29
	}
	if day > maxDay {
		return Timestamp{}, false
	}

	// Validate hour (positions 11-12): 00-23
	if !validHour[b[11]][b[12]] {
		return Timestamp{}, false
	}
	hour := int(b[11]-'0')*10 + int(b[12]-'0')

	// Validate minute (positions 14-15): 00-59
	if (b[14]-'0') > 5 || (b[15]-'0') > 9 {
		return Timestamp{}, false
	}
	minute := int(b[14]-'0')*10 + int(b[15]-'0')

	// Validate second (positions 17-18): 00-59
	if (b[17]-'0') > 5 || (b[18]-'0') > 9 {
		return Timestamp{}, false
	}
	second := int(b[17]-'0')*10 + int(b[18]-'0')

	// Initialize result
	result := Timestamp{
		Year:          year,
		Month:         month,
		Day:           day,
		Hour:          hour,
		Minute:        minute,
		Second:        second,
		Nanos:         0,
		OffsetSeconds: 0,
	}

	// Position 19+: optional fraction, timezone
	if n == 19 {
		return result, true // YYYY-MM-DDTHH:MM:SS
	}

	pos := 19

	// Optional fraction: .ddd...
	if b[pos] == '.' {
		pos++
		if pos >= n {
			return Timestamp{}, false // dot without digits
		}
		// Parse fractional seconds
		nanos := 0
		fracStart := pos
		for pos < n && (b[pos]-'0') <= 9 {
			if pos-fracStart < 9 {
				nanos = nanos*10 + int(b[pos]-'0')
			}
			pos++
		}
		fracLen := pos - fracStart
		if fracLen == 0 {
			return Timestamp{}, false // dot without fraction digits
		}
		// Pad to nanoseconds
		for i := fracLen; i < 9; i++ {
			nanos *= 10
		}
		result.Nanos = nanos
	}

	// Position should be at end or timezone
	if pos == n {
		return result, true // no timezone
	}

	// Timezone: Z or +/-HH:MM
	if b[pos] == 'Z' {
		if pos+1 != n {
			return Timestamp{}, false // Z must be last character
		}
		result.OffsetSeconds = 0 // Z = UTC = 0 offset
		return result, true
	}

	// Timezone offset: +/-HH:MM (6 characters remaining)
	if !isTZSign[b[pos]] {
		return Timestamp{}, false
	}

	// Need exactly 5 more characters: HH:MM
	if n-pos != 6 {
		return Timestamp{}, false
	}

	// Validate TZ hour: 00-23
	if !validHour[b[pos+1]][b[pos+2]] {
		return Timestamp{}, false
	}

	// Validate TZ colon
	if b[pos+3] != ':' {
		return Timestamp{}, false
	}

	// Validate TZ minute: 00-59
	if (b[pos+4]-'0') > 5 || (b[pos+5]-'0') > 9 {
		return Timestamp{}, false
	}

	// Compute numeric offset in seconds
	offsetSign := 1
	if b[pos] == '-' {
		offsetSign = -1
	}
	tzHour := int(b[pos+1]-'0')*10 + int(b[pos+2]-'0')
	tzMin := int(b[pos+4]-'0')*10 + int(b[pos+5]-'0')
	result.OffsetSeconds = offsetSign * (tzHour*3600 + tzMin*60)

	return result, true
}
