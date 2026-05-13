// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	validate "github.com/supercsv/supercsv/internal/v1_0/validate"
	"github.com/supercsv/supercsv/internal/v1_0/validate/dfa"
)

// DecodeMode specifies how to decode CSV rows
type DecodeMode int

const (
	// DecodeRaw returns raw []byte fields wrapped in []any
	// Bytes are copied to avoid aliasing with parser's buffer
	DecodeRaw DecodeMode = iota

	// DecodeShallow decodes scalars, returns containers as strings
	DecodeShallow

	// DecodeFull fully decodes both scalars and containers
	DecodeFull
)

// RowDecoder decodes CSV rows based on a schema
type RowDecoder struct {
	headerDef *headerdef.HeaderDef
	cols      []columnDecoder
	mode      DecodeMode
}

// columnDecoder holds a fully specialized decode function per column
// Built once during construction, invoked many times in hot path
type columnDecoder struct {
	decodeFn func([]byte) (any, error) // fully specialized, no branching
}

// NewRowDecoder creates a RowDecoder that decodes rows according to the schema
// Prebuild all decode functions here - hot path has zero branching
func NewRowDecoder(hdef *headerdef.HeaderDef, mode DecodeMode) *RowDecoder {
	cols := make([]columnDecoder, len(hdef.Columns))

	// Build fully specialized decode function per column
	// No branching in hot path - all decisions made here
	for i, col := range hdef.Columns {
		switch mode {
		case DecodeRaw:
			// Raw mode: copy bytes to avoid aliasing
			// Parser reuses backing buffer, so we must copy to stable memory
			cols[i].decodeFn = func(b []byte) (any, error) {
				dst := make([]byte, len(b))
				copy(dst, b)
				return dst, nil
			}

		case DecodeShallow:
			if col.Type.Kind == headerdef.TypeKindScalar {
				// Decode scalar - prebuild enum validator if needed
				var enumVal *dfa.EnumValidator
				if col.Type.Scalar.Kind == headerdef.ScalarEnum && col.Type.Scalar.Enum != nil {
					enumVal = dfa.NewEnumValidator(col.Type.Scalar.Enum)
				}
				cols[i].decodeFn = buildScalarDecodeFn(col.Type.Scalar, enumVal)
			} else {
				// Container as validated string
				cols[i].decodeFn = buildContainerValidateFn(col.Type)
			}

		case DecodeFull:
			if col.Type.Kind == headerdef.TypeKindScalar {
				// Decode scalar - prebuild enum validator if needed
				var enumVal *dfa.EnumValidator
				if col.Type.Scalar.Kind == headerdef.ScalarEnum && col.Type.Scalar.Enum != nil {
					enumVal = dfa.NewEnumValidator(col.Type.Scalar.Enum)
				}
				cols[i].decodeFn = buildScalarDecodeFn(col.Type.Scalar, enumVal)
			} else {
				// Decode container (list or array)
				cols[i].decodeFn = buildContainerDecodeFn(col.Type)
			}
		}
	}

	return &RowDecoder{
		headerDef: hdef,
		cols:      cols,
		mode:      mode,
	}
}

// DecodeRow decodes one row of CSV fields into []any using prebuilt decoders
// HOT PATH: Zero branching per field - all decisions made during construction
func (rd *RowDecoder) DecodeRow(fields [][]byte) ([]any, error) {
	// Validate field count
	if len(fields) != len(rd.cols) {
		return nil, fmt.Errorf("field count mismatch: expected %d, got %d",
			len(rd.cols), len(fields))
	}

	// Allocate output slice
	out := make([]any, len(rd.cols))

	// CRITICAL: Branch-free hot path
	// Each decodeFn is fully specialized - no type checking needed
	for i := range fields {
		var err error
		out[i], err = rd.cols[i].decodeFn(fields[i])
		if err != nil {
			colName := rd.headerDef.Columns[i].Name
			return nil, fmt.Errorf("column %s: %w", colName, err)
		}
	}

	return out, nil
}

// buildScalarDecodeFn returns a fully specialized scalar decoder
// CRITICAL: Enum validator captured in closure - zero heap allocation per row
func buildScalarDecodeFn(st headerdef.ScalarType, enumVal *dfa.EnumValidator) func([]byte) (any, error) {
	// Delegate to existing scalar decoder (handles all types including enums)
	return func(b []byte) (any, error) {
		return decodeScalarValue(b, st, enumVal)
	}
}

// buildContainerValidateFn returns a function that validates a container field and returns it as a string.
// Used by DecodeShallow: containers are validated but not fully decoded.
func buildContainerValidateFn(ct headerdef.ColumnType) func([]byte) (any, error) {
	// Prebuild enum validator if container elements are enums
	var enumVal *dfa.EnumValidator
	if ct.Kind == headerdef.TypeKindList && ct.List.Element.Kind == headerdef.ScalarEnum && ct.List.Element.Enum != nil {
		enumVal = dfa.NewEnumValidator(ct.List.Element.Enum)
	}
	if ct.Kind == headerdef.TypeKindArray && ct.Array.Element.Kind == headerdef.ScalarEnum && ct.Array.Element.Enum != nil {
		enumVal = dfa.NewEnumValidator(ct.Array.Element.Enum)
	}

	// Dispatch ONCE at construction based on container type
	if ct.Kind == headerdef.TypeKindList {
		elemType := ct.List.Element
		fixedLen := ct.List.FixedLength
		return func(b []byte) (any, error) {
			if isNullElement(b) {
				return nil, nil
			}
			if err := validate.StreamingListValidator(b, elemType, fixedLen, enumVal); err != nil {
				return nil, err
			}
			return string(b), nil
		}
	} else if ct.Kind == headerdef.TypeKindArray {
		schemaLength := ct.Array.Length
		schemaRows := ct.Array.Rows
		schemaCols := ct.Array.Cols
		prefer2D := ct.Array.Rank == headerdef.ArrayRank2DFixed
		return func(b []byte) (any, error) {
			if isNullElement(b) {
				return nil, nil
			}
			if err := validate.StreamingArrayValidator(b, ct.Array.Element, schemaLength, schemaRows, schemaCols, prefer2D, enumVal); err != nil {
				return nil, err
			}
			return string(b), nil
		}
	}
	panic("buildContainerValidateFn: unsupported container type")
}

// buildContainerDecodeFn returns a fully specialized container decoder
// CRITICAL: Dispatches list vs array at CONSTRUCTION, not in row loop
// The returned function has zero branching - it knows exactly what to decode
func buildContainerDecodeFn(ct headerdef.ColumnType) func([]byte) (any, error) {
	// Prebuild enum validator if container elements are enums
	var enumVal *dfa.EnumValidator
	if ct.Kind == headerdef.TypeKindList && ct.List.Element.Kind == headerdef.ScalarEnum && ct.List.Element.Enum != nil {
		enumVal = dfa.NewEnumValidator(ct.List.Element.Enum)
	}
	if ct.Kind == headerdef.TypeKindArray && ct.Array.Element.Kind == headerdef.ScalarEnum && ct.Array.Element.Enum != nil {
		enumVal = dfa.NewEnumValidator(ct.Array.Element.Enum)
	}

	// Dispatch ONCE at construction based on container type
	if ct.Kind == headerdef.TypeKindList {
		elemType := ct.List.Element
		fixedLen := ct.List.FixedLength
		// Return list decoder - no branching in returned function
		// NOTE: streamingListDecoder returns ([]any, error), so a nil result is a typed nil.
		// We must convert it to untyped nil so callers can do row[i] == nil safely.
		return func(b []byte) (any, error) {
			result, err := streamingListDecoder(b, elemType, fixedLen, enumVal)
			if result == nil {
				return nil, err
			}
			return result, err
		}
	} else if ct.Kind == headerdef.TypeKindArray {
		// Extract dimensions from schema (once at construction)
		rows := ct.Array.Rows
		cols := ct.Array.Cols
		elemType := ct.Array.Element
		is1D := ct.Array.Rank == headerdef.ArrayRank1DFixed
		// Return array decoder - no branching in returned function
		return func(b []byte) (any, error) {
			return streamingArrayDecoder(b, elemType, ct.Array.Length, rows, cols, is1D, enumVal)
		}
	}
	panic("buildContainerDecodeFn: unsupported container type")
}
