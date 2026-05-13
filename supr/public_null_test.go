// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"testing"

	"github.com/supercsv/supercsv/supr"
)

func TestNullScalars(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "null", "null_scalars.supr"))

	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	// Row 0: all null
	for col := 0; col < 4; col++ {
		if rows[0][col] != nil {
			t.Errorf("row 0 col %d: got %v (%T), want nil", col, rows[0][col], rows[0][col])
		}
	}

	// Row 1: all present
	if v, ok := rows[1][0].(int64); !ok || v != 42 {
		t.Errorf("row 1 col 0: got %v, want 42", rows[1][0])
	}
	if v, ok := rows[1][1].(string); !ok || v != "hello" {
		t.Errorf("row 1 col 1: got %v, want hello", rows[1][1])
	}
	if v, ok := rows[1][2].(bool); !ok || v != true {
		t.Errorf("row 1 col 2: got %v, want true", rows[1][2])
	}
	if v, ok := rows[1][3].(supr.DateValue); !ok || v.Year != 2024 {
		t.Errorf("row 1 col 3: got %v, want 2024-01-15", rows[1][3])
	}

	// Row 2: all null again
	for col := 0; col < 4; col++ {
		if rows[2][col] != nil {
			t.Errorf("row 2 col %d: got %v (%T), want nil", col, rows[2][col], rows[2][col])
		}
	}
}

func TestNullContainers(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "null", "null_containers.supr"))

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// Row 0: [1,_,3] - list with null element
	list, ok := rows[0][0].([]any)
	if !ok {
		t.Fatalf("row 0: type = %T, want []any", rows[0][0])
	}
	if len(list) != 3 {
		t.Fatalf("row 0: len = %d, want 3", len(list))
	}
	if v, ok := list[0].(int64); !ok || v != 1 {
		t.Errorf("row 0 elem 0: got %v, want 1", list[0])
	}
	if list[1] != nil {
		t.Errorf("row 0 elem 1: got %v, want nil", list[1])
	}
	if v, ok := list[2].(int64); !ok || v != 3 {
		t.Errorf("row 0 elem 2: got %v, want 3", list[2])
	}

	// Row 1: _ - null container field
	if rows[1][0] != nil {
		t.Errorf("row 1: got %v (%T), want nil", rows[1][0], rows[1][0])
	}
}

func TestNullEnum(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "null", "null_enum.supr"))

	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	// Row 0: pending -> index 0, name "pending"
	if v, ok := rows[0][0].(supr.EnumField); !ok {
		t.Errorf("row 0: got %v (%T), want EnumField", rows[0][0], rows[0][0])
	} else {
		if v.Index() != 0 {
			t.Errorf("row 0: Index: got %d, want 0", v.Index())
		}
		if v.Name() != "pending" {
			t.Errorf("row 0: Name: got %q, want %q", v.Name(), "pending")
		}
	}

	// Row 1: null
	if rows[1][0] != nil {
		t.Errorf("row 1: got %v (%T), want nil", rows[1][0], rows[1][0])
	}

	// Row 2: done -> index 2, name "done"
	if v, ok := rows[2][0].(supr.EnumField); !ok {
		t.Errorf("row 2: got %v (%T), want EnumField", rows[2][0], rows[2][0])
	} else {
		if v.Index() != 2 {
			t.Errorf("row 2: Index: got %d, want 2", v.Index())
		}
		if v.Name() != "done" {
			t.Errorf("row 2: Name: got %q, want %q", v.Name(), "done")
		}
	}
}
