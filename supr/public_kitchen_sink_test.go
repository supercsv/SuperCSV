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

// TestKitchenSinkValidate validates the everything.supr file.
func TestKitchenSinkValidate(t *testing.T) {
	path := testdataPath(t, "kitchen-sink", "everything.supr")

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, errs, fatalErr := supr.ValidateFile(f)
	if fatalErr != nil {
		t.Fatalf("ValidateFile fatal: %v", fatalErr)
	}
	if len(errs) > 0 {
		t.Errorf("ValidateFile errors: %v", errs)
	}
}

// TestKitchenSinkDecode decodes everything.supr and spot-checks values.
func TestKitchenSinkDecode(t *testing.T) {
	path := testdataPath(t, "kitchen-sink", "everything.supr")

	file, err := supr.DecodeFile(path)
	if err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}

	// Verify header
	if len(file.Header.Columns) != 10 {
		t.Fatalf("columns = %d, want 10", len(file.Header.Columns))
	}

	if len(file.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(file.Rows))
	}

	// Row 0: full data
	row := file.Rows[0]
	if got, ok := row[0].(int64); !ok || got != 1 {
		t.Errorf("row 0 id = %v (%T), want 1", row[0], row[0])
	}
	if got, ok := row[1].(string); !ok || got != "Alice" {
		t.Errorf("row 0 name = %v, want Alice", row[1])
	}
	if got, ok := row[3].(bool); !ok || got != true {
		t.Errorf("row 0 active = %v, want true", row[3])
	}

	// Verify tags list
	tags, ok := row[6].([]any)
	if !ok {
		t.Fatalf("row 0 tags type = %T, want []any", row[6])
	}
	if len(tags) != 2 {
		t.Fatalf("row 0 tags len = %d, want 2", len(tags))
	}

	// Verify enum (label-only) - grade A = index 0, name "A"
	if got, ok := row[7].(supr.EnumField); !ok {
		t.Errorf("row 0 grade = %v (%T), want EnumField", row[7], row[7])
	} else {
		if got.Index() != 0 {
			t.Errorf("row 0 grade: Index: got %d, want 0", got.Index())
		}
		if got.Name() != "A" {
			t.Errorf("row 0 grade: Name: got %q, want %q", got.Name(), "A")
		}
	}

	// Row 2: nulls
	row2 := file.Rows[2]
	if row2[1] != nil {
		t.Errorf("row 2 name = %v, want nil", row2[1])
	}
	if row2[4] != nil {
		t.Errorf("row 2 balance = %v, want nil", row2[4])
	}

	// Row 2: empty list - should be empty []any
	emptyTags, ok := row2[6].([]any)
	if !ok {
		t.Fatalf("row 2 tags type = %T, want []any", row2[6])
	}
	if len(emptyTags) != 0 {
		t.Errorf("row 2 tags len = %d, want 0", len(emptyTags))
	}

	// Row 2: null enum
	if row2[7] != nil {
		t.Errorf("row 2 grade = %v, want nil", row2[7])
	}
}

// TestKitchenSinkStreaming verifies streaming decode works with everything.supr.
func TestKitchenSinkStreaming(t *testing.T) {
	f, err := os.Open(testdataPath(t, "kitchen-sink", "everything.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	hdr := d.Header()
	if hdr == nil {
		t.Fatal("Header() = nil")
	}

	count := 0
	for d.Next() {
		row := d.Row()
		if len(row) != 10 {
			t.Errorf("row %d: len = %d, want 10", count, len(row))
		}
		count++
	}
	if err := d.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}
