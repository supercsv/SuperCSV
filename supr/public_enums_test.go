// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"os"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

func TestEnumNameOnly(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "enums", "enum_labels.supr"))

	type wantRow struct {
		index int32
		name  string
	}
	want := []wantRow{
		{0, "pending"},
		{1, "active"},
		{2, "done"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(supr.EnumField)
		if !ok {
			t.Fatalf("row %d: type = %T, want EnumField", i, rows[i][0])
		}
		if got.Index() != w.index {
			t.Errorf("row %d: Index: got %d, want %d", i, got.Index(), w.index)
		}
		if got.Name() != w.name {
			t.Errorf("row %d: Name: got %q, want %q", i, got.Name(), w.name)
		}
	}
}

func TestEnumWithValues(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "enums", "enum_coded.supr"))

	type wantRow struct {
		index int32
		name  string
		value string
	}
	// 6 rows: first 3 input by name, last 3 by value token - .Value() always returns declared value.
	want := []wantRow{
		{0, "low", "0"}, {1, "medium", "1"}, {2, "high", "2"},
		{0, "low", "0"}, {1, "medium", "1"}, {2, "high", "2"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].(supr.EnumField)
		if !ok {
			t.Fatalf("row %d: type = %T, want EnumField", i, rows[i][0])
		}
		if got.Index() != w.index {
			t.Errorf("row %d: Index: got %d, want %d", i, got.Index(), w.index)
		}
		if got.Name() != w.name {
			t.Errorf("row %d: Name: got %q, want %q", i, got.Name(), w.name)
		}
		if got.Value() != w.value {
			t.Errorf("row %d: Value: got %q, want %q", i, got.Value(), w.value)
		}
	}
}

func TestEnumBothStyles(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "enums", "enum_both_styles.supr"))

	// File has 2 columns: name-only enum + enum with values
	type row struct {
		statusIdx     int32
		statusName    string
		priorityIdx   int32
		priorityName  string
		priorityValue string
	}
	want := []row{
		{0, "pending", 0, "low", "0"},   // pending, low
		{1, "active", 1, "medium", "1"}, // active, medium
		{2, "done", 2, "high", "2"},     // done, high
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		s, ok := rows[i][0].(supr.EnumField)
		if !ok {
			t.Fatalf("row %d col 0: type = %T, want EnumField", i, rows[i][0])
		}
		p, ok := rows[i][1].(supr.EnumField)
		if !ok {
			t.Fatalf("row %d col 1: type = %T, want EnumField", i, rows[i][1])
		}
		if s.Index() != w.statusIdx {
			t.Errorf("row %d status: Index: got %d, want %d", i, s.Index(), w.statusIdx)
		}
		if s.Name() != w.statusName {
			t.Errorf("row %d status: Name: got %q, want %q", i, s.Name(), w.statusName)
		}
		if p.Index() != w.priorityIdx {
			t.Errorf("row %d priority: Index: got %d, want %d", i, p.Index(), w.priorityIdx)
		}
		if p.Name() != w.priorityName {
			t.Errorf("row %d priority: Name: got %q, want %q", i, p.Name(), w.priorityName)
		}
		if p.Value() != w.priorityValue {
			t.Errorf("row %d priority: Value: got %q, want %q", i, p.Value(), w.priorityValue)
		}
	}

	// Verify header EnumSpec is accessible
	f, err := openTestdata(t, "enums", "enum_both_styles.supr")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	header := d.Header()

	// Label-only enum: IsNumeric == false
	spec0 := header.Columns[0].Type.EnumSpec()
	if spec0.IsNumeric() {
		t.Error("column 0: IsNumeric() = true, want false (label-only enum)")
	}
	if len(spec0.Values) != 3 {
		t.Fatalf("column 0: %d values, want 3", len(spec0.Values))
	}
	if spec0.Values[0].Name != "pending" {
		t.Errorf("column 0 value 0: Name = %q, want %q", spec0.Values[0].Name, "pending")
	}

	// Enum with values: IsNumeric == true
	spec1 := header.Columns[1].Type.EnumSpec()
	if !spec1.IsNumeric() {
		t.Error("column 1: IsNumeric() = false, want true (enum with values)")
	}
	if spec1.Values[0].Value != "0" || spec1.Values[0].Name != "low" {
		t.Errorf("column 1 value 0: got {Value:%q, Name:%q}, want {Value:\"0\", Name:\"low\"}",
			spec1.Values[0].Value, spec1.Values[0].Name)
	}
}

// openTestdata opens a testdata file and returns the file handle.
func openTestdata(t *testing.T, parts ...string) (*os.File, error) {
	t.Helper()
	return os.Open(testdataPath(t, parts...))
}
