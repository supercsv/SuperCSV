// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"fmt"
	"io"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// Header represents a SuperCSV header with column definitions.
// Header is immutable after construction and safe to share across goroutines.
type Header struct {
	// RawText is the exact header text from the file, including all continuation lines.
	// Preserves original formatting with continuations.
	// Empty if header was constructed programmatically.
	RawText string

	// CanonicalText is the normalized single-line header representation.
	// This is the deterministic serialization form used by encoders.
	// Always set (either parsed or generated).
	CanonicalText string

	// Columns are the column definitions in order.
	Columns []Column
}

// Column represents a single column definition.
type Column struct {
	Name string // Column name
	Type Type   // Column type
}

// String returns the canonical header text.
func (h *Header) String() string {
	return h.CanonicalText
}

// NumColumns returns the number of columns.
func (h *Header) NumColumns() int {
	return len(h.Columns)
}

// --- Header Constructors ---

// NewHeader creates a new Header from column definitions.
// Generates CanonicalText from the columns.
// RawText is empty (constructed programmatically).
//
// Example:
//
//	header := supr.NewHeader([]supr.Column{
//	    {Name: "id", Type: supr.Int},
//	    {Name: "name", Type: supr.String},
//	    {Name: "scores", Type: supr.List(supr.Float)},
//	})
func NewHeader(columns []Column) *Header {
	h := &Header{RawText: "", Columns: columns}

	// Generate canonical header text by creating a temporary encoder in default mode.
	// NewFileEncoder defaults to headerCanonical when no options are passed (encoder.go:95).
	// This reuses all existing type formatting logic (formatType, formatScalarType, etc.)
	enc := NewFileEncoder(h, io.Discard)
	h.CanonicalText = enc.serializeHeader()

	return h
}

// TODO Phase 2: ParseHeader will be implemented when wrapping the validator
// It requires the csvstream reader which is internal to the validator

// --- Batch Validation Convenience ---

// ValidateFile validates an entire SuperCSV file.
// Returns the header and any validation errors.
// This is a convenience wrapper around NewValidator for batch processing.
//
// For streaming validation (constant memory), use NewValidator instead.
//
// Example:
//
//	file, _ := os.Open("data.csv")
//	defer file.Close()
//	header, errs, err := supr.ValidateFile(file)
//	if err != nil {
//	    log.Fatal(err)  // I/O or parse error
//	}
//	for _, e := range errs {
//	    log.Println(e)  // Validation errors
//	}
func ValidateFile(r io.Reader, opts ...ValidatorOption) (*Header, []error, error) {
	v := NewValidator(r, opts...)
	var errors []error

	for v.Next() {
		if err := v.RowError(); err != nil {
			errors = append(errors, err)
		}
	}

	if err := v.Err(); err != nil {
		return nil, nil, err
	}

	return v.Header(), errors, nil
}

// --- Internal Conversion Helpers ---

// columnTypeFromInternal converts internal headerdef.Column to public Type.
func columnTypeFromInternal(col headerdef.Column) Type {
	return convertColumnType(col.Type)
}

func convertColumnType(ct headerdef.ColumnType) Type {
	switch ct.Kind {
	case headerdef.TypeKindScalar:
		// Check if it's an enum
		if ct.Scalar.Kind == headerdef.ScalarEnum && ct.Scalar.Enum != nil {
			values := make([]EnumValue, len(ct.Scalar.Enum.Values))
			for i, v := range ct.Scalar.Enum.Values {
				values[i] = EnumValue{
					Name:  v.Name,
					Value: v.Value,
				}
			}
			return Enum(values)
		}
		// Regular scalar
		return convertScalarType(ct.Scalar.Kind)

	case headerdef.TypeKindList:
		elemType := convertScalarType(ct.List.Element.Kind)
		if ct.List.Element.Kind == headerdef.ScalarEnum && ct.List.Element.Enum != nil {
			values := make([]EnumValue, len(ct.List.Element.Enum.Values))
			for i, v := range ct.List.Element.Enum.Values {
				values[i] = EnumValue{
					Name:  v.Name,
					Value: v.Value,
				}
			}
			elemType = Enum(values)
		}
		// Check for fixed-size list
		if ct.List.FixedLength != 0 {
			return ListFixed(elemType, ct.List.FixedLength)
		}
		return List(elemType)

	case headerdef.TypeKindArray:
		elemType := convertScalarType(ct.Array.Element.Kind)
		if ct.Array.Element.Kind == headerdef.ScalarEnum && ct.Array.Element.Enum != nil {
			values := make([]EnumValue, len(ct.Array.Element.Enum.Values))
			for i, v := range ct.Array.Element.Enum.Values {
				values[i] = EnumValue{
					Name:  v.Name,
					Value: v.Value,
				}
			}
			elemType = Enum(values)
		}

		switch ct.Array.Rank {
		case headerdef.ArrayRankFlexible:
			return Arr(elemType)
		case headerdef.ArrayRank1DFixed:
			return ArrFixed1D(elemType, ct.Array.Length)
		case headerdef.ArrayRank2DFixed:
			return ArrFixed2D(elemType, ct.Array.Rows, ct.Array.Cols)
		default:
			panic(fmt.Sprintf("unsupported array rank: %d", ct.Array.Rank))
		}

	default:
		panic(fmt.Sprintf("unknown TypeKind: %d", ct.Kind))
	}
}

func convertScalarType(sk headerdef.ScalarKind) Type {
	switch sk {
	case headerdef.ScalarString:
		return String
	case headerdef.ScalarInt:
		return Int
	case headerdef.ScalarFloat:
		return Float
	case headerdef.ScalarDecimal:
		return Decimal
	case headerdef.ScalarBool:
		return Bool
	case headerdef.ScalarBytesHex:
		return BytesHex
	case headerdef.ScalarBytesB64:
		return BytesB64
	case headerdef.ScalarDate:
		return Date
	case headerdef.ScalarTime:
		return Time
	case headerdef.ScalarTimestamp:
		return Timestamp
	case headerdef.ScalarDatetime:
		return Datetime
	case headerdef.ScalarDatetimeTZ:
		return DatetimeTZ
	case headerdef.ScalarDuration:
		return Duration
	case headerdef.ScalarTimezone:
		return Timezone
	case headerdef.ScalarUUID:
		return UUID
	case headerdef.ScalarEnum:
		// Enum type without spec - should not happen in practice
		panic("enum scalar without EnumSpec")
	default:
		panic(fmt.Sprintf("unknown ScalarKind: %d", sk))
	}
}
