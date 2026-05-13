// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

// This file provides test helpers for public_*_test.go files.
// These are duplicates of helpers in roundtrip_test.go, renamed with a "pub"
// prefix so both sets can coexist in the same package without collision.
// The originals remain in roundtrip_test.go (which is not exported).

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// pubDecodeFunc is the signature shared by streaming and batch decode helpers.
type pubDecodeFunc func(t *testing.T, r io.Reader) *supr.File

// pubDecodeToFile decodes from reader to File using streaming Next()/Row().
func pubDecodeToFile(t *testing.T, r io.Reader) *supr.File {
	t.Helper()
	d := supr.NewDecoder(r)
	var rows [][]any
	for d.Next() {
		row := make([]any, len(d.Row()))
		copy(row, d.Row())
		rows = append(rows, row)
	}
	if err := d.Err(); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	return &supr.File{Header: d.Header(), Rows: rows}
}

// pubDecodeToBatchFile decodes from reader to File using batch DecodeFile().
func pubDecodeToBatchFile(t *testing.T, r io.Reader) *supr.File {
	t.Helper()
	d := supr.NewDecoder(r)
	file, err := d.DecodeFile()
	if err != nil {
		t.Fatalf("DecodeFile failed: %v", err)
	}
	return file
}

// pubForEachDecodeMode runs fn once for streaming decode and once for batch decode.
func pubForEachDecodeMode(t *testing.T, fn func(t *testing.T, decode pubDecodeFunc)) {
	t.Helper()
	modes := []struct {
		name   string
		decode pubDecodeFunc
	}{
		{"streaming", pubDecodeToFile},
		{"batch", pubDecodeToBatchFile},
	}
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			fn(t, m.decode)
		})
	}
}

// pubAssertEncodesTo verifies that encoding the given file produces output matching a golden file.
func pubAssertEncodesTo(t *testing.T, file *supr.File, goldenParts ...string) {
	t.Helper()
	var buf bytes.Buffer
	if err := supr.EncodeFile(&buf, file); err != nil {
		t.Fatalf("EncodeFile failed (golden check): %v", err)
	}
	golden := readTestdataFile(t, goldenParts...)
	if buf.String() != golden {
		t.Errorf("encoded output does not match golden file %s\ngot:\n%s\nwant:\n%s",
			filepath.Join(goldenParts...), buf.String(), golden)
	}
}
