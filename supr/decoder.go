// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"fmt"
	"io"
	"os"

	csvstream "github.com/supercsv/supercsv/internal/v1_0/csvstream"
	v1_0 "github.com/supercsv/supercsv/internal/v1_0/decode"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/headerdefreader"
)

// Decoder decodes a SuperCSV file in streaming mode.
// Use Next() to advance through rows, Row() to get the decoded row data.
//
// Decoder instances are NOT safe for concurrent use.
// Each goroutine must create its own Decoder with its own io.Reader.
//
// The Decoder does NOT close the io.Reader. The caller is responsible for closing it.
//
// Row Ownership Contract:
//   - The slice returned by Row() is valid only until the next call to Next()
//   - The decoder reuses the row buffer across calls
//   - To preserve row data, copy the slice: slices.Clone(d.Row())
//
// Example:
//
//	d := supr.NewDecoder(r)
//	for d.Next() {
//	    row := d.Row()
//	    // Process row immediately (don't store, buffer reused)
//	    processRow(row)
//	}
//	if err := d.Err(); err != nil {
//	    log.Fatal(err)
//	}
type Decoder struct {
	stream     *csvstream.CSVStream
	rowDecoder *v1_0.RowDecoder
	header     *Header
	rowBuf     []any
	fatalErr   error
	rowNum     int
}

// DecodeMode controls allocation and validation behavior.
type DecodeMode int

const (
	// DecodeRaw: Raw bytes, fresh allocations, strict validation.
	// Row() returns []any with []byte values.
	// Shallow copy sufficient: slices.Clone(d.Row())
	DecodeRaw DecodeMode = iota

	// DecodeShallow: Scalars decoded, containers as strings, relaxed validation.
	// Scalars are Go types, containers remain strings.
	// Shallow copy sufficient: slices.Clone(d.Row())
	DecodeShallow

	// DecodeFull: Full decode, fresh allocations per container.
	// All types fully decoded including containers.
	// Copy sufficient: slices.Clone(d.Row())
	DecodeFull
)

// DecoderOption configures a Decoder.
type DecoderOption func(*decoderConfig)

type decoderConfig struct {
	maxFieldSize  int
	strictVersion string
	mode          DecodeMode
}

// WithDecodeMaxFieldSize sets the maximum allowed size for any single field in bytes.
// Default: unlimited.
//
// Example:
//
//	d := supr.NewDecoder(r, supr.WithDecodeMaxFieldSize(1024*1024)) // 1MB limit
func WithDecodeMaxFieldSize(n int) DecoderOption {
	return func(c *decoderConfig) {
		c.maxFieldSize = n
	}
}

// WithDecodeStrictVersion enforces a required SuperCSV version directive.
// Default: no version check.
//
// Example:
//
//	d := supr.NewDecoder(r, supr.WithDecodeStrictVersion("1.0"))
func WithDecodeStrictVersion(version string) DecoderOption {
	return func(c *decoderConfig) {
		c.strictVersion = version
	}
}

// WithDecodeMode sets the decode mode.
// Default: DecodeFull
//
// Modes:
//   - DecodeRaw: Raw bytes ([]byte), fresh allocations
//   - DecodeShallow: Scalars decoded, containers as strings
//   - DecodeFull: Full decode, fresh allocations per container
//
// Example:
//
//	d := supr.NewDecoder(r, supr.WithDecodeMode(supr.DecodeShallow))
func WithDecodeMode(mode DecodeMode) DecoderOption {
	return func(c *decoderConfig) {
		c.mode = mode
	}
}

// NewDecoder creates a new streaming decoder.
// The reader must be positioned at the start of the file (before the header).
//
// Options:
//   - WithDecodeMaxFieldSize(n): reject fields larger than n bytes
//   - WithDecodeStrictVersion(v): require specific version directive
//   - WithDecodeMode(mode): set decode mode (Raw/Shallow/Full)
//
// Example:
//
//	d := supr.NewDecoder(r, supr.WithDecodeMode(supr.DecodeFull))
func NewDecoder(r io.Reader, opts ...DecoderOption) *Decoder {
	cfg := decoderConfig{
		maxFieldSize:  0,          // unlimited
		strictVersion: "",         // no version check
		mode:          DecodeFull, // default to full decode
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
		return &Decoder{fatalErr: fmt.Errorf("failed to create CSV stream: %w", err)}
	}

	d := &Decoder{
		stream: stream,
	}

	// Parse header
	if err := d.parseHeader(cfg.mode); err != nil {
		d.fatalErr = err
		return d
	}

	// Disable header mode for data rows
	stream.SetHeaderMode(false)

	// Allocate row buffer (reused across calls)
	d.rowBuf = make([]any, len(d.header.Columns))

	return d
}

func (d *Decoder) parseHeader(mode DecodeMode) error {
	// Read header row (may span multiple lines with continuations)
	schemaRow, err := headerdefreader.ReadHeaderWithContinuation(d.stream)
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

	// Construct header and generate CanonicalText using encoder logic
	// This ensures symmetric encode/decode behavior and stable round-trip tests
	d.header = &Header{
		RawText:       "",
		CanonicalText: "",
		Columns:       columns,
	}

	// Generate CanonicalText using the same logic as NewHeader
	// Create a temporary encoder in default mode (headerCanonical)
	enc := NewFileEncoder(d.header, io.Discard)
	d.header.CanonicalText = enc.serializeHeader()

	// Convert decode mode to internal mode
	var internalMode v1_0.DecodeMode
	switch mode {
	case DecodeRaw:
		internalMode = v1_0.DecodeRaw
	case DecodeShallow:
		internalMode = v1_0.DecodeShallow
	case DecodeFull:
		internalMode = v1_0.DecodeFull
	default:
		return fmt.Errorf("invalid decode mode: %d", mode)
	}

	// Create row decoder
	d.rowDecoder = v1_0.NewRowDecoder(hdef, internalMode)

	return nil
}

// Header returns the parsed header.
// Available after NewDecoder returns (before first Next() call).
// Returns nil if the header couldn't be parsed.
func (d *Decoder) Header() *Header {
	return d.header
}

// Next advances to the next row and decodes it into the row buffer.
// Returns true if a row is available, false if EOF or error.
//
// After Next() returns true, call Row() to get the decoded data.
// After Next() returns false, call Err() to check for errors.
//
// Example:
//
//	for d.Next() {
//	    row := d.Row()
//	    // Process row
//	}
//	if err := d.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (d *Decoder) Next() bool {
	// Check for fatal error from initialization
	if d.fatalErr != nil {
		return false
	}

	// Read next row from stream
	rawRow, err := d.stream.NextRow()
	if err != nil {
		if err == io.EOF {
			return false // Normal EOF
		}
		d.fatalErr = fmt.Errorf("CSV parse error: %w", err)
		return false
	}

	d.rowNum++

	// Extract fields from raw row
	fields := make([][]byte, len(rawRow.Fields))
	for i := range rawRow.Fields {
		fields[i] = rawRow.Fields[i].Bytes()
	}

	// Decode row into buffer
	decoded, err := d.rowDecoder.DecodeRow(fields)

	// Capture line number before releasing row resources
	rowLine := rawRow.Line

	// Release row resources
	rawRow.Release()

	if err != nil {
		d.fatalErr = &DecodeError{Row: rowLine, Err: err}
		return false
	}

	// Copy decoded values into row buffer
	copy(d.rowBuf, decoded)

	return true
}

// Row returns the decoded row data.
// Only valid after Next() returns true.
//
// WARNING: The returned slice is valid only until the next call to Next().
// The decoder reuses the row buffer across calls.
// To preserve row data, copy the slice: slices.Clone(d.Row())
//
// Example (streaming, no copy):
//
//	for d.Next() {
//	    row := d.Row()
//	    processRow(row) // Use immediately, don't store
//	}
//
// Example (accumulate with copy):
//
//	var rows [][]any
//	for d.Next() {
//	    rows = append(rows, slices.Clone(d.Row()))
//	}
func (d *Decoder) Row() []any {
	return d.rowBuf
}

// Err returns the first fatal error encountered, or nil.
// Only check this after Next() returns false.
//
// Fatal errors include:
//   - CSV parse errors (malformed CSV)
//   - Decode errors (type conversion failures)
//   - I/O errors
//
// Returns nil if EOF was reached successfully.
//
// Example:
//
//	for d.Next() {
//	    // Process rows
//	}
//	if err := d.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (d *Decoder) Err() error {
	return d.fatalErr
}

// RowNum returns the current row number (1-based).
// Returns 0 before the first call to Next().
//
// Example:
//
//	for d.Next() {
//	    fmt.Printf("Decoding row %d\n", d.RowNum())
//	}
func (d *Decoder) RowNum() int {
	return d.rowNum
}

// DecodeFile decodes the entire file into memory.
// This is a convenience method for batch processing.
//
// For streaming (constant memory), use NewDecoder and iterate with Next()/Row().
//
// Example:
//
//	d := supr.NewDecoder(r)
//	file, err := d.DecodeFile()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, row := range file.Rows {
//	    // Process row (safe to store)
//	}
func (d *Decoder) DecodeFile() (*File, error) {
	var rows [][]any

	for d.Next() {
		rowCopy := make([]any, len(d.rowBuf))
		copy(rowCopy, d.rowBuf)

		// Deep-copy container slices so returned rows are fully independent
		for i, v := range rowCopy {
			switch s := v.(type) {
			case []any:
				if s == nil {
					rowCopy[i] = nil // typed-nil []any -> untyped nil; nil == nil must hold for callers
					break
				}
				c := make([]any, len(s))
				copy(c, s)
				rowCopy[i] = c
			case [][]any:
				if s == nil {
					rowCopy[i] = nil // typed-nil [][]any -> untyped nil
					break
				}
				c := make([][]any, len(s))
				for j := range s {
					inner := make([]any, len(s[j]))
					copy(inner, s[j])
					c[j] = inner
				}
				rowCopy[i] = c
			}
		}

		rows = append(rows, rowCopy)
	}

	if err := d.Err(); err != nil {
		return nil, err
	}

	return &File{
		Header: d.header,
		Rows:   rows,
	}, nil
}

// DecodeFile decodes a SuperCSV file from a path.
// This is a convenience function for batch processing.
//
// Example:
//
//	file, err := supr.DecodeFile("data.csv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, row := range file.Rows {
//	    // Process row
//	}
func DecodeFile(path string, opts ...DecoderOption) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	d := NewDecoder(f, opts...)
	return d.DecodeFile()
}
