// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"
	"io"
	"os"
	"sync"

	csvstream "github.com/supercsv/supercsv/internal/v1_0/csvstream"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/headerdefreader"
	"github.com/supercsv/supercsv/internal/v1_0/validate/errors"
)

// fieldBuffers holds reusable slices for field extraction to avoid per-row allocations
type fieldBuffers struct {
	fields        [][]byte
	quoted        []bool
	commentCounts []int8
	metaCounts    []int8
}

// fieldBufferPool reuses field buffer slices across rows (2M allocation reduction)
var fieldBufferPool = sync.Pool{
	New: func() interface{} {
		return &fieldBuffers{
			fields:        make([][]byte, 0, 32), // Pre-allocate for typical row size
			quoted:        make([]bool, 0, 32),
			commentCounts: make([]int8, 0, 32),
			metaCounts:    make([]int8, 0, 32),
		}
	},
}

// FileValidator orchestrates file-level CSV validation.
// It coordinates CSV parsing, schema validation, and row-by-row data validation.
type FileValidator struct {
	maxFieldSize  int
	maxErrors     int
	strictVersion string // Required SuperCSV version (empty = no check)
}

// ValidationResult captures the outcome of validating a CSV file.
type ValidationResult struct {
	Valid       bool                      // False if any validation errors occurred
	Errors      []*errors.ValidationError // All collected validation errors
	RowsScanned int                       // Total number of data rows processed
}

// NewFileValidator creates a new file validator with the specified limits.
// maxFieldSize controls the maximum size of any single CSV field (0 = unlimited).
// maxErrors controls how many validation errors to collect before stopping (0 = unlimited).
// strictVersion enforces a required SuperCSV version directive (empty = no check).
func NewFileValidator(maxFieldSize, maxErrors int, strictVersion string) *FileValidator {
	return &FileValidator{
		maxFieldSize:  maxFieldSize,
		maxErrors:     maxErrors,
		strictVersion: strictVersion,
	}
}

// fatalError constructs a fatal validation error and returns a ValidationResult.
// This helper eliminates boilerplate for unrecoverable validation errors.
func (fv *FileValidator) fatalError(row int, column string, code string, err error, rowsScanned int, msgPrefix string) (*ValidationResult, error) {
	msg := fmt.Sprintf("FATAL: %v", err)
	if msgPrefix != "" {
		msg = fmt.Sprintf("FATAL: %s: %v", msgPrefix, err)
	}
	return &ValidationResult{
		Valid: false,
		Errors: []*errors.ValidationError{{
			Row:    row,
			Column: column,
			Code:   code,
			Msg:    msg,
			Fatal:  true,
		}},
		RowsScanned: rowsScanned,
	}, nil
}

// ValidateFile validates a SuperCSV file and returns all validation errors found.
// It performs schema validation on the header and type-aware validation on data rows.
// Returns ValidationResult with collected errors and row count, or a Go error for I/O/parse failures.
func (fv *FileValidator) ValidateFile(path string) (*ValidationResult, error) {
	// 1. Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// 2. Create CSV stream parser with header mode for first row
	stream, err := csvstream.NewCSVStream(f, csvstream.StreamOptions{
		MaxFieldSize:  fv.maxFieldSize,
		HeaderMode:    true, // Enable angle bracket tracking for type declarations
		StrictVersion: fv.strictVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV stream: %w", err)
	}

	// 3. Read and parse header (with continuation support)
	var validationErrors []*errors.ValidationError
	schemaRow, err := fv.readHeaderWithContinuation(stream)
	if err != nil {
		return fv.fatalError(1, "header", errors.CodeCSVParseError.Code, err, 0, "")
	}

	// Disable header mode for data rows
	stream.SetHeaderMode(false)

	// Parse schema
	hdef, err := headerdef.ParseHeader(schemaRow)
	if err != nil {
		code := errors.CodeInvalidHeaderSchema.Code
		line := schemaRow.Line
		if herr, ok := err.(*headerdef.Error); ok && herr.Code == headerdef.CodeInvalidEnum {
			code = errors.CodeInvalidEnumSchema.Code
			line = herr.Line
		} else if herr, ok := err.(*headerdef.Error); ok {
			line = herr.Line
			switch herr.Code {
			case headerdef.CodeDuplicateAnnotation:
				code = errors.CodeDuplicateAnnotation.Code
			case headerdef.CodeAnnotationAfterTrailing:
				code = errors.CodeAnnotationAfterTrailingComma.Code
			}
		}
		return fv.fatalError(line, "header", code, err, 0, "invalid header")
	}

	// Strict version validation happens upfront in newCSVStream via checkStrictVersionFirstLine
	// No need to check again here - stream creation would have already failed

	// 4. Create row validator
	rowValidator := NewRowValidator(hdef, fv.maxFieldSize)

	// 5. Process data rows
	rowsScanned := 0

	// Continuation state (rare path - only used for multi-line data rows per 9.1)
	var contFields [][]byte
	var contQuoted []bool
	var contCommentCounts []int8
	var contMetaCounts []int8
	var contStartLine int
	inContinuation := false

	for {
		rawRow, err := stream.NextRow()
		// Consume comment-line-skipped flag immediately so it doesn't persist
		// across iterations. Only an error when mid-data-continuation.
		commentLineSkipped := stream.CommentLineSkipped()
		// Consume standalone annotation counts (set by skipLeadingBlanks for
		// annotation lines before this physical row). Only meaningful in
		// continuation context where they apply to the next field.
		standaloneComments, standaloneMeta := stream.StandaloneAnnotationsSkipped()
		if err != nil {
			if err == io.EOF {
				if inContinuation {
					return fv.fatalError(contStartLine, "data", errors.CodeCSVParseError.Code,
						fmt.Errorf("data row continuation: unexpected EOF after trailing comma"), rowsScanned, "")
				}
				break
			}
			if err == csvstream.ErrFieldTooLarge {
				return fv.fatalError(stream.CurrentLine(), "rowErr", errors.CodeFieldTooLarge.Code, err, rowsScanned, "field too large")
			}
			return fv.fatalCSVParseError(err, stream.CurrentLine(), rowsScanned)
		}

		// Trailing comma detection for row continuation (9.1):
		trailing := rawRowHasTrailingComma(rawRow)

		// Detect annotation after trailing comma: spec forbids placing an inline
		// annotation after the trailing continuation comma on the same physical line
		// with no field value. E.g. "bob, ( comment )" is invalid; the annotation
		// must appear on the continuation line: "bob," then "( comment ) 34".
		if trailing && rawRow.Fields[len(rawRow.Fields)-1].HasOnlyAnnotations() {
			annErrs := []*errors.ValidationError{{
				Row:    rawRow.Line,
				Column: "rowErr",
				Code:   errors.CodeAnnotationAfterTrailingComma.Code,
				Msg:    "annotation after trailing comma: annotation must appear on the continuation line, not before it",
			}}
			if result := fv.appendErrorsOrStop(&validationErrors, annErrs, rawRow.Line, rowsScanned); result != nil {
				rawRow.Release()
				return result, nil
			}
		}

		if !inContinuation && !trailing {
			// -- COMMON PATH: single-line row, no continuation --
			buf := getFieldBuffers(rawRow)
			errs := rowValidator.ValidateRow(buf.fields, buf.quoted, buf.commentCounts, buf.metaCounts, rawRow.Line)
			releaseRowResources(buf, rawRow)

			if result := fv.appendErrorsOrStop(&validationErrors, errs, rawRow.Line, rowsScanned); result != nil {
				return result, nil
			}

			rowsScanned++
			continue
		}

		// -- CONTINUATION PATH (rare - multi-line data row per 9.1) --

		// Reject # comments mid-data-continuation (only ( ... ) is allowed)
		if inContinuation && commentLineSkipped {
			rawRow.Release()
			return fv.fatalError(contStartLine, "data", errors.CodeCSVParseError.Code,
				fmt.Errorf("'#' comments are not permitted within data rows"), rowsScanned, "")
		}

		// Track whether this is a mid-continuation line (second+ physical row).
		// Standalone annotations before the first physical row are row-level,
		// not field-level - only mid-continuation standalones count as field annotations.
		midContinuation := inContinuation

		if !inContinuation {
			contStartLine = rawRow.Line
			inContinuation = true
		}

		// Determine field count (strip trailing empty field produced by the trailing comma)
		fieldCount := len(rawRow.Fields)
		if trailing {
			fieldCount--
		}

		// Copy field bytes - chunk memory is invalidated on Release, so we must copy.
		// Allocations here are acceptable since multi-line rows are rare.
		for i := 0; i < fieldCount; i++ {
			src := rawRow.Fields[i].Bytes()
			dst := make([]byte, len(src))
			copy(dst, src)
			contFields = append(contFields, dst)
			contQuoted = append(contQuoted, rawRow.Fields[i].Quoted)
			cc, mc := rawRow.Fields[i].AnnotationCounts()
			// Standalone annotation lines between continuation rows
			// apply to the first field of that line. Only count when
			// mid-continuation (not for the first physical row, where
			// standalones are row-level annotations).
			if i == 0 && midContinuation {
				cc += standaloneComments
				mc += standaloneMeta
			}
			contCommentCounts = append(contCommentCounts, cc)
			contMetaCounts = append(contMetaCounts, mc)
		}

		rawRow.Release()

		if !trailing {
			// Last physical row of this logical row - validate the accumulated fields
			errs := rowValidator.ValidateRow(contFields, contQuoted, contCommentCounts, contMetaCounts, contStartLine)

			// Reset continuation state (reuse slices for next multi-line row)
			contFields = contFields[:0]
			contQuoted = contQuoted[:0]
			contCommentCounts = contCommentCounts[:0]
			contMetaCounts = contMetaCounts[:0]
			inContinuation = false

			if result := fv.appendErrorsOrStop(&validationErrors, errs, contStartLine, rowsScanned); result != nil {
				return result, nil
			}

			rowsScanned++
		}
	}

	// 6. Return result
	return &ValidationResult{
		Valid:       len(validationErrors) == 0,
		Errors:      validationErrors,
		RowsScanned: rowsScanned,
	}, nil
}

// ============================================================================
// Step 5 Helper Methods (orchestrator plumbing extracted for clarity)
// ============================================================================

// fatalCSVParseError returns a fatal validation result for CSV parse errors.
// Only called on malformed CSV (not hot path).
func (fv *FileValidator) fatalCSVParseError(err error, line, rowsScanned int) (*ValidationResult, error) {
	return fv.fatalError(line, "rowErr", errors.CodeCSVParseError.Code, err, rowsScanned, "CSV parse error")
}

// getFieldBuffers acquires a buffer from the pool and extracts fields from rawRow.
// Zero-copy for single-chunk fields. Called once per row (hot path - will inline).
func getFieldBuffers(rawRow *csvstream.RawRow) *fieldBuffers {
	buf := fieldBufferPool.Get().(*fieldBuffers)

	// Resize to match row width
	if cap(buf.fields) < len(rawRow.Fields) {
		buf.fields = make([][]byte, len(rawRow.Fields))
		buf.quoted = make([]bool, len(rawRow.Fields))
		buf.commentCounts = make([]int8, len(rawRow.Fields))
		buf.metaCounts = make([]int8, len(rawRow.Fields))
	} else {
		buf.fields = buf.fields[:len(rawRow.Fields)]
		buf.quoted = buf.quoted[:len(rawRow.Fields)]
		buf.commentCounts = buf.commentCounts[:len(rawRow.Fields)]
		buf.metaCounts = buf.metaCounts[:len(rawRow.Fields)]
	}

	// Extract fields (zero-copy for single-chunk fields)
	for i := range rawRow.Fields {
		buf.fields[i] = rawRow.Fields[i].Bytes()
		buf.quoted[i] = rawRow.Fields[i].Quoted
		buf.commentCounts[i], buf.metaCounts[i] = rawRow.Fields[i].AnnotationCounts()
	}

	return buf
}

// releaseRowResources releases both the field buffer and raw row.
// Lifecycle management in one place - impossible to forget either release.
// Called once per row (hot path - will inline).
func releaseRowResources(buf *fieldBuffers, row *csvstream.RawRow) {
	fieldBufferPool.Put(buf)
	row.Release()
}

// appendErrorsOrStop appends validation errors and stops if max error limit reached.
// Returns ValidationResult if limit exceeded (fatal error), nil otherwise.
// Called once per row with errors (error path when validation fails).
func (fv *FileValidator) appendErrorsOrStop(validationErrors *[]*errors.ValidationError, errs []*errors.ValidationError, rowLine, rowsScanned int) *ValidationResult {
	for _, e := range errs {
		*validationErrors = append(*validationErrors, e)

		// Stop if maxErrors reached (0 = unlimited)
		if fv.maxErrors > 0 && len(*validationErrors) >= fv.maxErrors {
			result, _ := fv.fatalError(rowLine, "ERRORS", errors.CodeMaxErrorsReached.Code,
				fmt.Errorf("validation stopped max errors limit reached"), rowsScanned+1, "")
			result.Errors = append(*validationErrors, result.Errors[0])
			return result
		}
	}
	return nil
}

// readHeaderWithContinuation reads the header row(s) with support for multi-line continuation.
// Per SuperCSV v1.0 6.2, headers can span multiple lines when a line ends with a trailing comma.
// Returns a headerdef.Row suitable for headerdef.ParseHeader, or an error.
func (fv *FileValidator) readHeaderWithContinuation(stream *csvstream.CSVStream) (*headerdef.Row, error) {
	schemaRow, err := headerdefreader.ReadHeaderWithContinuation(stream)
	if err != nil {
		return nil, err
	}
	stream.SetHeaderMode(false)
	return schemaRow, nil
}

// rawRowHasTrailingComma checks if a raw row ends with a trailing comma.
// A trailing comma is detected when the last field is unquoted and contains
// only whitespace (or is empty). This signals data row continuation per 9.1.
// Also returns true when the last field contains only annotation blocks and
// whitespace (annotation after trailing comma - a spec violation detected later).
// Called once per row - not hot path.
func rawRowHasTrailingComma(row *csvstream.RawRow) bool {
	n := len(row.Fields)
	if n == 0 {
		return false
	}
	if row.Fields[n-1].Quoted {
		return false
	}
	data := row.Fields[n-1].RawBytes()
	for _, b := range data {
		if !asciiWhitespace[b] {
			// Non-whitespace found: only treat as trailing comma if the entire
			// field content is annotation blocks with no actual value.
			return row.Fields[n-1].HasOnlyAnnotations()
		}
	}
	return true
}

// asciiWhitespace is a lookup table for ASCII whitespace characters.
var asciiWhitespace = [256]bool{
	' ':  true,
	'\t': true,
	'\r': true,
	'\n': true,
}
