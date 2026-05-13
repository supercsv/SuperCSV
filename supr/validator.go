// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"fmt"
	"io"

	csvstream "github.com/supercsv/supercsv/internal/v1_0/csvstream"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/headerdefreader"
	v1_0 "github.com/supercsv/supercsv/internal/v1_0/validate"
)

// Validator validates a SuperCSV file in streaming mode.
// Use Next() to advance through rows, RowError() to check each row's validation status.
//
// Validator instances are NOT safe for concurrent use.
// Each goroutine must create its own Validator with its own io.Reader.
//
// The Validator does NOT close the io.Reader. The caller is responsible for closing it.
//
// Example:
//
//	v := supr.NewValidator(r)
//	for v.Next() {
//	    if err := v.RowError(); err != nil {
//	        log.Printf("Row %d: %v", v.RowNum(), err)
//	    }
//	}
//	if err := v.Err(); err != nil {
//	    log.Fatal(err)  // I/O or parse error
//	}
//	fmt.Printf("Validated %d rows\n", v.RowNum())
type Validator struct {
	stream       *csvstream.CSVStream
	rowValidator *v1_0.RowValidator
	header       *Header
	rowErr       error
	fatalErr     error
	rowNum       int
	errorCount   int
	maxErrors    int
	anyErrors    bool // Track if any row errors occurred
}

// ValidatorOption configures a Validator.
type ValidatorOption func(*validatorConfig)

type validatorConfig struct {
	maxErrors     int
	maxFieldSize  int
	strictVersion string
}

// WithMaxErrors sets the maximum number of validation errors before stopping.
// Default: unlimited (validate all rows).
//
// When the limit is reached:
//   - Next() returns false
//   - Err() returns ErrMaxErrors
//
// Example:
//
//	v := supr.NewValidator(r, supr.WithMaxErrors(100))
func WithMaxErrors(n int) ValidatorOption {
	return func(c *validatorConfig) {
		c.maxErrors = n
	}
}

// WithMaxFieldSize sets the maximum allowed size for any single field in bytes.
// Default: unlimited.
//
// Fields exceeding this limit trigger ErrMaxFieldSize.
//
// Example:
//
//	v := supr.NewValidator(r, supr.WithMaxFieldSize(1024*1024)) // 1MB limit
func WithMaxFieldSize(n int) ValidatorOption {
	return func(c *validatorConfig) {
		c.maxFieldSize = n
	}
}

// WithStrictVersion enforces a required SuperCSV version directive.
// Default: no version check.
//
// Example:
//
//	v := supr.NewValidator(r, supr.WithStrictVersion("1.0"))
func WithStrictVersion(version string) ValidatorOption {
	return func(c *validatorConfig) {
		c.strictVersion = version
	}
}

// NewValidator creates a new streaming validator.
// The reader must be positioned at the start of the file (before the header).
//
// Options:
//   - WithMaxErrors(n): stop after n validation errors
//   - WithMaxFieldSize(n): reject fields larger than n bytes
//   - WithStrictVersion(v): require specific version directive
//
// Example:
//
//	v := supr.NewValidator(r, supr.WithMaxErrors(100))
func NewValidator(r io.Reader, opts ...ValidatorOption) *Validator {
	cfg := validatorConfig{
		maxErrors:     -1, // unlimited
		maxFieldSize:  0,  // unlimited
		strictVersion: "",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create CSV stream
	stream, err := csvstream.NewCSVStream(r, csvstream.StreamOptions{
		MaxFieldSize:  cfg.maxFieldSize,
		HeaderMode:    true, // Enable angle bracket tracking for header
		StrictVersion: cfg.strictVersion,
	})
	if err != nil {
		return &Validator{fatalErr: fmt.Errorf("failed to create CSV stream: %w", err)}
	}

	v := &Validator{
		stream:     stream,
		maxErrors:  cfg.maxErrors,
		errorCount: 0,
		anyErrors:  false,
	}

	// Parse header
	if err := v.parseHeader(cfg.maxFieldSize); err != nil {
		v.fatalErr = err
		return v
	}

	// Disable header mode for data rows
	stream.SetHeaderMode(false)

	return v
}

func (v *Validator) parseHeader(maxFieldSize int) error {
	// Read header row (may span multiple lines with continuations)
	schemaRow, err := headerdefreader.ReadHeaderWithContinuation(v.stream)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Parse header definition
	hdef, err := headerdef.ParseHeader(schemaRow)
	if err != nil {
		return fmt.Errorf("invalid header schema: %w", err)
	}

	// Convert to public Header type
	columns := make([]Column, len(hdef.Columns))
	for i, col := range hdef.Columns {
		columns[i] = Column{
			Name: col.Name,
			Type: columnTypeFromInternal(col),
		}
	}

	v.header = &Header{
		Columns: columns,
	}

	// Generate canonical text using the same logic as NewHeader and Decoder
	enc := NewFileEncoder(v.header, io.Discard)
	v.header.CanonicalText = enc.serializeHeader()

	// Create row validator
	v.rowValidator = v1_0.NewRowValidator(hdef, maxFieldSize)

	return nil
}

// Header returns the parsed header.
// Only valid after the first call to Next().
// Returns nil if the header hasn't been parsed yet or if there was a parse error.
func (v *Validator) Header() *Header {
	return v.header
}

// Next advances to the next row.
// Returns true if a row is available, false if EOF or error.
//
// After Next() returns true, call RowError() to check if the row is valid.
// After Next() returns false, call Err() to check for I/O or parse errors.
//
// Example:
//
//	for v.Next() {
//	    if err := v.RowError(); err != nil {
//	        log.Println(err)
//	    }
//	}
//	if err := v.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (v *Validator) Next() bool {
	// Check for fatal error from initialization
	if v.fatalErr != nil {
		return false
	}

	// Check if max errors reached
	if v.maxErrors > 0 && v.errorCount >= v.maxErrors {
		v.fatalErr = ErrMaxErrors
		return false
	}

	// Clear previous row error
	v.rowErr = nil

	// Read next row from stream
	rawRow, err := v.stream.NextRow()
	if err != nil {
		if err == io.EOF {
			return false // Normal EOF
		}
		v.fatalErr = fmt.Errorf("CSV parse error: %w", err)
		return false
	}

	v.rowNum++

	// Extract fields from raw row
	fields := make([][]byte, len(rawRow.Fields))
	quoted := make([]bool, len(rawRow.Fields))
	commentCounts := make([]int8, len(rawRow.Fields))
	metaCounts := make([]int8, len(rawRow.Fields))
	for i := range rawRow.Fields {
		fields[i] = rawRow.Fields[i].Bytes()
		quoted[i] = rawRow.Fields[i].Quoted
		commentCounts[i], metaCounts[i] = rawRow.Fields[i].AnnotationCounts()
	}

	// Validate row
	errs := v.rowValidator.ValidateRow(fields, quoted, commentCounts, metaCounts, rawRow.Line)

	// Release row resources
	rawRow.Release()

	// Store first error for this row
	if len(errs) > 0 {
		// Convert internal error to public error
		internalErr := errs[0]
		v.rowErr = &ValidationError{
			Row:    internalErr.Row,
			Col:    v.getColumnIndex(internalErr.Column) + 1, // 1-based
			Field:  string(fields[v.getColumnIndex(internalErr.Column)]),
			Reason: internalErr.Msg,
			Err:    ErrValidation,
		}
		v.errorCount++
		v.anyErrors = true
	}

	return true
}

func (v *Validator) getColumnIndex(colName string) int {
	if v.header == nil {
		return 0
	}
	for i, col := range v.header.Columns {
		if col.Name == colName {
			return i
		}
	}
	return 0
}

// RowError returns the validation error for the current row, or nil if valid.
// Only valid after Next() returns true.
//
// This does NOT advance to the next row. Call Next() to continue.
//
// Example:
//
//	for v.Next() {
//	    if err := v.RowError(); err != nil {
//	        log.Printf("Row %d invalid: %v", v.RowNum(), err)
//	    }
//	}
func (v *Validator) RowError() error {
	return v.rowErr
}

// Err returns the first I/O or parse error encountered, or nil.
// Only check this after Next() returns false.
//
// Returns:
//   - nil: EOF reached successfully
//   - ErrMaxErrors: stopped due to WithMaxErrors limit
//   - ErrInvalidHeader: header parse failed
//   - other: I/O or CSV parse error
//
// Example:
//
//	for v.Next() {
//	    // Process rows
//	}
//	if err := v.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (v *Validator) Err() error {
	return v.fatalErr
}

// RowNum returns the current row number (1-based).
// Returns 0 before the first call to Next().
// Increments even if the row is invalid.
//
// Example:
//
//	for v.Next() {
//	    fmt.Printf("Validating row %d\n", v.RowNum())
//	}
func (v *Validator) RowNum() int {
	return v.rowNum
}

// Valid returns true if no validation errors have been encountered.
// This includes both row-level errors and fatal errors.
// Returns false if any row failed validation or if a fatal error occurred.
//
// Example:
//
//	for v.Next() {
//	    if v.RowError() != nil {
//	        log.Println("Error:", v.RowError())
//	    }
//	}
//	if v.Err() == nil && v.Valid() {
//	    fmt.Println("All rows valid!")
//	}
func (v *Validator) Valid() bool {
	return !v.anyErrors && v.fatalErr == nil
}
