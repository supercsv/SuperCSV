// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/supercsv/supercsv/supr"
)

// TestValueConversions_Date demonstrates accessing Date fields and converting to time.Time.
func TestValueConversions_Date(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "date.supr"))
	// First row: 2024-01-15
	date := rows[0][0].(supr.DateValue)

	// Raw struct fields
	if date.Year != 2024 || date.Month != 1 || date.Day != 15 {
		t.Fatalf("date fields = %d-%d-%d, want 2024-1-15", date.Year, date.Month, date.Day)
	}

	// String() - canonical literal for cross-language use
	if s := date.String(); s != "2024-01-15" {
		t.Errorf("date.String() = %q, want %q", s, "2024-01-15")
	}

	// ToTime() - Go-native conversion
	got := date.ToTime()
	want := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("date.ToTime() = %v, want %v", got, want)
	}
}

// TestValueConversions_Time demonstrates accessing Time fields and converting to time.Time.
func TestValueConversions_Time(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "time.supr"))
	// Third row: 23:59:59.123456789
	tv := rows[2][0].(supr.TimeValue)

	// Raw struct fields
	if tv.Hour != 23 || tv.Minute != 59 || tv.Second != 59 || tv.Nanos != 123456789 {
		t.Fatalf("time fields = %d:%d:%d.%d, want 23:59:59.123456789",
			tv.Hour, tv.Minute, tv.Second, tv.Nanos)
	}

	// String() - canonical literal with fractional seconds
	if s := tv.String(); s != "23:59:59.123456789" {
		t.Errorf("time.String() = %q, want %q", s, "23:59:59.123456789")
	}

	// ToTime() - Go-native conversion
	got := tv.ToTime()
	if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 || got.Nanosecond() != 123456789 {
		t.Errorf("time.ToTime() = %v, want 23:59:59.123456789", got)
	}
}

// TestValueConversions_Timestamp demonstrates Timestamp with timezone offset.
func TestValueConversions_Timestamp(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "timestamp.supr"))

	// First row: 2024-01-15T14:30:00Z
	ts := rows[0][0].(supr.TimestampValue)
	if s := ts.String(); s != "2024-01-15T14:30:00Z" {
		t.Errorf("ts.String() = %q, want %q", s, "2024-01-15T14:30:00Z")
	}
	got := ts.ToTime()
	want := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ts.ToTime() = %v, want %v", got, want)
	}

	// Second row: 2024-06-15T09:00:00+05:30
	ts2 := rows[1][0].(supr.TimestampValue)
	if s := ts2.String(); s != "2024-06-15T09:00:00+05:30" {
		t.Errorf("ts2.String() = %q, want %q", s, "2024-06-15T09:00:00+05:30")
	}
	got2 := ts2.ToTime()
	_, offset := got2.Zone()
	if offset != 5*3600+30*60 {
		t.Errorf("ts2.ToTime() offset = %d, want %d", offset, 5*3600+30*60)
	}
}

// TestValueConversions_Datetime demonstrates Datetime (no timezone).
func TestValueConversions_Datetime(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "datetime.supr"))
	// First row: 2024-01-15T14:30:00
	dt := rows[0][0].(supr.DatetimeValue)

	if s := dt.String(); s != "2024-01-15T14:30:00" {
		t.Errorf("dt.String() = %q, want %q", s, "2024-01-15T14:30:00")
	}
	got := dt.ToTime()
	want := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("dt.ToTime() = %v, want %v", got, want)
	}
}

// TestValueConversions_DatetimeTZ demonstrates DatetimeTZ with required timezone.
func TestValueConversions_DatetimeTZ(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "datetimetz.supr"))

	// First row: 2024-01-15T14:30:00Z
	dtz := rows[0][0].(supr.DatetimeTZValue)
	if s := dtz.String(); s != "2024-01-15T14:30:00Z" {
		t.Errorf("dtz.String() = %q, want %q", s, "2024-01-15T14:30:00Z")
	}

	// Second row: 2024-06-15T09:00:00-05:00
	dtz2 := rows[1][0].(supr.DatetimeTZValue)
	if s := dtz2.String(); s != "2024-06-15T09:00:00-05:00" {
		t.Errorf("dtz2.String() = %q, want %q", s, "2024-06-15T09:00:00-05:00")
	}
	got := dtz2.ToTime()
	_, offset := got.Zone()
	if offset != -5*3600 {
		t.Errorf("dtz2.ToTime() offset = %d, want %d", offset, -5*3600)
	}
}

// TestValueConversions_Duration demonstrates Duration conversion to time.Duration.
func TestValueConversions_Duration(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "duration.supr"))

	// Row 0: PT30S
	d0 := rows[0][0].(supr.DurationValue)
	if s := d0.String(); s != "PT30S" {
		t.Errorf("d0.String() = %q, want %q", s, "PT30S")
	}
	if got := d0.ToDuration(); got != 30*time.Second {
		t.Errorf("d0.ToDuration() = %v, want %v", got, 30*time.Second)
	}

	// Row 3: P1DT2H30M
	d3 := rows[3][0].(supr.DurationValue)
	if s := d3.String(); s != "P1DT2H30M" {
		t.Errorf("d3.String() = %q, want %q", s, "P1DT2H30M")
	}
	want := 24*time.Hour + 2*time.Hour + 30*time.Minute
	if got := d3.ToDuration(); got != want {
		t.Errorf("d3.ToDuration() = %v, want %v", got, want)
	}

	// Row 4: PT1.5S (fractional seconds)
	d4 := rows[4][0].(supr.DurationValue)
	if s := d4.String(); s != "PT1.5S" {
		t.Errorf("d4.String() = %q, want %q", s, "PT1.5S")
	}
	if got := d4.ToDuration(); got != time.Second+500*time.Millisecond {
		t.Errorf("d4.ToDuration() = %v, want %v", got, time.Second+500*time.Millisecond)
	}
}

// TestValueConversions_Decimal demonstrates Decimal conversion to float64 and big.Float.
func TestValueConversions_Decimal(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "decimal.supr"))

	// Row 1: 123.45
	dec := rows[1][0].(supr.DecimalValue)

	// Raw field
	if dec.Value != "123.45" {
		t.Fatalf("dec.Value = %q, want %q", dec.Value, "123.45")
	}

	// String() - returns the literal
	if s := dec.String(); s != "123.45" {
		t.Errorf("dec.String() = %q, want %q", s, "123.45")
	}

	// ToFloat64() - Go float conversion
	f, ok := dec.ToFloat64()
	if !ok {
		t.Fatal("dec.ToFloat64() ok = false")
	}
	if f != 123.45 {
		t.Errorf("dec.ToFloat64() = %v, want 123.45", f)
	}

	// ToBigFloat() - arbitrary precision
	bf, ok := dec.ToBigFloat()
	if !ok {
		t.Fatal("dec.ToBigFloat() ok = false")
	}
	wantBF, _ := new(big.Float).SetString("123.45")
	if bf.Cmp(wantBF) != 0 {
		t.Errorf("dec.ToBigFloat() = %v, want %v", bf, wantBF)
	}
}
