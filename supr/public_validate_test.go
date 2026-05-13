// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

func TestValidateBasic(t *testing.T) {
	f, err := os.Open(testdataPath(t, "validate", "valid_basic.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v := supr.NewValidator(f)

	rowCount := 0
	for v.Next() {
		if err := v.RowError(); err != nil {
			t.Errorf("row %d: unexpected error: %v", v.RowNum(), err)
		}
		rowCount++
	}
	if err := v.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
	if !v.Valid() {
		t.Error("Valid() = false, want true")
	}
	if rowCount != 3 {
		t.Errorf("row count = %d, want 3", rowCount)
	}
}

func TestValidateInvalidRows(t *testing.T) {
	f, err := os.Open(testdataPath(t, "validate", "invalid_rows.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v := supr.NewValidator(f)

	var validationErrors []error
	for v.Next() {
		if err := v.RowError(); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}
	if err := v.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil (validation errors are not fatal)", err)
	}

	// "bad" and "xyz" are invalid ints - expect 2 errors
	if len(validationErrors) != 2 {
		t.Errorf("got %d validation errors, want 2", len(validationErrors))
	}
	if v.Valid() {
		t.Error("Valid() = true, want false (had validation errors)")
	}
}

func TestValidateWithMaxErrors(t *testing.T) {
	// Use inline data - 4 invalid rows but stop after 1
	input := "value:int\nbad\ninvalid\nwrong\nalso_bad\n"
	v := supr.NewValidator(strings.NewReader(input), supr.WithMaxErrors(1))

	for v.Next() {
		// consume
	}

	if !errors.Is(v.Err(), supr.ErrMaxErrors) {
		t.Errorf("Err() = %v, want ErrMaxErrors", v.Err())
	}
}

func TestValidateWithMaxFieldSize(t *testing.T) {
	f, err := os.Open(testdataPath(t, "validate", "max_field_size.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Set a 20-byte limit - the long field exceeds this
	v := supr.NewValidator(f, supr.WithMaxFieldSize(20))

	for v.Next() {
		// consume
	}

	if err := v.Err(); err == nil {
		t.Error("Err() = nil, want error for oversized field")
	}
}

func TestValidateFile(t *testing.T) {
	f, err := os.Open(testdataPath(t, "validate", "invalid_rows.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	header, errs, err := supr.ValidateFile(f)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}

	if header == nil {
		t.Fatal("header is nil")
	}
	if header.NumColumns() != 2 {
		t.Errorf("NumColumns() = %d, want 2", header.NumColumns())
	}

	// Expect 2 validation errors for "bad" and "xyz"
	if len(errs) != 2 {
		t.Errorf("got %d errors, want 2", len(errs))
	}
}
