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

func TestHeaderBasic(t *testing.T) {
	f, err := os.Open(testdataPath(t, "header", "header_basic.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	h := d.Header()

	if h.NumColumns() != 3 {
		t.Fatalf("NumColumns() = %d, want 3", h.NumColumns())
	}

	tests := []struct {
		name     string
		typeName string
	}{
		{"id", "int"},
		{"name", "string"},
		{"active", "bool"},
	}
	for i, tt := range tests {
		if h.Columns[i].Name != tt.name {
			t.Errorf("col %d name = %q, want %q", i, h.Columns[i].Name, tt.name)
		}
		if h.Columns[i].Type.String() != tt.typeName {
			t.Errorf("col %d type = %q, want %q", i, h.Columns[i].Type.String(), tt.typeName)
		}
	}
}

func TestHeaderContinuation(t *testing.T) {
	f, err := os.Open(testdataPath(t, "header", "header_continuation.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)
	h := d.Header()

	// Multi-line header should parse to same columns as single-line
	if h.NumColumns() != 3 {
		t.Fatalf("NumColumns() = %d, want 3", h.NumColumns())
	}
	if h.Columns[0].Name != "id" || h.Columns[1].Name != "name" || h.Columns[2].Name != "active" {
		t.Errorf("columns = [%s, %s, %s], want [id, name, active]",
			h.Columns[0].Name, h.Columns[1].Name, h.Columns[2].Name)
	}

	// CanonicalText should be the normalized single-line form
	want := "id:int,name:string,active:bool"
	if h.CanonicalText != want {
		t.Errorf("CanonicalText = %q, want %q", h.CanonicalText, want)
	}

	// Verify data rows still parse correctly
	var rowCount int
	for d.Next() {
		rowCount++
	}
	if err := d.Err(); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("row count = %d, want 2", rowCount)
	}
}

func TestHeaderComments(t *testing.T) {
	f, err := os.Open(testdataPath(t, "header", "header_comments.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)

	if d.Header().NumColumns() != 2 {
		t.Fatalf("NumColumns() = %d, want 2", d.Header().NumColumns())
	}

	// Comments are skipped; only data rows are returned
	var rowCount int
	for d.Next() {
		rowCount++
	}
	if err := d.Err(); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("row count = %d, want 2", rowCount)
	}
}

func TestHeaderNewHeader(t *testing.T) {
	// Programmatic header construction
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "tags", Type: supr.List(supr.String)},
		{Name: "status", Type: supr.Enum([]supr.EnumValue{
			{Name: "active"},
			{Name: "inactive"},
		})},
	})

	if header.NumColumns() != 4 {
		t.Fatalf("NumColumns() = %d, want 4", header.NumColumns())
	}

	// RawText is empty for programmatically constructed headers
	if header.RawText != "" {
		t.Errorf("RawText = %q, want empty", header.RawText)
	}

	// CanonicalText should be populated
	if header.CanonicalText == "" {
		t.Error("CanonicalText is empty, want non-empty")
	}

	// String() returns CanonicalText
	if header.String() != header.CanonicalText {
		t.Errorf("String() = %q, CanonicalText = %q, want equal", header.String(), header.CanonicalText)
	}

	// Columns are accessible
	if header.Columns[0].Type.Kind() != supr.KindScalar {
		t.Errorf("col 0 kind = %v, want KindScalar", header.Columns[0].Type.Kind())
	}
	if header.Columns[2].Type.Kind() != supr.KindList {
		t.Errorf("col 2 kind = %v, want KindList", header.Columns[2].Type.Kind())
	}
	if header.Columns[3].Type.Kind() != supr.KindEnum {
		t.Errorf("col 3 kind = %v, want KindEnum", header.Columns[3].Type.Kind())
	}
}
