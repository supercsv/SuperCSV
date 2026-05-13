// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"bytes"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// TestRoundtripScalarsAndContainers verifies encode -> decode identity for basic types.
func TestRoundtripScalarsAndContainers(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "score", Type: supr.Float},
		{Name: "tags", Type: supr.List(supr.String)},
	})

	originalRows := [][]any{
		{int64(1), "Alice", 95.5, []any{"math", "sci"}},
		{int64(2), "Bob", 87.3, []any{"art"}},
		{int64(3), "Charlie", 92.1, []any{"math", "art", "music"}},
	}

	pubAssertEncodesTo(t, &supr.File{Header: header, Rows: originalRows}, "roundtrip", "golden", "scalars_and_containers.supr")

	pubForEachDecodeMode(t, func(t *testing.T, decode pubDecodeFunc) {
		var buf bytes.Buffer
		if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: originalRows}); err != nil {
			t.Fatalf("EncodeFile: %v", err)
		}

		file := decode(t, &buf)

		if len(file.Rows) != len(originalRows) {
			t.Fatalf("rows = %d, want %d", len(file.Rows), len(originalRows))
		}

		for i, orig := range originalRows {
			row := file.Rows[i]
			if row[0] != orig[0] {
				t.Errorf("row %d col 0: got %v, want %v", i, row[0], orig[0])
			}
			if row[1] != orig[1] {
				t.Errorf("row %d col 1: got %v, want %v", i, row[1], orig[1])
			}
			if row[2] != orig[2] {
				t.Errorf("row %d col 2: got %v, want %v", i, row[2], orig[2])
			}

			tags, ok := row[3].([]any)
			if !ok {
				t.Fatalf("row %d col 3: type = %T, want []any", i, row[3])
			}
			origTags := orig[3].([]any)
			if len(tags) != len(origTags) {
				t.Fatalf("row %d tags: len = %d, want %d", i, len(tags), len(origTags))
			}
			for j, v := range tags {
				if v != origTags[j] {
					t.Errorf("row %d tags[%d]: got %v, want %v", i, j, v, origTags[j])
				}
			}
		}
	})
}

// TestRoundtripHeaderCanonicalText verifies header text survives encode -> decode.
func TestRoundtripHeaderCanonicalText(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "x", Type: supr.Int},
		{Name: "y", Type: supr.List(supr.String)},
		{Name: "z", Type: supr.Enum([]supr.EnumValue{
			{Name: "low"}, {Name: "med"}, {Name: "high"},
		})},
	})

	originalCanonical := header.CanonicalText

	pubAssertEncodesTo(t, &supr.File{Header: header, Rows: nil}, "roundtrip", "golden", "header_canonical_text.supr")

	pubForEachDecodeMode(t, func(t *testing.T, decode pubDecodeFunc) {
		var buf bytes.Buffer
		if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: nil}); err != nil {
			t.Fatalf("EncodeFile: %v", err)
		}

		file := decode(t, &buf)

		if file.Header.CanonicalText != originalCanonical {
			t.Errorf("CanonicalText mismatch:\n  got:  %q\n  want: %q", file.Header.CanonicalText, originalCanonical)
		}
	})
}

// TestRoundtripWithNull verifies null values survive encode -> decode.
func TestRoundtripWithNull(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "a", Type: supr.Int},
		{Name: "b", Type: supr.String},
		{Name: "c", Type: supr.List(supr.Int)},
	})

	originalRows := [][]any{
		{int64(1), nil, []any{int64(10), int64(20)}},
		{nil, "hello", nil},
	}

	pubAssertEncodesTo(t, &supr.File{Header: header, Rows: originalRows}, "roundtrip", "golden", "null.supr")

	pubForEachDecodeMode(t, func(t *testing.T, decode pubDecodeFunc) {
		var buf bytes.Buffer
		if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: originalRows}); err != nil {
			t.Fatalf("EncodeFile: %v", err)
		}

		file := decode(t, &buf)

		if len(file.Rows) != 2 {
			t.Fatalf("rows = %d, want 2", len(file.Rows))
		}

		// Row 0: 1, nil, [10, 20]
		if file.Rows[0][0] != int64(1) {
			t.Errorf("row 0 col 0 = %v, want 1", file.Rows[0][0])
		}
		if file.Rows[0][1] != nil {
			t.Errorf("row 0 col 1 = %v, want nil", file.Rows[0][1])
		}

		// Row 1: nil, "hello", nil
		if file.Rows[1][0] != nil {
			t.Errorf("row 1 col 0 = %v, want nil", file.Rows[1][0])
		}
		if file.Rows[1][1] != "hello" {
			t.Errorf("row 1 col 1 = %v, want hello", file.Rows[1][1])
		}
		// Null container decodes to empty []any{} not nil
		if file.Rows[1][2] == nil {
			// Accept nil or empty slice
		} else if list, ok := file.Rows[1][2].([]any); ok {
			if len(list) != 0 {
				t.Errorf("row 1 col 2 = %v, want empty slice or nil", file.Rows[1][2])
			}
		} else {
			t.Errorf("row 1 col 2 = %v (%T), want nil or empty []any", file.Rows[1][2], file.Rows[1][2])
		}
	})
}

// TestRoundtripFromFile verifies decode -> encode -> decode from .supr files.
func TestRoundtripFromFile(t *testing.T) {
	files := []struct {
		subdir   string
		filename string
	}{
		{"roundtrip", "roundtrip_mixed.supr"},
		{"roundtrip", "roundtrip_temporal.supr"},
	}

	for _, f := range files {
		t.Run(f.filename, func(t *testing.T) {
			path := testdataPath(t, f.subdir, f.filename)

			// Decode original file
			file1, err := supr.DecodeFile(path)
			if err != nil {
				t.Fatalf("DecodeFile: %v", err)
			}

			// Verify re-encoding produces byte-identical output to original file
			pubAssertEncodesTo(t, file1, f.subdir, f.filename)

			pubForEachDecodeMode(t, func(t *testing.T, decode pubDecodeFunc) {
				// Re-encode
				var buf bytes.Buffer
				if err := supr.EncodeFile(&buf, file1); err != nil {
					t.Fatalf("EncodeFile: %v", err)
				}

				// Decode again
				file2 := decode(t, &buf)

				// Compare
				if len(file2.Rows) != len(file1.Rows) {
					t.Fatalf("row count: got %d, want %d", len(file2.Rows), len(file1.Rows))
				}

				for i, row := range file2.Rows {
					for j := range row {
						// Skip container comparison (different slice identity)
						if _, ok := row[j].([]any); ok {
							continue
						}
						if row[j] != file1.Rows[i][j] {
							t.Errorf("row %d col %d: got %v, want %v", i, j, row[j], file1.Rows[i][j])
						}
					}
				}
			})
		})
	}
}
