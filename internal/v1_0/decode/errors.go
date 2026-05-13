// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import "fmt"

// ContainerElementError wraps an element decoding error with its 0-based index.
type ContainerElementError struct {
	Index int   // 0-based element index
	Err   error // underlying decoding error
}

func (e *ContainerElementError) Error() string {
	return e.Err.Error()
}

func (e *ContainerElementError) Unwrap() error {
	return e.Err
}

// Array2DElementError wraps a 2D array element error with 0-based row/col indices.
type Array2DElementError struct {
	Row int   // 0-based row index
	Col int   // 0-based column index
	Err error // underlying decoding error
}

func (e *Array2DElementError) Error() string {
	return e.Err.Error()
}

func (e *Array2DElementError) Unwrap() error {
	return e.Err
}

// LengthMismatchError indicates expected vs actual element count mismatch.
type LengthMismatchError struct {
	Expected int
	Actual   int
}

func (e *LengthMismatchError) Error() string {
	return fmt.Sprintf("expected %d elements, got %d", e.Expected, e.Actual)
}
