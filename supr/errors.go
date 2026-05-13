// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"errors"
	"fmt"
)

// Common sentinel errors

var (
	// ErrInvalidHeader indicates the header is malformed or invalid.
	ErrInvalidHeader = errors.New("invalid header")

	// ErrInvalidRow indicates a row has the wrong number of fields.
	ErrInvalidRow = errors.New("invalid row")

	// ErrValidation indicates a field failed type validation.
	ErrValidation = errors.New("validation error")

	// ErrMaxErrors indicates the validator stopped due to reaching MaxErrors limit.
	ErrMaxErrors = errors.New("max errors reached")

	// ErrMaxFieldSize indicates a field exceeded MaxFieldSize limit.
	ErrMaxFieldSize = errors.New("max field size exceeded")

	// ErrDecode indicates a field could not be decoded to the target type.
	ErrDecode = errors.New("decode error")

	// ErrEncode indicates a value could not be encoded.
	ErrEncode = errors.New("encode error")

	// ErrIO indicates an I/O error occurred.
	ErrIO = errors.New("I/O error")
)

// ValidationError provides detailed context for validation failures.
type ValidationError struct {
	Row    int    // 1-based row number (0 = header)
	Col    int    // 1-based column number
	Field  string // Raw field value
	Reason string // Human-readable reason
	Err    error  // Wrapped error
}

func (e *ValidationError) Error() string {
	if e.Col > 0 {
		return fmt.Sprintf("row %d, col %d (%q): %s", e.Row, e.Col, e.Field, e.Reason)
	}
	return fmt.Sprintf("row %d: %s", e.Row, e.Reason)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// DecodeError provides context for decoding failures.
type DecodeError struct {
	Row int   // 1-based row number
	Err error // Underlying error (contains column/type details)
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("row %d decode error: %s", e.Row, e.Err)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}

// EncodeError provides context for encoding failures.
type EncodeError struct {
	Row int   // 1-based row number
	Col int   // 1-based column number (0 if not column-specific)
	Err error // Underlying error
}

func (e *EncodeError) Error() string {
	if e.Col > 0 {
		return fmt.Sprintf("row %d, col %d encode error: %s", e.Row, e.Col, e.Err)
	}
	return fmt.Sprintf("row %d encode error: %s", e.Row, e.Err)
}

func (e *EncodeError) Unwrap() error {
	return e.Err
}
