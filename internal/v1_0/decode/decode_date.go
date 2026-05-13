// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// Days in each month (non-leap year) - index 0 unused, months are 1-12
var dateDaysInMonth = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// decodeDate validates and decodes ISO 8601 dates: YYYY-MM-DD or YYYY/MM/DD
// Validates structure, ranges, and calendar rules (leap years, month-day limits)
// Returns (Date{}, true) on success, (Date{}, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeDate(value []byte) (Date, bool) {
	// Must be exactly 10 characters: YYYY-MM-DD or YYYY/MM/DD
	if len(value) != 10 {
		return Date{}, false
	}

	// Check structure: YYYY-MM-DD or YYYY/MM/DD
	if !isDateSep[value[4]] || value[7] != value[4] {
		return Date{}, false
	}

	// Validate year digits (YYYY)
	if !isDigit[value[0]] || !isDigit[value[1]] || !isDigit[value[2]] || !isDigit[value[3]] {
		return Date{}, false
	}

	// Validate month (01-12) and day (01-31) using 2-char lookups
	if !validMonth[value[5]][value[6]] || !validDay[value[8]][value[9]] {
		return Date{}, false
	}

	// Extract year, month, day for calendar validation
	year := int(value[0]-'0')*1000 + int(value[1]-'0')*100 + int(value[2]-'0')*10 + int(value[3]-'0')
	month := int(value[5]-'0')*10 + int(value[6]-'0')
	day := int(value[8]-'0')*10 + int(value[9]-'0')

	// Validate day is within correct range for month
	maxDay := dateDaysInMonth[month]
	if month == 2 && (year&3 == 0) && (year%100 != 0 || year%400 == 0) {
		maxDay = 29
	}

	if day > maxDay {
		return Date{}, false
	}

	return Date{Year: year, Month: month, Day: day}, true
}
