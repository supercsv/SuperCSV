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

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateQuotedStringStructure validates quoted string structure without materialization.
// Zero allocations - validates syntax only, does not unescape.
func validateQuotedStringStructure(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("invalid structure: quoted string too short")
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid structure: missing quotes")
	}

	// Validate internal quote escaping without materializing
	content := data[1 : len(data)-1]
	n := len(content)
	for i := 0; i < n; i++ {
		if content[i] == '"' {
			// Must be part of a doubled quote pair
			if i+1 < n && content[i+1] == '"' {
				i++ // Skip the second quote
			} else {
				return fmt.Errorf("invalid structure: stray quote")
			}
		}
	}

	return nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateUnquotedStringStructure validates an already-trimmed unquoted string
// element without materialization. It matches scalar string semantics for
// forbidden characters and non-structural edge whitespace.
func validateUnquotedStringStructure(data []byte) error {
	// Trim already done by caller
	// Empty unquoted strings not allowed (must use null _ or quoted "")
	if len(data) == 0 {
		return fmt.Errorf("empty unquoted string (use _ or \"\")")
	}

	lastLead := 0
	for i := 0; i < len(data); i++ {
		c := byteclass.CharClass[data[i]]
		if c&charForbidden != 0 {
			return fmt.Errorf("unquoted string contains forbidden character: %q", data[i])
		}
		if c&charContinuation == 0 {
			lastLead = i
		}
	}

	if byteclass.CharClass[data[0]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(data) {
		return fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}
	if byteclass.CharClass[data[lastLead]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(data[lastLead:]) {
		return fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}

	return nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// StreamingListValidator validates list literals without building AST structures.
// Validates syntax, element types, and length constraints in a single pass.
// Zero allocations for valid lists (except error messages on failure).
// expectedLength: -1 for variable-length, >0 for fixed/known size
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func StreamingListValidator(data []byte, elementType headerdef.ScalarType, expectedLength int, enumValidator *dfa.EnumValidator) error {
	// Parse brackets
	content, err := parseBracketedContent(data)
	if err != nil {
		return err
	}

	// Empty list
	if len(content) == 0 {
		if expectedLength > 0 {
			return fmt.Errorf("expected %d elements, got 0", expectedLength)
		}
		return nil
	}

	// Parse optional prefix
	prefixLen, content, err := parsePrefix1D(content, expectedLength)
	if err != nil {
		return err
	}

	// Count and validate elements
	elementCount := 0
	pos := 0
	n := len(content)
	// allowNewline tracks whether a \n continuation is currently permitted.
	// True right after '[' (start of content after bracket stripping) and after each ','.
	allowNewline := true

	for pos < n {
		// Parse next element
		element, isQuoted, newPos, newAllowNewline, err := parseNextElement(content, pos, allowNewline)
		if err != nil {
			return err
		}
		pos = newPos
		allowNewline = newAllowNewline

		// Validate element
		if err := validateListElement(element, isQuoted, elementType, elementCount, enumValidator); err != nil {
			return err
		}

		elementCount++

		// Check if we're at the end but there was a trailing comma
		if pos >= n {
			// Check if content ended with comma (parseNextElement would have consumed it)
			// Look backwards from original position to detect trailing comma
			checkPos := newPos - 1
			for checkPos >= 0 && byteclass.IsWhitespace[content[checkPos]] {
				checkPos--
			}
			// parseNextElement skips comma, so if we're at end after comma, that's trailing
			break
		}
	}

	// Final check for trailing comma: if last non-whitespace char is comma
	trimmedEnd := len(content)
	for trimmedEnd > 0 && byteclass.IsWhitespace[content[trimmedEnd-1]] {
		trimmedEnd--
	}
	if trimmedEnd > 0 && content[trimmedEnd-1] == ',' {
		return fmt.Errorf("trailing comma not allowed in container")
	}

	// Validate counts
	if prefixLen >= 0 && prefixLen != elementCount {
		return fmt.Errorf("prefix length %d does not match actual length %d", prefixLen, elementCount)
	}

	if expectedLength > 0 && expectedLength != elementCount {
		return &LengthMismatchError{Expected: expectedLength, Actual: elementCount}
	}

	return nil
}

// streamingListValidatorWithCount is like streamingListValidator but returns element count.
// Used ONLY for first row of dynamic 2D arrays to establish column count.
// expectedLength MUST be -1 (variable-length discovery mode).
// HOT_PATH -> zero allocations
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func streamingListValidatorWithCount(data []byte, elementType headerdef.ScalarType, enumValidator *dfa.EnumValidator) (int, error) {
	// Parse brackets
	content, err := parseBracketedContent(data)
	if err != nil {
		return 0, err
	}

	// Empty list
	if len(content) == 0 {
		return 0, nil
	}

	// Parse optional prefix (should not exist for dynamic arrays, but handle it)
	prefixLen, content, err := parsePrefix1D(content, -1)
	if err != nil {
		return 0, err
	}

	// Count and validate elements
	elementCount := 0
	pos := 0
	n := len(content)
	// allowNewline tracks whether a \n continuation is currently permitted.
	// True right after '[' (start of content after bracket stripping) and after each ','.
	allowNewline := true

	for pos < n {
		// Parse next element
		element, isQuoted, newPos, newAllowNewline, err := parseNextElement(content, pos, allowNewline)
		if err != nil {
			return 0, err
		}
		pos = newPos
		allowNewline = newAllowNewline

		// Validate element
		if err := validateListElement(element, isQuoted, elementType, elementCount, enumValidator); err != nil {
			return 0, err
		}

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
		return 0, fmt.Errorf("trailing comma not allowed in container")
	}

	// Validate prefix if present
	if prefixLen >= 0 && prefixLen != elementCount {
		return 0, fmt.Errorf("prefix length %d does not match actual length %d", prefixLen, elementCount)
	}

	return elementCount, nil
}

// validateListElement validates a single list element (already extracted).
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func validateListElement(element []byte, isQuoted bool, elementType headerdef.ScalarType, index int, enumValidator *dfa.EnumValidator) error {
	// Trim whitespace
	element = trimASCIISpace(element)

	// Check for null
	if isNullElement(element) {
		return nil // Nulls allowed in containers
	}

	// Explicit nesting detection: Check if unquoted element starts with '['
	// This provides a clear semantic error instead of generic "forbidden character"
	if !isQuoted && len(element) > 0 && element[0] == '[' {
		return fmt.Errorf("lists cannot contain nested containers (element %d)", index)
	}

	// Reject quoted numeric values
	if isQuoted && isNumericType(elementType.Kind) {
		return &ContainerElementError{Index: index, Err: fmt.Errorf("numeric values must not be quoted")}
	}

	// For quoted elements, validate as quoted string/scalar
	if isQuoted {
		if elementType.Kind == headerdef.ScalarString {
			// Quoted string - validate structure (zero-allocation)
			if err := validateQuotedStringStructure(element); err != nil {
				return &ContainerElementError{Index: index, Err: err}
			}
			return nil
		} else {
			// Quoted non-string scalar - should have been rejected above
			return &ContainerElementError{Index: index, Err: fmt.Errorf("non-string scalars must not be quoted")}
		}
	}

	// Unquoted element
	if elementType.Kind == headerdef.ScalarString {
		// Validate unquoted string (zero-allocation)
		if err := validateUnquotedStringStructure(element); err != nil {
			return &ContainerElementError{Index: index, Err: err}
		}
		return nil
	}

	// Validate scalar with DFA (element already trimmed above)
	if err := validateScalarWithDFA(element, elementType, enumValidator); err != nil {
		return &ContainerElementError{Index: index, Err: err}
	}

	return nil
}

// StreamingArrayValidator validates array literals without building AST structures.
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func StreamingArrayValidator(data []byte, elementType headerdef.ScalarType, schemaLength, schemaRows, schemaCols int, prefer2D bool, enumValidator *dfa.EnumValidator) error {
	// Strip whitespace
	data = trimASCIISpace(data)

	// Check brackets
	if len(data) < 2 || data[0] != '[' || data[len(data)-1] != ']' {
		return fmt.Errorf("invalid structure: missing brackets")
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
		// 2D array - validate structure
		return validate2DArray(content, elementType, schemaRows, schemaCols, enumValidator)
	}

	// 1D array - use list validator
	if err := StreamingListValidator(data, elementType, schemaLength, enumValidator); err != nil {
		return err
	}

	return nil
}

// validate2DArray validates 2D array structure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// For enum elements, pass the pre-built EnumValidator; for stateless scalars, pass nil.
func validate2DArray(content []byte, elementType headerdef.ScalarType, schemaRows, schemaCols int, enumValidator *dfa.EnumValidator) error {
	// Parse 2D prefix
	prefixRows, prefixCols, content, err := parsePrefix2D(content)
	if err != nil {
		return err
	}

	// Validate prefix constraints
	if schemaRows > 0 && prefixRows >= 0 {
		return fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
	}
	if schemaCols > 0 && prefixCols >= 0 {
		return fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
	}

	// Determine expected column count
	expectedCols := -1 // variable-length by default
	if prefixCols >= 0 {
		expectedCols = prefixCols
	} else if schemaCols > 0 {
		expectedCols = schemaCols
	}

	// Count rows and validate structure
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
			return fmt.Errorf("expected '[' at start of row %d", rowCount)
		}

		// Find matching ']'
		rowStart := pos
		depth := 0
		for pos < n {
			if content[pos] == '[' {
				depth++
				// Explicit depth check: Arrays may be 1D or 2D only (max depth 2)
				if depth > 2 {
					return fmt.Errorf("arrays may be 1D or 2D only (found depth %d at row %d)", depth, rowCount)
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
			return fmt.Errorf("unmatched brackets in row %d", rowCount)
		}

		rowData := content[rowStart : pos+1]

		// For first row of dynamic array: validate AND count to establish rectangular constraint
		if rowCount == 0 && expectedCols == -1 {
			// Dynamic 2D array: use variant that returns count to establish rectangular constraint
			var err error
			colsPerRow, err = streamingListValidatorWithCount(rowData, elementType, enumValidator)
			if err != nil {
				// If it's a ContainerElementError, wrap it as Array2DElementError
				if elemErr, ok := err.(*ContainerElementError); ok {
					return &Array2DElementError{Row: 0, Col: elemErr.Index, Err: elemErr.Err}
				}
				return fmt.Errorf("row 0: %w", err)
			}
			expectedCols = colsPerRow // Use first row count for subsequent rows
		} else {
			// Validate row with expected column count (from schema, prefix, or first row).
			// Special case: expectedCols == 0 means every row must be empty [].
			// StreamingListValidator treats 0 as "skip check" (headerdef convention: 0 = dynamic),
			// so we must handle the zero-column case explicitly.
			if expectedCols == 0 {
				count, err := streamingListValidatorWithCount(rowData, elementType, enumValidator)
				if err != nil {
					if elemErr, ok := err.(*ContainerElementError); ok {
						return &Array2DElementError{Row: rowCount, Col: elemErr.Index, Err: elemErr.Err}
					}
					return fmt.Errorf("row %d: %w", rowCount, err)
				}
				if count != 0 {
					return fmt.Errorf("row %d has %d columns, expected %d", rowCount, count, 0)
				}
			} else if err := StreamingListValidator(rowData, elementType, expectedCols, enumValidator); err != nil {
				// If it's a ContainerElementError, wrap it as Array2DElementError with row/col position
				if elemErr, ok := err.(*ContainerElementError); ok {
					return &Array2DElementError{Row: rowCount, Col: elemErr.Index, Err: elemErr.Err}
				}
				// Check if it's a length mismatch - reformat based on context (no string parsing!)
				if lenErr, ok := err.(*LengthMismatchError); ok {
					// For prefix 2D arrays, use prefix error format
					// For non-prefix 2D arrays (dynamic or fixed-schema), use row mismatch format
					if prefixCols >= 0 {
						// Prefix 2D: format like 1D prefix mismatch
						return fmt.Errorf("row %d: prefix length %d does not match actual length %d", rowCount, lenErr.Expected, lenErr.Actual)
					}
					// Non-prefix 2D: use row length mismatch format
					return fmt.Errorf("row %d has %d columns, expected %d", rowCount, lenErr.Actual, lenErr.Expected)
				}
				// Non-element errors (structure errors) keep row context
				return fmt.Errorf("row %d: %w", rowCount, err)
			}

			// Set colsPerRow for dimension tracking (first row only)
			if rowCount == 0 {
				colsPerRow = expectedCols
			}
		}

		rowCount++
		pos++

		// Skip whitespace between ']' and ','.
		// Per spec: \n is not allowed before the comma (only after comma is continuation permitted).
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			if content[pos] == '\n' {
				return fmt.Errorf("newline not allowed between row and comma (row %d)", rowCount-1)
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
						return fmt.Errorf("blank line not allowed between rows (after row %d)", rowCount-1)
					}
					sawNewlineAfterComma = true
				}
				pos++
			}
			if pos >= n {
				return fmt.Errorf("trailing comma not allowed in container")
			}
		}

		// If prefix specified, enforce exact row count
		if prefixRows >= 0 && rowCount >= prefixRows {
			break
		}
	}

	// If prefix specified, validate row count
	if prefixRows >= 0 && rowCount != prefixRows {
		return fmt.Errorf("expected %d rows from prefix, got %d", prefixRows, rowCount)
	}

	// Ensure no trailing data after prefix rows
	if prefixRows >= 0 {
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			pos++
		}
		if pos < n {
			return fmt.Errorf("unexpected data after prefix rows")
		}
	}

	// Validate dimensions
	if schemaRows > 0 && rowCount != schemaRows {
		return fmt.Errorf("expected %d rows, got %d", schemaRows, rowCount)
	}

	if schemaCols > 0 && colsPerRow != schemaCols {
		return fmt.Errorf("expected %d columns, got %d", schemaCols, colsPerRow)
	}

	return nil
}

// isNullElement checks if a container element is the null literal "_".
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func isNullElement(data []byte) bool {
	// Only "_" is null
	return len(data) == 1 && data[0] == '_'
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parseBracketedContent strips outer brackets from container value and returns inner content.
// Zero allocations.
func parseBracketedContent(value []byte) ([]byte, error) {
	value = trimASCIISpace(value)

	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("invalid structure: missing brackets")
	}

	content := value[1 : len(value)-1]
	return trimASCIISpace(content), nil
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parsePrefix1D parses optional "N][" prefix for 1D containers.
// Input is bracket-stripped content, e.g. "3][1,2,3" from original "[3][1,2,3]".
// Returns (prefixLength, remainingContent, error).
// prefixLength = -1 if no prefix found.
// Enforces spec rule: prefix forbidden on fixed-size containers.
// Zero allocations.
func parsePrefix1D(content []byte, fixedLength int) (int, []byte, error) {
	n := len(content)
	pos := 0

	// Skip leading whitespace
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		pos++
	}

	// Check for strict prefix grammar: [1-9][0-9]*][
	if pos < n && content[pos] >= '1' && content[pos] <= '9' {
		prefixNum := 0
		for pos < n && content[pos] >= '0' && content[pos] <= '9' {
			prefixNum = prefixNum*10 + int(content[pos]-'0')
			pos++
		}

		// Check for ][ to confirm this is a prefix
		if pos+1 < n && content[pos] == ']' && content[pos+1] == '[' {
			// Fixed-size containers cannot use prefix notation
			if fixedLength > 0 {
				return -1, nil, fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
			}

			content = content[pos+2:]

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
						return -1, nil, fmt.Errorf("blank line not allowed after prefix delimiter")
					}
					sawNewline = true
					continue
				}
				break
			}

			return prefixNum, trimASCIISpace(content), nil
		}
	}

	return -1, content, nil
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parsePrefix2D parses optional "R,C][" prefix for 2D arrays.
// Input is bracket-stripped content, e.g. "2,3][[1,2,3],[4,5,6]" from original "[2,3][[1,2,3],[4,5,6]]".
// Returns (rows, cols, remainingContent, error).
// rows = -1 if no prefix found.
// Zero allocations.
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

// parseNextElement parses the next element from content starting at pos.
// Returns (element, isQuoted, newPos, allowNewline, error).
// allowNewline tracks whether a \n continuation is permitted at the current position:
//   - true on first call (right after '[') and right after each ',' (comma enables continuation)
//   - false after any \n is consumed (blank-line prevention: second \n errors)
//   - false after element body starts (no \n mid-element or before comma)
//
// NOTE: content passed here has already had its outer brackets stripped by
// parseBracketedContent, so the depth visible to this function is 1 less than
// the raw field depth.
//
// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// Handles quoted strings with doubled quote escapes and unquoted elements.
// Zero allocations.
func parseNextElement(content []byte, pos int, allowNewline bool) ([]byte, bool, int, bool, error) {
	n := len(content)

	// Loop 1: Skip leading whitespace.
	// A single \n is allowed when allowNewline is true (continuation after '[' or ',').
	// A second \n with allowNewline already false is a blank line -> error.
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		if content[pos] == '\n' {
			if !allowNewline {
				return nil, false, pos, false, fmt.Errorf("blank line not allowed in container")
			}
			allowNewline = false
		}
		pos++
	}

	if pos >= n {
		return nil, false, pos, allowNewline, fmt.Errorf("unexpected end of content")
	}

	elemStart := pos
	isQuoted := false

	if content[pos] == '"' {
		// Quoted element
		isQuoted = true
		pos++

		for pos < n {
			if content[pos] == '"' {
				if pos+1 < n && content[pos+1] == '"' {
					// Doubled quote escape
					pos += 2
				} else {
					// End quote
					pos++
					break
				}
			} else {
				pos++
			}
		}
	} else {
		// Loop 2: Unquoted element - read until comma or end.
		// \n inside an unquoted element body is never valid.
		for pos < n && content[pos] != ',' {
			if content[pos] == '\n' {
				return nil, false, pos, false, fmt.Errorf("newline not allowed inside unquoted element")
			}
			pos++
		}
	}

	element := content[elemStart:pos]

	// Loop 3: Skip trailing whitespace after element (before comma or end).
	// allowNewline is false here (element body was just read); any \n is invalid.
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		if content[pos] == '\n' {
			return nil, false, pos, false, fmt.Errorf("newline not allowed between element and comma")
		}
		pos++
	}

	// Handle comma if present
	if pos < n && content[pos] == ',' {
		pos++ // Skip comma

		// Comma resets continuation: the next \n is permitted.
		allowNewline = true

		// Loop 4: Skip whitespace after comma.
		// A single \n is allowed (continuation); a second \n is a blank line -> error.
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			if content[pos] == '\n' {
				if !allowNewline {
					return nil, false, pos, false, fmt.Errorf("blank line not allowed in container")
				}
				allowNewline = false
			}
			pos++
		}

		// Check for double comma (consecutive commas)
		if pos < n && content[pos] == ',' {
			return nil, false, pos, false, fmt.Errorf("consecutive commas not allowed (missing element)")
		}
	}

	return element, isQuoted, pos, allowNewline, nil
}
