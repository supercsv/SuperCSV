// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// TestErrorValidationSentinel verifies errors.Is works with ErrValidation via RowError().
func TestErrorValidationSentinel(t *testing.T) {
	f, err := os.Open(testdataPath(t, "errors", "invalid_rows.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v := supr.NewValidator(f)
	var rowErr error
	for v.Next() {
		if e := v.RowError(); e != nil {
			rowErr = e
			break
		}
	}

	if rowErr == nil {
		t.Fatal("expected validation row error, got nil")
	}

	if !errors.Is(rowErr, supr.ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false; err = %v", rowErr)
	}
}

// TestErrorValidationErrorAs verifies errors.As works with ValidationError via RowError().
func TestErrorValidationErrorAs(t *testing.T) {
	f, err := os.Open(testdataPath(t, "errors", "invalid_rows.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v := supr.NewValidator(f)
	var rowErr error
	for v.Next() {
		if e := v.RowError(); e != nil {
			rowErr = e
			break
		}
	}

	if rowErr == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *supr.ValidationError
	if !errors.As(rowErr, &ve) {
		t.Fatalf("errors.As(err, *ValidationError) = false; err type = %T", rowErr)
	}

	// ValidationError should have meaningful fields
	if ve.Row == 0 {
		t.Errorf("ValidationError.Row = 0, want non-zero")
	}
	if ve.Reason == "" {
		t.Errorf("ValidationError.Reason is empty")
	}
}

// TestErrorMaxErrors verifies ErrMaxErrors sentinel.
func TestErrorMaxErrors(t *testing.T) {
	f, err := os.Open(testdataPath(t, "errors", "invalid_rows.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v := supr.NewValidator(f, supr.WithMaxErrors(1))
	for v.Next() {
		// consume rows
	}

	err = v.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, supr.ErrMaxErrors) {
		t.Errorf("errors.Is(err, ErrMaxErrors) = false; err = %v", err)
	}
}

// TestErrorInvalidHeader verifies decoder error on bad header.
func TestErrorInvalidHeader(t *testing.T) {
	f, err := os.Open(testdataPath(t, "errors", "bad_header.supr"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)

	// Decoder should fail on first Next() or return false immediately
	if d.Next() {
		t.Error("Next() = true on bad header, want false")
	}

	err = d.Err()
	if err == nil {
		t.Fatal("expected error for bad header, got nil")
	}

	// Error message should mention header or schema
	errMsg := err.Error()
	if !strings.Contains(errMsg, "header") && !strings.Contains(errMsg, "schema") {
		t.Errorf("error = %q, expected to mention header or schema", errMsg)
	}
}

// TestErrorEncoderTypeMismatch verifies encoder rejects wrong types.
func TestErrorEncoderTypeMismatch(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)
	if err := e.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	// Pass a string where int is expected
	err := e.WriteRow([]any{"not_a_number"})
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}
}

// TestErrorEncoderRowLength verifies encoder rejects wrong row length.
func TestErrorEncoderRowLength(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "a", Type: supr.Int},
		{Name: "b", Type: supr.String},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)
	if err := e.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	// Pass too few columns
	err := e.WriteRow([]any{int64(1)})
	if err == nil {
		t.Fatal("expected error for wrong row length, got nil")
	}

	if !strings.Contains(err.Error(), "row length") {
		t.Errorf("error = %q, expected to mention row length", err.Error())
	}
}

// TestErrorDecodeErrorAs verifies errors.As works with DecodeError on decode failure.
func TestErrorDecodeErrorAs(t *testing.T) {
	input := "id:int\nabc\n"
	d := supr.NewDecoder(strings.NewReader(input))

	if d.Next() {
		t.Fatal("Next() = true on invalid int, want false")
	}

	err := d.Err()
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	var de *supr.DecodeError
	if !errors.As(err, &de) {
		t.Fatalf("errors.As(err, *DecodeError) = false; err type = %T, err = %v", err, err)
	}

	if de.Row != 2 { // line 2 in file (header is line 1)
		t.Errorf("DecodeError.Row = %d, want 2", de.Row)
	}

	if de.Err == nil {
		t.Error("DecodeError.Err = nil, want underlying error")
	}
}

// TestErrorEncodeErrorAs verifies errors.As works with EncodeError on encode failure.
func TestErrorEncodeErrorAs(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)
	if err := e.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	// Pass a string where int is expected
	err := e.WriteRow([]any{"not_a_number", "Alice"})
	if err == nil {
		t.Fatal("expected encode error, got nil")
	}

	var ee *supr.EncodeError
	if !errors.As(err, &ee) {
		t.Fatalf("errors.As(err, *EncodeError) = false; err type = %T, err = %v", err, err)
	}

	if ee.Row != 1 {
		t.Errorf("EncodeError.Row = %d, want 1", ee.Row)
	}

	if ee.Col != 1 {
		t.Errorf("EncodeError.Col = %d, want 1", ee.Col)
	}

	if ee.Err == nil {
		t.Error("EncodeError.Err = nil, want underlying error")
	}
}

// TestErrorDecoderInvalidData verifies decoder error on invalid data.
func TestErrorDecoderInvalidData(t *testing.T) {
	// Valid header, invalid data row (wrong type)
	input := "val:int\nnot_a_number\n"

	d := supr.NewDecoder(strings.NewReader(input))
	if d.Next() {
		t.Error("Next() = true for invalid data, want false")
	}

	if d.Err() == nil {
		t.Fatal("expected decode error, got nil")
	}
}
