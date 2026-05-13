// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// testdataPath returns the absolute path to a public testdata file.
func testdataPath(t *testing.T, parts ...string) string {
	t.Helper()
	elems := append([]string{"testdata", "public"}, parts...)
	return filepath.Join(elems...)
}

// decodeTestdataFile is a helper that opens and decodes a .supr file, returning all rows.
func decodeTestdataFile(t *testing.T, path string) [][]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	var rows [][]any
	for d.Next() {
		// Copy row since buffer is reused
		src := d.Row()
		row := make([]any, len(src))
		copy(row, src)
		rows = append(rows, row)
	}
	if err := d.Err(); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return rows
}

// readTestdataFile reads the raw content of a public testdata file as a string.
func readTestdataFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := testdataPath(t, parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestScalarInt(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "int.supr"))

	want := []int64{0, 42, -999, 9223372036854775807, -9223372036854775808}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(int64)
		if !ok {
			t.Fatalf("row %d: type = %T, want int64", i, rows[i][0])
		}
		if got != w {
			t.Errorf("row %d: got %d, want %d", i, got, w)
		}
	}
}

func TestScalarFloat(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "float.supr"))

	// Rows: 0.0, 3.14159, -2.71828, 1.23e10, 1.23e-10, nan, inf, -inf
	if len(rows) != 8 {
		t.Fatalf("got %d rows, want 8", len(rows))
	}

	tests := []struct {
		row      int
		want     float64
		isNaN    bool
		isPosInf bool
		isNegInf bool
	}{
		{0, 0.0, false, false, false},
		{1, 3.14159, false, false, false},
		{2, -2.71828, false, false, false},
		{3, 1.23e10, false, false, false},
		{4, 1.23e-10, false, false, false},
		{5, 0, true, false, false},
		{6, 0, false, true, false},
		{7, 0, false, false, true},
	}

	for _, tt := range tests {
		got, ok := rows[tt.row][0].(float64)
		if !ok {
			t.Fatalf("row %d: type = %T, want float64", tt.row, rows[tt.row][0])
		}
		switch {
		case tt.isNaN:
			if !math.IsNaN(got) {
				t.Errorf("row %d: got %v, want NaN", tt.row, got)
			}
		case tt.isPosInf:
			if !math.IsInf(got, 1) {
				t.Errorf("row %d: got %v, want +Inf", tt.row, got)
			}
		case tt.isNegInf:
			if !math.IsInf(got, -1) {
				t.Errorf("row %d: got %v, want -Inf", tt.row, got)
			}
		default:
			if got != tt.want {
				t.Errorf("row %d: got %v, want %v", tt.row, got, tt.want)
			}
		}
	}
}

func TestScalarDecimal(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "decimal.supr"))

	want := []string{"0.00", "123.45", "-999.99", "3.141592653589793"}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(supr.DecimalValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.DecimalValue", i, rows[i][0])
		}
		if got.Value != w {
			t.Errorf("row %d: got %q, want %q", i, got.Value, w)
		}
	}
}

func TestScalarBool(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "bool.supr"))

	// Rows: 1, 0, true, false
	want := []bool{true, false, true, false}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(bool)
		if !ok {
			t.Fatalf("row %d: type = %T, want bool", i, rows[i][0])
		}
		if got != w {
			t.Errorf("row %d: got %v, want %v", i, got, w)
		}
	}
}

func TestScalarString(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "string.supr"))

	// Rows: Hello, "Hello World", "Hello,World", "Say ""Hi""", "Hello 世界", ""
	want := []string{"Hello", "Hello World", "Hello,World", "Say \"Hi\"", "Hello 世界", ""}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(string)
		if !ok {
			t.Fatalf("row %d: type = %T, want string", i, rows[i][0])
		}
		if got != w {
			t.Errorf("row %d: got %q, want %q", i, got, w)
		}
	}
}

func TestScalarBytesHex(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "bytes_hex.supr"))

	// Rows: 48656C6C6F (Hello), 68656c6c6f (hello), DEADBEEF
	want := [][]byte{
		[]byte("Hello"),
		[]byte("hello"),
		{0xDE, 0xAD, 0xBE, 0xEF},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]byte)
		if !ok {
			t.Fatalf("row %d: type = %T, want []byte", i, rows[i][0])
		}
		if string(got) != string(w) {
			t.Errorf("row %d: got %x, want %x", i, got, w)
		}
	}
}

func TestScalarBytesB64(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "bytes_b64.supr"))

	// Rows: SGVsbG8= (Hello), V29ybGQ= (World)
	want := [][]byte{
		[]byte("Hello"),
		[]byte("World"),
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]byte)
		if !ok {
			t.Fatalf("row %d: type = %T, want []byte", i, rows[i][0])
		}
		if string(got) != string(w) {
			t.Errorf("row %d: got %x, want %x", i, got, w)
		}
	}
}

func TestScalarDate(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "date.supr"))

	tests := []struct {
		year, month, day int
	}{
		{2024, 1, 15},
		{2024, 2, 29},
		{2000, 12, 31},
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.DateValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.DateValue", i, rows[i][0])
		}
		if got.Year != tt.year || got.Month != tt.month || got.Day != tt.day {
			t.Errorf("row %d: got %d-%02d-%02d, want %d-%02d-%02d",
				i, got.Year, got.Month, got.Day, tt.year, tt.month, tt.day)
		}
	}
}

func TestScalarTime(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "time.supr"))

	tests := []struct {
		hour, minute, second, nanos int
	}{
		{14, 30, 0, 0},
		{0, 0, 0, 0},
		{23, 59, 59, 123456789},
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.TimeValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.TimeValue", i, rows[i][0])
		}
		if got.Hour != tt.hour || got.Minute != tt.minute || got.Second != tt.second || got.Nanos != tt.nanos {
			t.Errorf("row %d: got %02d:%02d:%02d.%09d, want %02d:%02d:%02d.%09d",
				i, got.Hour, got.Minute, got.Second, got.Nanos,
				tt.hour, tt.minute, tt.second, tt.nanos)
		}
	}
}

func TestScalarTimestamp(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "timestamp.supr"))

	tests := []struct {
		year, month, day, hour, minute, second, nanos, offset int
	}{
		{2024, 1, 15, 14, 30, 0, 0, 0},                // Z
		{2024, 6, 15, 9, 0, 0, 0, 19800},              // +05:30
		{2024, 12, 31, 23, 59, 59, 999000000, -28800}, // -08:00
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.TimestampValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.TimestampValue", i, rows[i][0])
		}
		if got.Year != tt.year || got.Month != tt.month || got.Day != tt.day ||
			got.Hour != tt.hour || got.Minute != tt.minute || got.Second != tt.second ||
			got.Nanos != tt.nanos || got.OffsetSeconds != tt.offset {
			t.Errorf("row %d: got {%d-%02d-%02d %02d:%02d:%02d.%09d offset=%d}, want {%d-%02d-%02d %02d:%02d:%02d.%09d offset=%d}",
				i, got.Year, got.Month, got.Day, got.Hour, got.Minute, got.Second, got.Nanos, got.OffsetSeconds,
				tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second, tt.nanos, tt.offset)
		}
	}
}

func TestScalarDatetime(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "datetime.supr"))

	tests := []struct {
		year, month, day, hour, minute, second, nanos int
	}{
		{2024, 1, 15, 14, 30, 0, 0},
		{2024, 6, 15, 9, 0, 0, 123000000},
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.DatetimeValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.DatetimeValue", i, rows[i][0])
		}
		if got.Year != tt.year || got.Month != tt.month || got.Day != tt.day ||
			got.Hour != tt.hour || got.Minute != tt.minute || got.Second != tt.second ||
			got.Nanos != tt.nanos {
			t.Errorf("row %d: got {%d-%02d-%02d %02d:%02d:%02d.%09d}, want {%d-%02d-%02d %02d:%02d:%02d.%09d}",
				i, got.Year, got.Month, got.Day, got.Hour, got.Minute, got.Second, got.Nanos,
				tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second, tt.nanos)
		}
	}
}

func TestScalarDatetimeTZ(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "datetimetz.supr"))

	tests := []struct {
		year, month, day, hour, minute, second, nanos, offset int
	}{
		{2024, 1, 15, 14, 30, 0, 0, 0},               // Z
		{2024, 6, 15, 9, 0, 0, 0, -18000},            // -05:00
		{2024, 12, 31, 23, 59, 59, 999000000, 32400}, // +09:00
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.DatetimeTZValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.DatetimeTZValue", i, rows[i][0])
		}
		if got.Year != tt.year || got.Month != tt.month || got.Day != tt.day ||
			got.Hour != tt.hour || got.Minute != tt.minute || got.Second != tt.second ||
			got.Nanos != tt.nanos || got.OffsetSeconds != tt.offset {
			t.Errorf("row %d: got {%d-%02d-%02d %02d:%02d:%02d.%09d offset=%d}, want {%d-%02d-%02d %02d:%02d:%02d.%09d offset=%d}",
				i, got.Year, got.Month, got.Day, got.Hour, got.Minute, got.Second, got.Nanos, got.OffsetSeconds,
				tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second, tt.nanos, tt.offset)
		}
	}
}

func TestScalarDuration(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "duration.supr"))

	tests := []struct {
		days, hours, minutes, seconds, nanos int
	}{
		{0, 0, 0, 30, 0},        // PT30S
		{0, 2, 0, 0, 0},         // PT2H
		{5, 0, 0, 0, 0},         // P5D
		{1, 2, 30, 0, 0},        // P1DT2H30M
		{0, 0, 0, 1, 500000000}, // PT1.5S
	}
	if len(rows) != len(tests) {
		t.Fatalf("got %d rows, want %d", len(rows), len(tests))
	}
	for i, tt := range tests {
		got, ok := rows[i][0].(supr.DurationValue)
		if !ok {
			t.Fatalf("row %d: type = %T, want supr.DurationValue", i, rows[i][0])
		}
		if got.Days != tt.days || got.Hours != tt.hours || got.Minutes != tt.minutes ||
			got.Seconds != tt.seconds || got.Nanos != tt.nanos {
			t.Errorf("row %d: got P%dDT%dH%dM%dS.%09d, want P%dDT%dH%dM%dS.%09d",
				i, got.Days, got.Hours, got.Minutes, got.Seconds, got.Nanos,
				tt.days, tt.hours, tt.minutes, tt.seconds, tt.nanos)
		}
	}
}

func TestScalarTimezone(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "timezone.supr"))

	want := []string{"Z", "+05:30", "-08:00"}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(string)
		if !ok {
			t.Fatalf("row %d: type = %T, want string", i, rows[i][0])
		}
		if got != w {
			t.Errorf("row %d: got %q, want %q", i, got, w)
		}
	}
}

func TestScalarUUID(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "scalars", "uuid.supr"))

	want := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(string)
		if !ok {
			t.Fatalf("row %d: type = %T, want string", i, rows[i][0])
		}
		if got != w {
			t.Errorf("row %d: got %q, want %q", i, got, w)
		}
	}
}
