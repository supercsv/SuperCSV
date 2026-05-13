// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/byteclass"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/validate/dfa"
)

// streamingListDecoder decodes list literals without building AST structures.
// Returns decoded values as []any.
// Mirrors streamingListValidator but returns decoded values instead of just validating.
// HOT_PATH -> minimal allocations (only for result slice)
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func streamingListDecoder(data []byte, elementType headerdef.ScalarType, expectedLength int, enumValidator *dfa.EnumValidator) ([]any, error) {
	// Handle null literal '_' (must be checked before bracket parsing)
	if isNullElement(data) {
		return nil, nil
	}

	// Parse brackets - reuse validator helper
	content, err := parseBracketedContent(data)
	if err != nil {
		return nil, err
	}

	// Empty list
	if len(content) == 0 {
		if expectedLength > 0 {
			return nil, fmt.Errorf("expected %d elements, got 0", expectedLength)
		}
		return []any{}, nil
	}

	// Parse optional prefix - reuse validator helper
	prefixLen, content, err := parsePrefix1D(content, expectedLength, elementType)
	if err != nil {
		return nil, err
	}

	// Pre-allocate result slice if we know the size
	var result []any
	if prefixLen >= 0 {
		result = make([]any, 0, prefixLen)
	} else if expectedLength > 0 {
		result = make([]any, 0, expectedLength)
	} else {
		result = make([]any, 0, 8) // reasonable default capacity
	}

	// Parse and decode elements
	elementCount := 0
	pos := 0
	n := len(content)
	// allowNewline tracks whether a \n continuation is currently permitted.
	// True right after '[' (start of content after bracket stripping) and after each ','.
	allowNewline := true

	for pos < n {
		// Parse next element - reuse validator helper
		element, isQuoted, newPos, newAllowNewline, err := parseNextElement(content, pos, allowNewline)
		if err != nil {
			return nil, err
		}
		pos = newPos
		allowNewline = newAllowNewline

		// Decode element
		decodedVal, err := decodeListElement(element, isQuoted, elementType, elementCount, enumValidator)
		if err != nil {
			return nil, err
		}

		result = append(result, decodedVal)
		elementCount++

		if pos >= n {
			break
		}
	}

	// Final check for trailing comma
	trimmedEnd := len(content)
	for trimmedEnd > 0 && byteclass.IsWhitespace[content[trimmedEnd-1]] {
		trimmedEnd--
	}
	if trimmedEnd > 0 && content[trimmedEnd-1] == ',' {
		return nil, fmt.Errorf("trailing comma not allowed in container")
	}

	// Validate counts
	if prefixLen >= 0 && prefixLen != elementCount {
		return nil, fmt.Errorf("prefix length %d does not match actual length %d", prefixLen, elementCount)
	}

	if expectedLength > 0 && expectedLength != elementCount {
		return nil, &LengthMismatchError{Expected: expectedLength, Actual: elementCount}
	}

	return result, nil
}

// decodeListElement decodes a single list element (already extracted).
// Returns the decoded value or error.
// HOT_PATH -> minimal allocations
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func decodeListElement(element []byte, isQuoted bool, elementType headerdef.ScalarType, index int, enumValidator *dfa.EnumValidator) (any, error) {
	// Trim whitespace
	element = trimASCIISpace(element)

	// Check for null - return nil
	if isNullElement(element) {
		return nil, nil // Nulls allowed in containers
	}

	// Explicit nesting detection
	if !isQuoted && len(element) > 0 && element[0] == '[' {
		return nil, fmt.Errorf("lists cannot contain nested containers (element %d)", index)
	}

	// Reject quoted numeric values (but allow enum and string)
	if isQuoted && isNumericType(elementType.Kind) {
		return nil, &ContainerElementError{Index: index, Err: fmt.Errorf("numeric values must not be quoted")}
	}

	// For quoted elements (string or enum)
	if isQuoted {
		if elementType.Kind == headerdef.ScalarString {
			// Quoted string - validate structure then decode
			if err := validateQuotedStringStructure(element); err != nil {
				return nil, &ContainerElementError{Index: index, Err: err}
			}
			// Decode quoted string - remove outer quotes and unescape
			return decodeQuotedString(element), nil
		} else if elementType.Kind == headerdef.ScalarEnum {
			// Quoted enum - validate structure, remove quotes, then dispatch to enum decoder
			if err := validateQuotedStringStructure(element); err != nil {
				return nil, &ContainerElementError{Index: index, Err: err}
			}
			// Decode quoted string and pass to dispatcher
			unquoted := decodeQuotedString(element)
			val, err := decodeScalarValue([]byte(unquoted), elementType, enumValidator)
			if err != nil {
				return nil, &ContainerElementError{Index: index, Err: err}
			}
			return val, nil
		} else {
			// Quoted non-string/non-enum scalar
			return nil, &ContainerElementError{Index: index, Err: fmt.Errorf("non-string scalars must not be quoted")}
		}
	}

	// Unquoted element - handle strings specially, dispatch all others
	if elementType.Kind == headerdef.ScalarString {
		// Validate unquoted string structure
		if err := validateUnquotedStringStructure(element); err != nil {
			return nil, &ContainerElementError{Index: index, Err: err}
		}
		// Return unquoted string as-is
		return string(element), nil
	}

	// Decode scalar (including enum) using dispatcher
	val, err := decodeScalarValue(element, elementType, enumValidator)
	if err != nil {
		return nil, &ContainerElementError{Index: index, Err: err}
	}

	return val, nil
}

// decodeQuotedString removes outer quotes and unescapes doubled quotes.
// Input: "hello""world" -> Output: hello"world
// HOT_PATH -> single allocation for result string
func decodeQuotedString(quoted []byte) string {
	// Remove outer quotes
	if len(quoted) < 2 {
		return ""
	}
	content := quoted[1 : len(quoted)-1]

	// Check if any escaped quotes exist
	hasEscape := false
	for i := 0; i < len(content)-1; i++ {
		if content[i] == '"' && content[i+1] == '"' {
			hasEscape = true
			break
		}
	}

	// Fast path: no escapes
	if !hasEscape {
		return string(content)
	}

	// Slow path: unescape doubled quotes
	result := make([]byte, 0, len(content))
	i := 0
	for i < len(content) {
		if i < len(content)-1 && content[i] == '"' && content[i+1] == '"' {
			result = append(result, '"')
			i += 2 // Skip both quotes
		} else {
			result = append(result, content[i])
			i++
		}
	}

	return string(result)
}

// streamingArrayDecoder decodes array literals without building AST structures.
// Returns decoded values as []any (1D) or [][]any (2D).
// Mirrors streamingArrayValidator but returns decoded values instead of just validating.
// HOT_PATH -> minimal allocations (only for result slices)
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func streamingArrayDecoder(data []byte, elementType headerdef.ScalarType, schemaLength, schemaRows, schemaCols int, prefer2D bool, enumValidator *dfa.EnumValidator) (any, error) {
	// Strip whitespace
	data = trimASCIISpace(data)

	// Handle null literal '_' (must be checked before bracket parsing)
	if isNullElement(data) {
		return nil, nil
	}

	// Check brackets
	if len(data) < 2 || data[0] != '[' || data[len(data)-1] != ']' {
		return nil, fmt.Errorf("invalid structure: missing brackets")
	}

	// Check if it's 2D (contains inner brackets or has prefix)
	content := data[1 : len(data)-1]
	content = trimASCIISpace(content)

	// Check for 2D array:
	// - Starts with '[' (normal format: [[1,2],[3,4]])
	// - Has 2D prefix pattern (prefix format: 2,3][[1,2,3],[4,5,6]])
	is2D := len(content) > 0 && content[0] == '['
	if !is2D && len(content) > 0 {
		// Check for prefix pattern: digit(s),digit(s)][
		pos := 0
		// Skip first number
		for pos < len(content) && content[pos] >= '0' && content[pos] <= '9' {
			pos++
		}
		// Check for comma
		if pos > 0 && pos < len(content) && content[pos] == ',' {
			pos++
			// Skip second number
			start := pos
			for pos < len(content) && content[pos] >= '0' && content[pos] <= '9' {
				pos++
			}
			// Check for ][
			if pos > start && pos+1 < len(content) && content[pos] == ']' && content[pos+1] == '[' {
				is2D = true
			}
		}
	}

	if is2D {
		// 2D array - decode and return as [][]any
		return decode2DArray(content, elementType, schemaRows, schemaCols, enumValidator)
	}

	// 1D array - use list decoder and return as []any
	result, err := streamingListDecoder(data, elementType, schemaLength, enumValidator)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// decode2DArray decodes 2D array structure
// Returns [][]any
// HOT_PATH -> minimal allocations (only for result slices)
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func decode2DArray(content []byte, elementType headerdef.ScalarType, schemaRows, schemaCols int, enumValidator *dfa.EnumValidator) ([][]any, error) {
	// Parse 2D prefix
	prefixRows, prefixCols, content, err := parsePrefix2D(content)
	if err != nil {
		return nil, err
	}

	// Validate prefix constraints
	if schemaRows > 0 && prefixRows >= 0 {
		return nil, fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
	}
	if schemaCols > 0 && prefixCols >= 0 {
		return nil, fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
	}

	// Determine expected column count
	expectedCols := -1 // variable-length by default
	if prefixCols >= 0 {
		expectedCols = prefixCols
	} else if schemaCols > 0 {
		expectedCols = schemaCols
	}

	// Pre-allocate result if we know the row count
	var result [][]any
	if prefixRows >= 0 {
		result = make([][]any, 0, prefixRows)
	} else if schemaRows > 0 {
		result = make([][]any, 0, schemaRows)
	} else {
		result = make([][]any, 0, 4) // reasonable default capacity
	}

	// Count rows and decode structure
	rowCount := 0
	colsPerRow := -1
	pos := 0
	n := len(content)

	for pos < n {
		// Skip whitespace
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			pos++
		}
		if pos >= n {
			break
		}

		// Expect '['
		if content[pos] != '[' {
			return nil, fmt.Errorf("expected '[' at start of row %d", rowCount)
		}

		// Find matching ']'
		rowStart := pos
		depth := 0
		for pos < n {
			if content[pos] == '[' {
				depth++
				// Explicit depth check: Arrays may be 1D or 2D only (max depth 2)
				if depth > 2 {
					return nil, fmt.Errorf("arrays may be 1D or 2D only (found depth %d at row %d)", depth, rowCount)
				}
			} else if content[pos] == ']' {
				depth--
				if depth == 0 {
					break
				}
			}
			pos++
		}

		if depth != 0 {
			return nil, fmt.Errorf("unmatched brackets in row %d", rowCount)
		}

		rowData := content[rowStart : pos+1]

		// Decode row
		rowResult, err := streamingListDecoder(rowData, elementType, expectedCols, enumValidator)
		if err != nil {
			// If it's a ContainerElementError, wrap it as Array2DElementError
			if elemErr, ok := err.(*ContainerElementError); ok {
				return nil, &Array2DElementError{Row: rowCount, Col: elemErr.Index, Err: elemErr.Err}
			}
			// Check if it's a length mismatch - reformat based on context
			if lenErr, ok := err.(*LengthMismatchError); ok {
				// For prefix 2D arrays, use prefix error format
				// For non-prefix 2D arrays (dynamic or fixed-schema), use row mismatch format
				if prefixCols >= 0 {
					// Prefix 2D: format like 1D prefix mismatch
					return nil, fmt.Errorf("row %d: prefix length %d does not match actual length %d", rowCount, lenErr.Expected, lenErr.Actual)
				}
				// Non-prefix 2D: use row length mismatch format
				return nil, fmt.Errorf("row %d has %d columns, expected %d", rowCount, lenErr.Actual, lenErr.Expected)
			}
			// Non-element errors (structure errors) keep row context
			return nil, fmt.Errorf("row %d: %w", rowCount, err)
		}

		// For first row of dynamic array: establish rectangular constraint
		if rowCount == 0 && expectedCols == -1 {
			colsPerRow = len(rowResult)
			expectedCols = colsPerRow // Use first row count for subsequent rows
		} else if rowCount == 0 {
			colsPerRow = expectedCols
		}

		result = append(result, rowResult)
		rowCount++
		pos++

		// Skip whitespace between ']' and ','.
		// Per spec: \n is not allowed before the comma (only after comma is continuation permitted).
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			if content[pos] == '\n' {
				return nil, fmt.Errorf("newline not allowed between row and comma (row %d)", rowCount-1)
			}
			pos++
		}
		if pos < n && content[pos] == ',' {
			pos++
			// Whitespace after the comma: one \n is permitted (continuation).
			// A second \n (blank line) is rejected.
			sawNewlineAfterComma := false
			for pos < n && byteclass.IsWhitespace[content[pos]] {
				if content[pos] == '\n' {
					if sawNewlineAfterComma {
						return nil, fmt.Errorf("blank line not allowed between rows (after row %d)", rowCount-1)
					}
					sawNewlineAfterComma = true
				}
				pos++
			}
			if pos >= n {
				return nil, fmt.Errorf("trailing comma not allowed in container")
			}
		}

		// If prefix specified, enforce exact row count
		if prefixRows >= 0 && rowCount >= prefixRows {
			break
		}
	}

	// If prefix specified, validate row count
	if prefixRows >= 0 && rowCount != prefixRows {
		return nil, fmt.Errorf("expected %d rows from prefix, got %d", prefixRows, rowCount)
	}

	// Ensure no trailing data after prefix rows
	if prefixRows >= 0 {
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			pos++
		}
		if pos < n {
			return nil, fmt.Errorf("unexpected data after prefix rows")
		}
	}

	// Validate dimensions
	if schemaRows > 0 && rowCount != schemaRows {
		return nil, fmt.Errorf("expected %d rows, got %d", schemaRows, rowCount)
	}

	if schemaCols > 0 && colsPerRow != schemaCols {
		return nil, fmt.Errorf("expected %d columns, got %d", schemaCols, colsPerRow)
	}

	return result, nil
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parsePrefix2D parses the 2D prefix notation: rows,cols][content
// Returns (rows, cols, remaining_content, error)
// If no valid prefix is found, returns (-1, -1, original_content, nil)
// Prefix format: [1-9][0-9]*,[1-9][0-9]*][
// Example: "2,3][[1,2,3],[4,5,6]" returns (2, 3, "[1,2,3],[4,5,6]", nil)
func parsePrefix2D(content []byte) (int, int, []byte, error) {
	n := len(content)
	pos := 0

	// Check for prefix: must start with [1-9]
	if n == 0 || content[0] < '1' || content[0] > '9' {
		return -1, -1, content, nil
	}

	// Parse first number (rows)
	rows := 0
	for pos < n && content[pos] >= '0' && content[pos] <= '9' {
		rows = rows*10 + int(content[pos]-'0')
		pos++
	}

	// Check for comma
	if pos >= n || content[pos] != ',' {
		return -1, -1, content, nil
	}
	pos++

	// Second number must also start with [1-9]
	if pos >= n || content[pos] < '1' || content[pos] > '9' {
		return -1, -1, content, nil
	}

	// Parse second number (cols)
	cols := 0
	for pos < n && content[pos] >= '0' && content[pos] <= '9' {
		cols = cols*10 + int(content[pos]-'0')
		pos++
	}

	// Check for ][ to confirm this is a prefix
	if pos+1 >= n || content[pos] != ']' || content[pos+1] != '[' {
		return -1, -1, content, nil
	}
	pos += 2

	// Valid 2D prefix found
	content = content[pos:]

	// Allow at most one \n after the prefix delimiter (line continuation).
	// A second \n would create a blank line, which is invalid.
	// trimASCIISpace treats \n as whitespace and would silently swallow it,
	// so we must check before calling trimASCIISpace.
	sawNewline := false
	for i := 0; i < len(content); i++ {
		if content[i] == ' ' || content[i] == '\t' || content[i] == '\r' {
			continue
		}
		if content[i] == '\n' {
			if sawNewline {
				return -1, -1, nil, fmt.Errorf("blank line not allowed after prefix delimiter")
			}
			sawNewline = true
			continue
		}
		break
	}

	return rows, cols, trimASCIISpace(content), nil
}
