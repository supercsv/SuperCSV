// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"
	"io"
	"os"

	csvstream "github.com/supercsv/supercsv/internal/v1_0/csvstream"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/headerdefreader"
)

// FileDecoder orchestrates file-level CSV decoding.
// It coordinates CSV parsing, schema parsing, and row-by-row data decoding.
type FileDecoder struct {
	maxFieldSize int
	mode         DecodeMode
}

// DecodedFile captures the outcome of decoding a CSV file.
type DecodedFile struct {
	HeaderDef   *headerdef.HeaderDef
	Rows        [][]any
	RowsDecoded int
}

// fieldBuffers holds field slices for field extraction.
type fieldBuffers struct {
	fields [][]byte
}

// NewFileDecoder creates a new file decoder with the specified configuration.
// maxFieldSize controls the maximum size of any single CSV field (0 = unlimited).
// mode specifies the decode mode (Basic, Easy, or Performant).
func NewFileDecoder(maxFieldSize int, mode DecodeMode) *FileDecoder {
	return &FileDecoder{
		maxFieldSize: maxFieldSize,
		mode:         mode,
	}
}

// getFieldBuffers extracts fields from rawRow into a buffer.
// Mirrors FileValidator's getFieldBuffers, but decoder doesn't need pooling (simpler).
func getFieldBuffers(rawRow *csvstream.RawRow) *fieldBuffers {
	fields := make([][]byte, len(rawRow.Fields))
	for i := range rawRow.Fields {
		fields[i] = rawRow.Fields[i].Bytes()
	}
	return &fieldBuffers{fields: fields}
}

// releaseRowResources releases the raw row.
// Mirrors FileValidator's resource management.
func releaseRowResources(buf *fieldBuffers, row *csvstream.RawRow) {
	row.Release()
}

// readHeaderWithContinuation reads multi-line headers.
// Delegates to the shared internal header reader.
// Per SuperCSV v1.0 6.2, headers can span multiple lines with trailing comma.
// Returns *headerdef.Row for headerdef.ParseHeader.
func (fd *FileDecoder) readHeaderWithContinuation(stream *csvstream.CSVStream) (*headerdef.Row, error) {
	schemaRow, err := headerdefreader.ReadHeaderWithContinuation(stream)
	if err != nil {
		return nil, err
	}
	stream.SetHeaderMode(false)
	return schemaRow, nil
}

// DecodeFile decodes a SuperCSV file and returns all decoded rows.
// It performs schema parsing on the header and type-aware decoding on data rows.
// Returns DecodedFile with decoded data, or a Go error for I/O/parse failures.
func (fd *FileDecoder) DecodeFile(path string) (*DecodedFile, error) {
	// 1. Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// 2. Create CSV stream parser with header mode (mirrors FileValidator)
	stream, err := csvstream.NewCSVStream(f, csvstream.StreamOptions{
		MaxFieldSize:  fd.maxFieldSize,
		HeaderMode:    true, // Enable angle bracket tracking for type declarations
		StrictVersion: "",   // Decoder doesn't enforce version
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV stream: %w", err)
	}

	// 3. Read header (with continuation support)
	schemaRow, err := fd.readHeaderWithContinuation(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Parse schema from the header DTO.
	hdef, err := headerdef.ParseHeader(schemaRow)
	if err != nil {
		return nil, fmt.Errorf("invalid header schema at line %d: %w", schemaRow.Line, err)
	}

	// 4. Create row decoder
	rowDecoder := NewRowDecoder(hdef, fd.mode)

	// 5. Decode rows (mirrors FileValidator's loop)
	var rows [][]any
	rowsDecoded := 0

	for {
		rawRow, err := stream.NextRow()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("CSV parse error at line %d: %w",
				stream.CurrentLine(), err)
		}

		// Extract fields from rawRow (mirrors getFieldBuffers pattern)
		buf := getFieldBuffers(rawRow)

		// Decode row (no quoted parameter - Phase 3 doesn't need it)
		decoded, err := rowDecoder.DecodeRow(buf.fields)
		if err != nil {
			releaseRowResources(buf, rawRow)
			return nil, fmt.Errorf("decode error at line %d: %w",
				stream.CurrentLine(), err)
		}

		rows = append(rows, decoded)
		releaseRowResources(buf, rawRow)
		rowsDecoded++
	}

	// 6. Return result
	return &DecodedFile{
		HeaderDef:   hdef,
		Rows:        rows,
		RowsDecoded: rowsDecoded,
	}, nil
}
