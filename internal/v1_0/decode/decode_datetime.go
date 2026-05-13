// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeDatetime validates and decodes ISO 8601 datetime WITHOUT timezone.
// decodeDatetime validates and decodes ISO 8601 datetime WITHOUT timezone.
// Format: YYYY-MM-DDTHH:MM:SS[.fff]
// NO timezone allowed (datetime type forbids Z and +/-HH:MM)
// Validates actual calendar dates (e.g., rejects Feb 30, Nov 31, non-leap Feb 29)
// Returns (Datetime, true) on success, (Datetime{}, false) on failure
func decodeDatetime(b []byte) (Datetime, bool) {
	n := len(b)

	// Length check: minimum 19 (YYYY-MM-DDTHH:MM:SS), maximum 29 (with .nnnnnnnnn)
	// No timezone allowed for datetime type
	if n < 19 || n > 29 {
		return Datetime{}, false
	}

	// Fixed positions validation
	// Position 4: date separator (- or /)
	dateSep := b[4]
	if !isDateSep[dateSep] {
		return Datetime{}, false
	}

	// Position 7: same date separator
	if b[7] != dateSep {
		return Datetime{}, false
	}

	// Position 10: datetime separator (T or space)
	if !isDateTimeSep[b[10]] {
		return Datetime{}, false
	}

	// Position 13, 16: time colons
	if b[13] != ':' || b[16] != ':' {
		return Datetime{}, false
	}

	// Validate year (positions 0-3): any 4 digits
	if (b[0]-'0') > 9 || (b[1]-'0') > 9 || (b[2]-'0') > 9 || (b[3]-'0') > 9 {
		return Datetime{}, false
	}
	year := int(b[0]-'0')*1000 + int(b[1]-'0')*100 + int(b[2]-'0')*10 + int(b[3]-'0')

	// Validate month (positions 5-6): 01-12
	if !validMonth[b[5]][b[6]] {
		return Datetime{}, false
	}
	month := int(b[5]-'0')*10 + int(b[6]-'0')

	// Validate day (positions 8-9): format check 01-31
	if !validDay[b[8]][b[9]] {
		return Datetime{}, false
	}
	day := int(b[8]-'0')*10 + int(b[9]-'0')

	// Calendar validation: check actual days in month
	maxDay := timestampDaysInMonth[month]
	if month == 2 && (year&3 == 0) && (year%100 != 0 || year%400 == 0) {
		maxDay = 29
	}
	if day > maxDay {
		return Datetime{}, false
	}

	// Validate hour (positions 11-12): 00-23
	if !validHour[b[11]][b[12]] {
		return Datetime{}, false
	}
	hour := int(b[11]-'0')*10 + int(b[12]-'0')

	// Validate minute (positions 14-15): 00-59
	if (b[14]-'0') > 5 || (b[15]-'0') > 9 {
		return Datetime{}, false
	}
	minute := int(b[14]-'0')*10 + int(b[15]-'0')

	// Validate second (positions 17-18): 00-59
	if (b[17]-'0') > 5 || (b[18]-'0') > 9 {
		return Datetime{}, false
	}
	second := int(b[17]-'0')*10 + int(b[18]-'0')

	// Initialize result
	result := Datetime{
		Year:   year,
		Month:  month,
		Day:    day,
		Hour:   hour,
		Minute: minute,
		Second: second,
		Nanos:  0,
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
			return Datetime{}, false // dot without digits
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
			return Datetime{}, false // dot without fraction digits
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

	// Datetime type forbids timezone - any characters after fraction are invalid
	return Datetime{}, false
}
