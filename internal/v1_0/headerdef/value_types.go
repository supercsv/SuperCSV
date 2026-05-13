// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package headerdef

// Date represents an ISO-8601 calendar date (YYYY-MM-DD or YYYY/MM/DD)
// Spec: year=4 digits, month=01-12, day validated for month with leap year rules
type Date struct {
	Year  int // 4-digit year (e.g., 2025)
	Month int // 1-12
	Day   int // 1-31, validated against month/leap year
}

// Time represents an ISO-8601 local time (HH:MM:SS[.fffffffff])
// Spec: fractional seconds 0-9 digits, NO timezone
type Time struct {
	Hour   int // 0-23
	Minute int // 0-59
	Second int // 0-59
	Nanos  int // 0-999999999 (fractional seconds as nanoseconds)
}

// Timestamp represents an ISO-8601 timestamp with optional timezone offset
// Spec: Date + Time, separator is space or 'T', timezone is Z or +/-HH:MM (no seconds)
type Timestamp struct {
	Year          int // 4-digit year
	Month         int // 1-12
	Day           int // 1-31
	Hour          int // 0-23
	Minute        int // 0-59
	Second        int // 0-59
	Nanos         int // 0-999999999
	OffsetSeconds int // Z=0, +05:00=+18000, -07:30=-27000, no timezone=0
}

// Datetime represents an ISO-8601 datetime WITHOUT timezone
// Spec: Date + Time, separator is space or 'T', NO timezone allowed
type Datetime struct {
	Year   int // 4-digit year
	Month  int // 1-12
	Day    int // 1-31
	Hour   int // 0-23
	Minute int // 0-59
	Second int // 0-59
	Nanos  int // 0-999999999
}

// DatetimeTZ represents an ISO-8601 datetime WITH required timezone
// Spec: Date + Time, separator is space or 'T', timezone is Z or +/-HH:MM (REQUIRED)
type DatetimeTZ struct {
	Year          int // 4-digit year
	Month         int // 1-12
	Day           int // 1-31
	Hour          int // 0-23
	Minute        int // 0-59
	Second        int // 0-59
	Nanos         int // 0-999999999
	OffsetSeconds int // Z=0, +05:00=+18000, -07:30=-27000 (REQUIRED, never 0 if timezone present)
}

// Duration represents an ISO-8601 duration (P[n]DT[n]H[n]M[n]S)
// Spec v1.0: Days, Hours, Minutes, Seconds ONLY (NO years, months, weeks)
// Spec: Fractional seconds allowed ONLY on seconds component
type Duration struct {
	Days    int // 0+ (from P[n]D)
	Hours   int // 0+ (from T[n]H)
	Minutes int // 0+ (from T[n]M)
	Seconds int // 0+ (from T[n]S integer part)
	Nanos   int // 0-999999999 (from T[n.f]S fractional part)
}

// Decimal represents arbitrary-precision decimal number
// Spec: max 128 chars total, max 64 fractional digits, no exponent notation
// Spec: optional leading minus, integer part (no leading zeros except "0"), optional fractional
// Spec: negative zero NOT allowed, trailing zeros in fractional part ARE allowed
type Decimal struct {
	// For Phase 1: store as validated string
	// Format: [-]digits[.digits]
	Value string

	// Future enhancement: parse into coefficient/exponent for arithmetic
	// e.g., 12.345 -> coefficient=12345, exponent=-3
}

// EnumField is the decoded representation of an enum field value.
// It holds a pointer into the column's EnumSpec.Values slice - zero allocation.
type EnumField struct {
	item *EnumValue
}

func (e EnumField) Name() string  { return e.item.Name }
func (e EnumField) Value() string { return e.item.Value }
func (e EnumField) Index() int32  { return e.item.Index }

// NewEnumField constructs an EnumField from a pointer into an EnumSpec.Values slice.
// The pointer must point into a live EnumSpec - it is not copied.
// For internal use by the decoder only; external callers receive EnumField via type assertion.
// Zero allocation: inlined by the compiler, no heap allocation occurs.
func NewEnumField(v *EnumValue) EnumField {
	return EnumField{item: v}
}
