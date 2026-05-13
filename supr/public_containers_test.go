// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"testing"
)

func TestContainerList(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "list.supr"))

	// [1,2,3], [], [42]
	want := [][]int64{{1, 2, 3}, {}, {42}}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]any)
		if !ok {
			t.Fatalf("row %d: type = %T, want []any", i, rows[i][0])
		}
		if len(got) != len(w) {
			t.Errorf("row %d: len = %d, want %d", i, len(got), len(w))
			continue
		}
		for j, wv := range w {
			if v, ok := got[j].(int64); !ok || v != wv {
				t.Errorf("row %d elem %d: got %v (%T), want %d", i, j, got[j], got[j], wv)
			}
		}
	}
}

func TestContainerListFixed(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "list_fixed.supr"))

	want := [][]string{{"A", "B", "C"}, {"X", "Y", "Z"}}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]any)
		if !ok {
			t.Fatalf("row %d: type = %T, want []any", i, rows[i][0])
		}
		if len(got) != len(w) {
			t.Errorf("row %d: len = %d, want %d", i, len(got), len(w))
			continue
		}
		for j, wv := range w {
			if v, ok := got[j].(string); !ok || v != wv {
				t.Errorf("row %d elem %d: got %v (%T), want %q", i, j, got[j], got[j], wv)
			}
		}
	}
}

func TestContainerArray1D(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "array_1d.supr"))

	want := [][]int64{{1, 2, 3}, {10, 20, 30}}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]any)
		if !ok {
			t.Fatalf("row %d: type = %T, want []any", i, rows[i][0])
		}
		if len(got) != len(w) {
			t.Errorf("row %d: len = %d, want %d", i, len(got), len(w))
			continue
		}
		for j, wv := range w {
			if v, ok := got[j].(int64); !ok || v != wv {
				t.Errorf("row %d elem %d: got %v (%T), want %d", i, j, got[j], got[j], wv)
			}
		}
	}
}

func TestContainerArray2D(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "array_2d.supr"))

	want := [][][]int64{
		{{1, 2, 3}, {4, 5, 6}},
		{{10, 20, 30}, {40, 50, 60}},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([][]any)
		if !ok {
			t.Fatalf("row %d: type = %T, want [][]any", i, rows[i][0])
		}
		if len(got) != len(w) {
			t.Errorf("row %d: rows = %d, want %d", i, len(got), len(w))
			continue
		}
		for r, wr := range w {
			if len(got[r]) != len(wr) {
				t.Errorf("row %d subrow %d: len = %d, want %d", i, r, len(got[r]), len(wr))
				continue
			}
			for c, wv := range wr {
				if v, ok := got[r][c].(int64); !ok || v != wv {
					t.Errorf("row %d [%d,%d]: got %v (%T), want %d", i, r, c, got[r][c], got[r][c], wv)
				}
			}
		}
	}
}

func TestContainerArrDynamic(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "arr_dynamic.supr"))

	want := [][]int64{{10, 20, 30}, {5, 6}}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		got, ok := rows[i][0].([]any)
		if !ok {
			t.Fatalf("row %d: type = %T, want []any", i, rows[i][0])
		}
		if len(got) != len(w) {
			t.Errorf("row %d: len = %d, want %d", i, len(got), len(w))
			continue
		}
		for j, wv := range w {
			if v, ok := got[j].(int64); !ok || v != wv {
				t.Errorf("row %d elem %d: got %v (%T), want %d", i, j, got[j], got[j], wv)
			}
		}
	}
}

func TestContainerMixed(t *testing.T) {
	rows := decodeTestdataFile(t, testdataPath(t, "containers", "mixed.supr"))

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// Row 0: id=1, tags=[red,green,blue], coords=[10,20,30]
	id, ok := rows[0][0].(int64)
	if !ok || id != 1 {
		t.Errorf("row 0 col 0: got %v (%T), want 1", rows[0][0], rows[0][0])
	}

	tags, ok := rows[0][1].([]any)
	if !ok {
		t.Fatalf("row 0 col 1: type = %T, want []any", rows[0][1])
	}
	wantTags := []string{"red", "green", "blue"}
	if len(tags) != len(wantTags) {
		t.Errorf("row 0 tags: len = %d, want %d", len(tags), len(wantTags))
	} else {
		for j, w := range wantTags {
			if v, ok := tags[j].(string); !ok || v != w {
				t.Errorf("row 0 tags[%d]: got %v, want %q", j, tags[j], w)
			}
		}
	}

	coords, ok := rows[0][2].([]any)
	if !ok {
		t.Fatalf("row 0 col 2: type = %T, want []any", rows[0][2])
	}
	wantCoords := []int64{10, 20, 30}
	if len(coords) != len(wantCoords) {
		t.Errorf("row 0 coords: len = %d, want %d", len(coords), len(wantCoords))
	} else {
		for j, w := range wantCoords {
			if v, ok := coords[j].(int64); !ok || v != w {
				t.Errorf("row 0 coords[%d]: got %v, want %d", j, coords[j], w)
			}
		}
	}
}
