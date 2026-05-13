// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package headerdef

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
)

// --- String methods (canonical literal representation) ---
//
// Each String() method produces the canonical SuperCSV literal for the type.
// These are the cross-language "raw" representation - any language wrapper
// can call String() to get a parseable literal for its own native type system.

// String returns the canonical date literal (YYYY-MM-DD).
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// String returns the canonical time literal (HH:MM:SS[.fraction]).
// Fractional seconds use minimal digits (trailing zeros trimmed).
func (t Time) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second) + formatFrac(t.Nanos)
}

// String returns the canonical timestamp literal (YYYY-MM-DDTHH:MM:SS[.fraction]Z|+/-HH:MM).
// OffsetSeconds=0 is rendered as "Z".
func (t Timestamp) String() string {
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		t.Year, t.Month, t.Day, t.Hour, t.Minute, t.Second) +
		formatFrac(t.Nanos) + formatOffset(t.OffsetSeconds)
}

// String returns the canonical datetime literal (YYYY-MM-DDTHH:MM:SS[.fraction]).
// No timezone is included.
func (d Datetime) String() string {
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		d.Year, d.Month, d.Day, d.Hour, d.Minute, d.Second) +
		formatFrac(d.Nanos)
}

// String returns the canonical datetimetz literal (YYYY-MM-DDTHH:MM:SS[.fraction]Z|+/-HH:MM).
// OffsetSeconds=0 is rendered as "Z".
func (d DatetimeTZ) String() string {
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		d.Year, d.Month, d.Day, d.Hour, d.Minute, d.Second) +
		formatFrac(d.Nanos) + formatOffset(d.OffsetSeconds)
}

// String returns the canonical ISO 8601 duration literal (P[n]DT[n]H[n]M[n[.n]]S).
// Zero durations are rendered as "PT0S".
func (d Duration) String() string {
	if d.Days == 0 && d.Hours == 0 && d.Minutes == 0 && d.Seconds == 0 && d.Nanos == 0 {
		return "PT0S"
	}
	s := "P"
	if d.Days > 0 {
		s += fmt.Sprintf("%dD", d.Days)
	}
	if d.Hours > 0 || d.Minutes > 0 || d.Seconds > 0 || d.Nanos > 0 {
		s += "T"
		if d.Hours > 0 {
			s += fmt.Sprintf("%dH", d.Hours)
		}
		if d.Minutes > 0 {
			s += fmt.Sprintf("%dM", d.Minutes)
		}
		if d.Seconds > 0 || d.Nanos > 0 {
			s += fmt.Sprintf("%d", d.Seconds) + formatFrac(d.Nanos) + "S"
		}
	}
	return s
}

// String returns the decimal literal (preserving the original representation).
func (d Decimal) String() string {
	return d.Value
}

// --- Go-native conversion methods ---
//
// These are Go-specific convenience methods. Other language wrappers should
// use String() and parse with their own native libraries instead.

// ToTime returns the date as a time.Time at midnight UTC.
func (d Date) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// ToTime returns the time as a time.Time on the zero date (0000-01-01) in UTC.
func (t Time) ToTime() time.Time {
	return time.Date(0, 1, 1, t.Hour, t.Minute, t.Second, t.Nanos, time.UTC)
}

// ToTime returns the timestamp as a time.Time.
// OffsetSeconds=0 is treated as UTC.
func (t Timestamp) ToTime() time.Time {
	loc := time.UTC
	if t.OffsetSeconds != 0 {
		loc = time.FixedZone("", t.OffsetSeconds)
	}
	return time.Date(t.Year, time.Month(t.Month), t.Day,
		t.Hour, t.Minute, t.Second, t.Nanos, loc)
}

// ToTime returns the datetime as a time.Time in UTC (no timezone information).
func (d Datetime) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day,
		d.Hour, d.Minute, d.Second, d.Nanos, time.UTC)
}

// ToTime returns the datetimetz as a time.Time with the specified offset.
// OffsetSeconds=0 is treated as UTC.
func (d DatetimeTZ) ToTime() time.Time {
	loc := time.UTC
	if d.OffsetSeconds != 0 {
		loc = time.FixedZone("", d.OffsetSeconds)
	}
	return time.Date(d.Year, time.Month(d.Month), d.Day,
		d.Hour, d.Minute, d.Second, d.Nanos, loc)
}

// ToDuration returns the duration as a time.Duration.
// Days are treated as exactly 24 hours (no DST adjustment).
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d.Days)*24*time.Hour +
		time.Duration(d.Hours)*time.Hour +
		time.Duration(d.Minutes)*time.Minute +
		time.Duration(d.Seconds)*time.Second +
		time.Duration(d.Nanos)
}

// ToFloat64 returns the decimal value as a float64.
// Returns false if the string cannot be parsed (should not occur for validated data).
func (d Decimal) ToFloat64() (float64, bool) {
	f, err := strconv.ParseFloat(d.Value, 64)
	return f, err == nil
}

// ToBigFloat returns the decimal value as a *big.Float for arbitrary precision.
// Returns false if the string cannot be parsed (should not occur for validated data).
func (d Decimal) ToBigFloat() (*big.Float, bool) {
	return new(big.Float).SetString(d.Value)
}

// --- internal helpers ---

// formatFrac formats nanoseconds as a fractional seconds suffix ".NNN"
// with trailing zeros trimmed. Returns "" if nanos is 0.
func formatFrac(nanos int) string {
	if nanos == 0 {
		return ""
	}
	s := fmt.Sprintf(".%09d", nanos)
	i := len(s) - 1
	for i > 1 && s[i] == '0' {
		i--
	}
	return s[:i+1]
}

// formatOffset formats a timezone offset in seconds as "Z" or "+/-HH:MM".
func formatOffset(offsetSeconds int) string {
	if offsetSeconds == 0 {
		return "Z"
	}
	sign := "+"
	off := offsetSeconds
	if off < 0 {
		sign = "-"
		off = -off
	}
	h := off / 3600
	m := (off % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}
