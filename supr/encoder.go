// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	v1_0 "github.com/supercsv/supercsv/internal/v1_0/encode"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// FileEncoder encodes SuperCSV files in canonical format.
// Writes header and data rows to an io.Writer in streaming mode.
//
// FileEncoder instances are NOT safe for concurrent use.
//
// The encoder does NOT close the io.Writer. The caller is responsible for closing it.
//
// Example:
//
//	e := supr.NewFileEncoder(header, w)
//	if err := e.WriteHeader(); err != nil {
//	    log.Fatal(err)
//	}
//	for _, row := range rows {
//	    if err := e.WriteRow(row); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//	if err := e.Close(); err != nil {
//	    log.Fatal(err)
//	}
type FileEncoder struct {
	header      *Header
	writer      *bufio.Writer
	headerMode  headerMode
	omitVersion bool
	headerDone  bool
	closed      bool
	rowsWritten int
	rowBuf      []byte // HOT_PATH - reusable row buffer
}

type headerMode int

const (
	headerCanonical headerMode = iota
	headerSmall
	headerTiny
)

// EncoderOption configures a FileEncoder.
type EncoderOption func(*encoderConfig)

type encoderConfig struct {
	mode        headerMode
	omitVersion bool // testing only: suppress version directive output
}

// WithSmallHeaderTypes uses compact type names in the header.
// Example: "i" instead of "int", "str" instead of "string"
//
// Example:
//
//	e := supr.NewFileEncoder(h, w, supr.WithSmallHeaderTypes())
func WithSmallHeaderTypes() EncoderOption {
	return func(c *encoderConfig) {
		c.mode = headerSmall
	}
}

// WithTinyHeaderTypes uses minimal type names in the header.
// Example: "i" instead of "int", "s" instead of "string"
//
// Example:
//
//	e := supr.NewFileEncoder(h, w, supr.WithTinyHeaderTypes())
func WithTinyHeaderTypes() EncoderOption {
	return func(c *encoderConfig) {
		c.mode = headerTiny
	}
}

// NewFileEncoder creates a new file encoder.
// The header is validated but not written until WriteHeader() is called.
//
// Options:
//   - WithSmallHeaderTypes(): use compact type names
//   - WithTinyHeaderTypes(): use minimal type names
//
// Example:
//
//	e := supr.NewFileEncoder(header, w)
func NewFileEncoder(h *Header, w io.Writer, opts ...EncoderOption) *FileEncoder {
	cfg := encoderConfig{
		mode: headerCanonical,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// HOT_PATH - Initialize buffer (4KB or 4*columns, whichever is larger)
	bufSize := 4096
	if len(h.Columns)*4 > bufSize {
		bufSize = len(h.Columns) * 4
	}

	return &FileEncoder{
		header:      h,
		writer:      bufio.NewWriter(w),
		headerMode:  cfg.mode,
		omitVersion: cfg.omitVersion,
		rowBuf:      make([]byte, 0, bufSize),
	}
}

// WriteHeader writes the header row to the output.
// Must be called before WriteRow().
// Can only be called once.
//
// Example:
//
//	if err := e.WriteHeader(); err != nil {
//	    log.Fatal(err)
//	}
func (e *FileEncoder) WriteHeader() error {
	if e.headerDone {
		return fmt.Errorf("header already written")
	}

	if !e.omitVersion {
		if _, err := e.writer.WriteString(v1_0.VersionDirectiveLine); err != nil {
			return fmt.Errorf("failed to write version directive: %w", err)
		}
	}

	headerLine := e.serializeHeader()
	if _, err := e.writer.WriteString(headerLine); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := e.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write header newline: %w", err)
	}

	e.headerDone = true
	return nil
}

// WriteRow writes a data row to the output.
// WriteHeader() must be called first.
//
// The row must:
//   - Have exactly len(Header.Columns) elements
//   - Match the types defined in the header
//   - Use nil for null values
//
// Type validation is always enabled - no WithStrictTypes option needed.
//
// Example:
//
//	row := []any{int64(42), "Alice", []any{int64(1), int64(2), int64(3)}}
//	if err := e.WriteRow(row); err != nil {
//	    log.Fatal(err)
//	}
func (e *FileEncoder) WriteRow(row []any) error {
	if e.closed {
		return fmt.Errorf("cannot write row: encoder is closed")
	}
	if !e.headerDone {
		return fmt.Errorf("WriteHeader() must be called before WriteRow()")
	}

	// Validate row length
	if len(row) != len(e.header.Columns) {
		return fmt.Errorf("row length mismatch: expected %d columns, got %d", len(e.header.Columns), len(row))
	}

	// HOT_PATH - Build entire row in buffer, then write once
	e.rowBuf = e.rowBuf[:0] // Reset buffer (keeps capacity)

	for i, val := range row {
		if i > 0 {
			e.rowBuf = append(e.rowBuf, ',')
		}

		// Convert public Type to internal ColumnType and encode
		colType := e.header.Columns[i].Type
		internalType := e.toInternalType(colType)

		var err error
		e.rowBuf, err = v1_0.AppendValue(e.rowBuf, val, internalType)
		if err != nil {
			return &EncodeError{Row: e.rowsWritten + 1, Col: i + 1, Err: err}
		}
	}

	// Write entire row + newline in one operation
	e.rowBuf = append(e.rowBuf, '\n')
	if _, err := e.writer.Write(e.rowBuf); err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	e.rowsWritten++
	return nil
}

// Close flushes any buffered data to the underlying writer.
// Does NOT close the underlying io.Writer.
// The caller is responsible for closing the writer.
//
// Example:
//
//	if err := e.Close(); err != nil {
//	    log.Fatal(err)
//	}
func (e *FileEncoder) Close() error {
	if e.closed {
		return nil // Idempotent - multiple Close() calls are safe
	}
	if err := e.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	e.closed = true
	return nil
}

// RowsWritten returns the number of rows written so far.
func (e *FileEncoder) RowsWritten() int {
	return e.rowsWritten
}

// serializeHeader generates the header line based on the header mode.
func (e *FileEncoder) serializeHeader() string {
	var sb strings.Builder
	for i, col := range e.header.Columns {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(col.Name)
		sb.WriteByte(':')
		sb.WriteString(e.formatType(col.Type))
	}
	return sb.String()
}

// formatType formats a type name according to the header mode.
func (e *FileEncoder) formatType(typ Type) string {
	switch typ.Kind() {
	case KindScalar:
		return e.formatScalarType(typ.ScalarKind())

	case KindEnum:
		spec := typ.EnumSpec()
		var sb strings.Builder
		keyword := e.formatContainerKeyword("enum")
		sb.WriteString(keyword)
		sb.WriteByte('<')
		for i, val := range spec.Values {
			if i > 0 {
				sb.WriteByte(',')
			}
			if val.Value != "" {
				// value=name style enum per spec
				sb.WriteString(val.Value)
				sb.WriteByte('=')
			}
			sb.WriteString(val.Name)
		}
		sb.WriteByte('>')
		return sb.String()

	case KindList:
		info := typ.Container()
		keyword := e.formatContainerKeyword("list")
		if info.Rank == RankListFixed {
			return fmt.Sprintf("%s<%s>[%d]", keyword, e.formatType(info.ElementType), info.Dim1)
		}
		return fmt.Sprintf("%s<%s>", keyword, e.formatType(info.ElementType))

	case KindArray1D:
		info := typ.Container()
		keyword := e.formatContainerKeyword("arr")
		return fmt.Sprintf("%s<%s>[%d]", keyword, e.formatType(info.ElementType), info.Dim1)

	case KindArray2D:
		info := typ.Container()
		keyword := e.formatContainerKeyword("arr")
		return fmt.Sprintf("%s<%s>[%d,%d]", keyword, e.formatType(info.ElementType), info.Dim1, info.Dim2)

	case KindArrayDynamic:
		info := typ.Container()
		keyword := e.formatContainerKeyword("arr")
		return fmt.Sprintf("%s<%s>", keyword, e.formatType(info.ElementType))

	default:
		return fmt.Sprintf("unknown<%d>", typ.Kind())
	}
}

// formatScalarType formats a scalar type name using headerdef lookup tables.
func (e *FileEncoder) formatScalarType(kind ScalarKind) string {
	// Convert supr.ScalarKind to headerdef.ScalarKind
	internalKind := e.toInternalScalarKind(kind)

	// Use headerdef functions for correct alias mappings
	switch e.headerMode {
	case headerSmall:
		return headerdef.FormatScalarSmall(internalKind)
	case headerTiny:
		return headerdef.FormatScalarTiny(internalKind)
	default:
		return headerdef.FormatScalarCanonical(internalKind)
	}
}

// formatContainerKeyword formats container keywords (list, arr, enum) according to header mode.
func (e *FileEncoder) formatContainerKeyword(canonical string) string {
	switch e.headerMode {
	case headerSmall:
		return headerdef.FormatContainerKeywordSmall(canonical)
	case headerTiny:
		return headerdef.FormatContainerKeywordTiny(canonical)
	default:
		return canonical
	}
}

// toInternalType converts public Type to internal headerdef.ColumnType
func (e *FileEncoder) toInternalType(t Type) headerdef.ColumnType {
	switch t.Kind() {
	case KindScalar:
		return headerdef.ColumnType{
			Kind: headerdef.TypeKindScalar,
			Scalar: headerdef.ScalarType{
				Kind: e.toInternalScalarKind(t.ScalarKind()),
			},
		}

	case KindEnum:
		spec := t.EnumSpec()
		// Convert public EnumSpec to internal
		values := make([]headerdef.EnumValue, len(spec.Values))
		for i, v := range spec.Values {
			values[i] = headerdef.EnumValue{
				Name:  v.Name,
				Value: v.Value,
			}
		}

		return headerdef.ColumnType{
			Kind: headerdef.TypeKindScalar,
			Scalar: headerdef.ScalarType{
				Kind: headerdef.ScalarEnum,
				Enum: &headerdef.EnumSpec{
					Values: values,
				},
			},
		}

	case KindList:
		info := t.Container()
		elemType := e.toInternalType(info.ElementType)
		return headerdef.ColumnType{
			Kind: headerdef.TypeKindList,
			List: headerdef.ListType{
				Element: elemType.Scalar,
			},
		}

	case KindArray1D:
		info := t.Container()
		elemType := e.toInternalType(info.ElementType)
		return headerdef.ColumnType{
			Kind: headerdef.TypeKindArray,
			Array: headerdef.ArrayType{
				Element: elemType.Scalar,
				Rank:    headerdef.ArrayRank1DFixed,
				Length:  info.Dim1,
			},
		}

	case KindArray2D:
		info := t.Container()
		elemType := e.toInternalType(info.ElementType)
		return headerdef.ColumnType{
			Kind: headerdef.TypeKindArray,
			Array: headerdef.ArrayType{
				Element: elemType.Scalar,
				Rank:    headerdef.ArrayRank2DFixed,
				Rows:    info.Dim1,
				Cols:    info.Dim2,
			},
		}

	case KindArrayDynamic:
		info := t.Container()
		elemType := e.toInternalType(info.ElementType)
		return headerdef.ColumnType{
			Kind: headerdef.TypeKindArray,
			Array: headerdef.ArrayType{
				Element: elemType.Scalar,
				Rank:    headerdef.ArrayRankFlexible,
			},
		}

	default:
		// Should never happen
		panic(fmt.Sprintf("unknown type kind: %d", t.Kind()))
	}
}

// toInternalScalarKind converts public ScalarKind to internal
func (e *FileEncoder) toInternalScalarKind(kind ScalarKind) headerdef.ScalarKind {
	switch kind {
	case KindString:
		return headerdef.ScalarString
	case KindInt:
		return headerdef.ScalarInt
	case KindFloat:
		return headerdef.ScalarFloat
	case KindDecimal:
		return headerdef.ScalarDecimal
	case KindBool:
		return headerdef.ScalarBool
	case KindBytesHex:
		return headerdef.ScalarBytesHex
	case KindBytesB64:
		return headerdef.ScalarBytesB64
	case KindDate:
		return headerdef.ScalarDate
	case KindTime:
		return headerdef.ScalarTime
	case KindTimestamp:
		return headerdef.ScalarTimestamp
	case KindDatetime:
		return headerdef.ScalarDatetime
	case KindDatetimeTZ:
		return headerdef.ScalarDatetimeTZ
	case KindDuration:
		return headerdef.ScalarDuration
	case KindTimezone:
		return headerdef.ScalarTimezone
	case KindUUID:
		return headerdef.ScalarUUID
	default:
		panic(fmt.Sprintf("unknown scalar kind: %d", kind))
	}
}

// EncodeFile encodes a complete File to a writer.
// This is a convenience function for batch encoding.
//
// Example:
//
//	file := supr.File{Header: header, Rows: rows}
//	if err := supr.EncodeFile(w, file); err != nil {
//	    log.Fatal(err)
//	}
func EncodeFile(w io.Writer, file *File, opts ...EncoderOption) error {
	e := NewFileEncoder(file.Header, w, opts...)

	if err := e.WriteHeader(); err != nil {
		return err
	}

	for _, row := range file.Rows {
		if err := e.WriteRow(row); err != nil {
			return err
		}
	}

	return e.Close()
}
