// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"os"
	"slices"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// TestDecodeStreaming verifies the streaming Next/Row/Err pattern.
func TestDecodeStreaming(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_basic.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)

	// Header should be available before iterating
	hdr := d.Header()
	if hdr == nil {
		t.Fatal("Header() returned nil")
	}
	if len(hdr.Columns) != 4 {
		t.Fatalf("Header columns = %d, want 4", len(hdr.Columns))
	}

	var rows [][]any
	for d.Next() {
		rows = append(rows, slices.Clone(d.Row()))
	}
	if err := d.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	// Verify first row: 1, Alice, 95.5, true
	if got, ok := rows[0][0].(int64); !ok || got != 1 {
		t.Errorf("row 0 col 0 = %v (%T), want 1 (int64)", rows[0][0], rows[0][0])
	}
	if got, ok := rows[0][1].(string); !ok || got != "Alice" {
		t.Errorf("row 0 col 1 = %v (%T), want Alice (string)", rows[0][1], rows[0][1])
	}
	if got, ok := rows[0][2].(float64); !ok || got != 95.5 {
		t.Errorf("row 0 col 2 = %v (%T), want 95.5 (float64)", rows[0][2], rows[0][2])
	}
	if got, ok := rows[0][3].(bool); !ok || got != true {
		t.Errorf("row 0 col 3 = %v (%T), want true (bool)", rows[0][3], rows[0][3])
	}
}

// TestDecodeRowNum verifies RowNum increments correctly.
func TestDecodeRowNum(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_basic.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)

	if d.RowNum() != 0 {
		t.Errorf("RowNum() before Next = %d, want 0", d.RowNum())
	}

	for i := 1; d.Next(); i++ {
		if d.RowNum() != i {
			t.Errorf("RowNum() after Next %d = %d, want %d", i, d.RowNum(), i)
		}
	}
	if err := d.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
}

// TestDecodeBatchPath verifies supr.DecodeFile(path) convenience function.
func TestDecodeBatchPath(t *testing.T) {
	path := testdataPath(t, "decode", "decode_basic.supr")

	file, err := supr.DecodeFile(path)
	if err != nil {
		t.Fatalf("DecodeFile(%q) = %v", path, err)
	}

	if file.Header == nil {
		t.Fatal("File.Header is nil")
	}
	if len(file.Header.Columns) != 4 {
		t.Fatalf("Header columns = %d, want 4", len(file.Header.Columns))
	}
	if len(file.Rows) != 3 {
		t.Fatalf("Rows = %d, want 3", len(file.Rows))
	}

	// Spot-check row 2: 3, Charlie, 92.1, true
	row := file.Rows[2]
	if got, ok := row[0].(int64); !ok || got != 3 {
		t.Errorf("row 2 col 0 = %v (%T), want 3", row[0], row[0])
	}
	if got, ok := row[1].(string); !ok || got != "Charlie" {
		t.Errorf("row 2 col 1 = %v (%T), want Charlie", row[1], row[1])
	}
}

// TestDecodeBatchReader verifies decoder.DecodeFile() (reader-based batch).
func TestDecodeBatchReader(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_basic.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	file, err := d.DecodeFile()
	if err != nil {
		t.Fatalf("DecodeFile() = %v", err)
	}

	if len(file.Rows) != 3 {
		t.Fatalf("Rows = %d, want 3", len(file.Rows))
	}

	// Verify Header is accessible from File
	if file.Header.Columns[0].Name != "id" {
		t.Errorf("Column[0].Name = %q, want %q", file.Header.Columns[0].Name, "id")
	}
}

// TestDecodeModeRaw verifies DecodeRaw returns []byte for all fields.
func TestDecodeModeRaw(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_modes.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeRaw))
	if !d.Next() {
		t.Fatalf("Next() = false; err: %v", d.Err())
	}

	row := d.Row()

	// In Raw mode, all fields are []byte
	num, ok := row[0].([]byte)
	if !ok {
		t.Fatalf("DecodeRaw: row[0] type = %T, want []byte", row[0])
	}
	if string(num) != "42" {
		t.Errorf("DecodeRaw: row[0] = %q, want %q", num, "42")
	}

	tags, ok := row[1].([]byte)
	if !ok {
		t.Fatalf("DecodeRaw: row[1] type = %T, want []byte", row[1])
	}
	if string(tags) != "[A,B,C]" {
		t.Errorf("DecodeRaw: row[1] = %q, want %q", tags, "[A,B,C]")
	}
}

// TestDecodeModeShallow verifies DecodeShallow decodes scalars but keeps containers as strings.
func TestDecodeModeShallow(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_modes.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeShallow))
	if !d.Next() {
		t.Fatalf("Next() = false; err: %v", d.Err())
	}

	row := d.Row()

	// Scalar should be decoded
	num, ok := row[0].(int64)
	if !ok {
		t.Fatalf("DecodeShallow: row[0] type = %T, want int64", row[0])
	}
	if num != 42 {
		t.Errorf("DecodeShallow: row[0] = %d, want 42", num)
	}

	// Container should remain as string
	tags, ok := row[1].(string)
	if !ok {
		t.Fatalf("DecodeShallow: row[1] type = %T, want string", row[1])
	}
	if tags != "[A,B,C]" {
		t.Errorf("DecodeShallow: row[1] = %q, want %q", tags, "[A,B,C]")
	}
}

// TestDecodeModeFull verifies DecodeFull fully decodes all fields.
func TestDecodeModeFull(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_modes.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeFull))
	if !d.Next() {
		t.Fatalf("Next() = false; err: %v", d.Err())
	}

	row := d.Row()

	// Scalar fully decoded
	num, ok := row[0].(int64)
	if !ok {
		t.Fatalf("DecodeFull: row[0] type = %T, want int64", row[0])
	}
	if num != 42 {
		t.Errorf("DecodeFull: row[0] = %d, want 42", num)
	}

	// Container fully decoded to []any
	tags, ok := row[1].([]any)
	if !ok {
		t.Fatalf("DecodeFull: row[1] type = %T, want []any", row[1])
	}
	if len(tags) != 3 {
		t.Fatalf("DecodeFull: len(tags) = %d, want 3", len(tags))
	}
	for i, want := range []string{"A", "B", "C"} {
		if got, ok := tags[i].(string); !ok || got != want {
			t.Errorf("DecodeFull: tags[%d] = %v (%T), want %q", i, tags[i], tags[i], want)
		}
	}
}

// TestDecodeHeaderAccess verifies Decoder.Header() is available before iteration.
func TestDecodeHeaderAccess(t *testing.T) {
	f, err := os.Open(testdataPath(t, "decode", "decode_modes.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	hdr := d.Header()
	if hdr == nil {
		t.Fatal("Header() returned nil before Next()")
	}

	if len(hdr.Columns) != 2 {
		t.Fatalf("Header columns = %d, want 2", len(hdr.Columns))
	}
	if hdr.Columns[0].Name != "num" {
		t.Errorf("Column[0].Name = %q, want %q", hdr.Columns[0].Name, "num")
	}
	if hdr.Columns[1].Name != "tags" {
		t.Errorf("Column[1].Name = %q, want %q", hdr.Columns[1].Name, "tags")
	}
}
