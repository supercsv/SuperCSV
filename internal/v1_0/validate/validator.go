// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"bytes"
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/byteclass"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/validate/dfa"
)

// Character classification flags for combined lookup table
const (
	charWhitespace    = byteclass.CharWhitespace
	charForbidden     = byteclass.CharForbidden
	charContinuation  = byteclass.CharContinuation
	charUnicodeWSLead = byteclass.CharUnicodeWSLead
)

// isUnicodeEdgeWS delegates to the shared byteclass edge-invalid lookup.
func isUnicodeEdgeWS(b []byte) bool {
	return byteclass.IsUnicodeEdgeWS(b)
}

// validateUnquotedString validates an unquoted string field.
// Rules (per spec v1.0 Section 8.1 / Whitespace section):
//   - Only ASCII whitespace (0x20 space, 0x09 tab, 0x0A LF, 0x0D CR) is trimmed
//     from leading and trailing edges.
//   - Unicode whitespace (NBSP, EM SPACE, IDEOGRAPHIC SPACE, ...) or other control
//     characters (\v 0x0B, \f 0x0C) at field edges are a validation error; the
//     caller must use a quoted string to carry such content.
//   - No forbidden characters inside the value: , # " [ ] ( ) < > { } ' ` ; : = ? / \ | @ \n \r
//   - Empty or whitespace-only values are invalid (must use _ or "")
//
// Returns the ASCII-trimmed value.
// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
func validateUnquotedString(data []byte) ([]byte, error) {
	// Trim only ASCII whitespace from edges (space, tab, LF, CR).
	trimmed := trimASCIISpace(data)

	// Per spec: "Whitespace-only unquoted values are invalid"
	// Per spec: "The empty field is _ (null literal)"
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty unquoted string (use _ or \"\")")
	}

	// Single forward pass: check forbidden chars and track last rune's lead byte.
	// lastLead piggybacks on the byte already loaded for charClass - no extra work.
	// Continuation bytes (0x80-0xBF) have b&0xC0 == 0x80; all other bytes start
	// a new rune (ASCII or multi-byte lead).
	// Single forward pass: check forbidden chars and track last rune's lead byte.
	// charContinuation and charUnicodeWSLead are folded into charClass so we get
	// all three tests from the single byte already loaded for charForbidden -
	// no separate table lookup, no b&0xC0 arithmetic.
	lastLead := 0
	for i := 0; i < len(trimmed); i++ {
		c := byteclass.CharClass[trimmed[i]]
		if c&charForbidden != 0 {
			return nil, fmt.Errorf("invalid character in unquoted string: %q", trimmed[i])
		}
		if c&charContinuation == 0 {
			lastLead = i
		}
	}

	// Edge unicode whitespace check: charUnicodeWSLead is in charClass, already
	// loaded above for index 0. For lastLead we do one more lookup (cheap).
	// isUnicodeEdgeWS uses direct byte comparisons - no rune decoding.
	if byteclass.CharClass[trimmed[0]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(trimmed) {
		return nil, fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}
	if byteclass.CharClass[trimmed[lastLead]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(trimmed[lastLead:]) {
		return nil, fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}

	return trimmed, nil
}

// validateQuotedString validates a quoted string field.
// Rules:
// - Must start and end with quotes
// - Unescape doubled quotes ("" -> ")
// - Preserve internal whitespace
// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
func validateQuotedString(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("invalid structure: quoted string too short")
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return nil, fmt.Errorf("invalid structure: missing quotes")
	}

	// Extract content between quotes
	content := data[1 : len(data)-1]

	// Check for invalid quote escaping (single quotes not at boundaries)
	for i := 0; i < len(content); i++ {
		if content[i] == '"' {
			// Must be part of a doubled quote pair
			if i+1 < len(content) && content[i+1] == '"' {
				i++ // Skip the second quote
			} else {
				return nil, fmt.Errorf("invalid structure: stray quote")
			}
		}
	}

	// Unescape doubled quotes
	if bytes.IndexByte(content, '"') >= 0 {
		return unescapeDoubledQuotes(content), nil
	}

	return content, nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateScalarWithDFA validates a cleaned scalar value using pure validation functions.
// The value must already be cleaned (trimmed/unescaped) before calling.
// For enums, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func validateScalarWithDFA(cleaned []byte, scalarType headerdef.ScalarType, enumValidator *dfa.EnumValidator) error {
	switch scalarType.Kind {
	case headerdef.ScalarString:
		// Strings already validated by quote parsing layer
		// By the time we reach this function, string validation is complete
		return nil

	case headerdef.ScalarEnum:
		// Enums validated by pre-built EnumValidator (passed as parameter)
		if enumValidator == nil {
			return fmt.Errorf("internal error: enum validator not provided")
		}
		if !enumValidator.Validate(cleaned) {
			return fmt.Errorf("%s", dfa.EnumError())
		}
		return nil

	case headerdef.ScalarBool:
		if !dfa.ValidateBool(cleaned) {
			return fmt.Errorf("%s", dfa.BoolError())
		}
	case headerdef.ScalarInt:
		if !dfa.ValidateInt(cleaned) {
			return fmt.Errorf("%s", dfa.IntErrorFor(cleaned))
		}
	case headerdef.ScalarFloat:
		if !dfa.ValidateFloat(cleaned) {
			return fmt.Errorf("%s", dfa.FloatError())
		}
	case headerdef.ScalarDecimal:
		if !dfa.ValidateDecimal(cleaned) {
			return fmt.Errorf("%s", dfa.DecimalError())
		}
	case headerdef.ScalarDate:
		if !dfa.ValidateDate(cleaned) {
			return fmt.Errorf("%s", dfa.DateError())
		}
	case headerdef.ScalarTime:
		if !dfa.ValidateTime(cleaned) {
			return fmt.Errorf("%s", dfa.TimeError())
		}
	case headerdef.ScalarTimestamp:
		if !dfa.ValidateTimestamp(cleaned) {
			return fmt.Errorf("%s", dfa.TimestampError())
		}
	case headerdef.ScalarBytesHex:
		if !dfa.ValidateBytesHex(cleaned) {
			return fmt.Errorf("%s", dfa.BytesHexError())
		}
	case headerdef.ScalarBytesB64:
		if !dfa.ValidateBytesB64(cleaned) {
			return fmt.Errorf("%s", dfa.BytesB64Error())
		}
	case headerdef.ScalarUUID:
		if !dfa.ValidateUUID(cleaned) {
			return fmt.Errorf("%s", dfa.UUIDError())
		}
	case headerdef.ScalarDuration:
		if !dfa.ValidateDuration(cleaned) {
			return fmt.Errorf("%s", dfa.DurationError())
		}
	case headerdef.ScalarTimezone:
		if !dfa.ValidateTimezone(cleaned) {
			return fmt.Errorf("%s", dfa.TimezoneError())
		}
	default:
		return fmt.Errorf("unknown scalar type: %v", scalarType.Kind)
	}
	return nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateUnquotedScalar trims leading/trailing whitespace and rejects internal whitespace.
// Exception: string and temporal scalars (datetime, datetimetz, timestamp) allow internal whitespace.
// Temporal types use space as valid separator in ISO 8601 format.
// String types inherently allow any content.
func validateUnquotedScalar(data []byte, scalarKind headerdef.ScalarKind) ([]byte, error) {
	// Trim whitespace using fast ASCII-only trim
	cleaned := trimASCIISpace(data)

	// String and temporal types allow internal whitespace
	if scalarKind != headerdef.ScalarString &&
		scalarKind != headerdef.ScalarDatetime &&
		scalarKind != headerdef.ScalarDatetimeTZ &&
		scalarKind != headerdef.ScalarTimestamp &&
		hasInternalWhitespace(cleaned) {
		return nil, fmt.Errorf("unquoted scalar has internal whitespace")
	}

	return cleaned, nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// hasInternalWhitespace checks if data contains any whitespace.
// For scalars, any whitespace is considered "internal" after trimming.
// Uses combined lookup table for fast ASCII whitespace detection.
func hasInternalWhitespace(data []byte) bool {
	for _, b := range data {
		if byteclass.CharClass[b]&charWhitespace != 0 {
			return true
		}
	}
	return false
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// trimASCIISpace trims ASCII whitespace (space, tab, \n, \r) from both ends.
// Delegates to the shared byteclass trim helper.
func trimASCIISpace(data []byte) []byte {
	return byteclass.TrimASCIISpace(data)
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// unescapeDoubledQuotes replaces "" with " in CSV quoted field content.
// This is the ONLY escape mechanism in CSV (not backslash).
func unescapeDoubledQuotes(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] == '"' && i+1 < len(data) && data[i+1] == '"' {
			result = append(result, '"')
			i += 2
		} else {
			result = append(result, data[i])
			i++
		}
	}
	return result
}

// isNumericType returns true if the scalar type must not be quoted per spec section 8.2.
// Numeric types include: int, float, decimal, bool, bytes, uuid.
func isNumericType(kind headerdef.ScalarKind) bool {
	return kind == headerdef.ScalarInt ||
		kind == headerdef.ScalarFloat ||
		kind == headerdef.ScalarDecimal ||
		kind == headerdef.ScalarBool ||
		kind == headerdef.ScalarBytesHex ||
		kind == headerdef.ScalarBytesB64 ||
		kind == headerdef.ScalarUUID
}

// validateListValue parses and validates a list literal with element type validation.
// Returns detailed error if parsing or element validation fails.
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func validateListValue(data []byte, elementType headerdef.ScalarType, fixedLength int, row int, columnName string, enumValidator *dfa.EnumValidator) error {
	// Use streaming validator - zero allocations, validates in single pass
	return StreamingListValidator(data, elementType, fixedLength, enumValidator)
}

// validateArrayValue parses and validates an array literal with element type validation.
// Returns detailed error if parsing or element validation fails.
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func validateArrayValue(data []byte, elementType headerdef.ScalarType, schemaLength, schemaRows, schemaCols int, row int, columnName string, prefer2D bool, enumValidator *dfa.EnumValidator) error {
	// Use streaming validator - zero allocations, validates in single pass
	return StreamingArrayValidator(data, elementType, schemaLength, schemaRows, schemaCols, prefer2D, enumValidator)
}
