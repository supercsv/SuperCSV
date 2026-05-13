// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import "fmt"

// ContainerElementError wraps an element validation error with its 0-based index.
// Used for both lists and 1D arrays.
type ContainerElementError struct {
	Index int   // 0-based element index
	Err   error // underlying validation error
}

func (e *ContainerElementError) Error() string {
	// Position info is now in Column field (e.g., "Tags(2)"), so just return the error
	return e.Err.Error()
}

func (e *ContainerElementError) Unwrap() error {
	return e.Err
}

// Array2DElementError wraps a 2D array element error with 0-based row/col indices.
type Array2DElementError struct {
	Row int   // 0-based row index
	Col int   // 0-based column index
	Err error // underlying validation error
}

func (e *Array2DElementError) Error() string {
	// Position info is now in Column field (e.g., "Matrix(1,2)"), so just return the error
	return e.Err.Error()
}

func (e *Array2DElementError) Unwrap() error {
	return e.Err
}

// LengthMismatchError indicates expected vs actual element count mismatch.
// Used for structured error reporting without string parsing.
type LengthMismatchError struct {
	Expected int
	Actual   int
}

func (e *LengthMismatchError) Error() string {
	return fmt.Sprintf("expected %d elements, got %d", e.Expected, e.Actual)
}
