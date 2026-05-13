// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package containers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	verr "github.com/supercsv/supercsv/internal/v1_0/validate/errors"
)

// Element represents a scalar literal inside a container.
type Element struct {
	Value  string
	Quoted bool
}

// ListLiteral contains the parsed form of a list<T> value.
type ListLiteral struct {
	Elements  []Element
	PrefixLen *int
}

// ArrayLiteral contains the parsed form of an arr<T> value.
type ArrayLiteral struct {
	Dim        int // 1 or 2
	Elements   []Element
	Rows       [][]Element
	PrefixLen  *int
	PrefixRows *int
	PrefixCols *int
}

const (
	dim1D = 1
	dim2D = 2
)

var (
	errNestedContainer = errors.New("nested container not allowed")
)

// ParseListLiteral parses a list<T> literal and returns its elements along with
// any optional length prefix. Errors map directly to ValidationError codes.
func ParseListLiteral(row int, column, raw string) (*ListLiteral, *verr.ValidationError) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return nil, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "lists must be enclosed in []")
	}

	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	prefixText, body, hasPrefix, err := splitPrefix(inner)
	if err != nil {
		return nil, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "%s", err.Error())
	}

	var prefixLen *int
	if hasPrefix {
		if strings.TrimSpace(body) == "" {
			return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "prefix form not allowed for empty lists")
		}
		length, perr := parsePositiveInt(prefixText)
		if perr != nil {
			return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "invalid list prefix %q", prefixText)
		}
		prefixLen = &length
	}

	elements, perr := parseElementSequence(row, column, body)
	if perr != nil {
		return nil, perr
	}

	return &ListLiteral{Elements: elements, PrefixLen: prefixLen}, nil
}

// ParseArrayLiteral parses an arr<T> literal. prefer2D forces empty literals to
// be interpreted as 2D (used for schemas that declare a fixed 2D shape).
func ParseArrayLiteral(row int, column, raw string, prefer2D bool) (*ArrayLiteral, *verr.ValidationError) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return nil, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "arrays must be enclosed in []")
	}

	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])

	// Parser-level: enforce maximum bracket depth = 2
	// This catches [[[...]]] before semantic validation
	maxDepth := 0
	currentDepth := 0
	inQuotes := false
	for i := 0; i < len(inner); i++ {
		if inner[i] == '"' {
			if inQuotes {
				if i+1 < len(inner) && inner[i+1] == '"' {
					i++ // Skip escaped quote
				} else {
					inQuotes = false
				}
			} else {
				inQuotes = true
			}
			continue
		}
		if !inQuotes {
			if inner[i] == '[' {
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			} else if inner[i] == ']' {
				currentDepth--
			}
		}
	}
	if maxDepth > 2 {
		return nil, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "arrays cannot be nested more than 2 levels deep")
	}

	prefixText, body, hasPrefix, err := splitPrefix(inner)
	if err != nil {
		return nil, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "%s", err.Error())
	}

	var (
		prefixLen  *int
		prefixRows *int
		prefixCols *int
		prefix2D   bool
	)

	if hasPrefix {
		if strings.TrimSpace(body) == "" {
			return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "prefix form not allowed for empty arrays")
		}
		if strings.Contains(prefixText, ",") {
			parts := strings.SplitN(prefixText, ",", 2)
			if len(parts) != 2 {
				return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "invalid array prefix %q", prefixText)
			}
			rows, rerr := parsePositiveInt(strings.TrimSpace(parts[0]))
			if rerr != nil {
				return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "invalid row prefix %q", prefixText)
			}
			cols, cerr := parsePositiveInt(strings.TrimSpace(parts[1]))
			if cerr != nil {
				return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "invalid column prefix %q", prefixText)
			}
			prefixRows = &rows
			prefixCols = &cols
			prefix2D = true
		} else {
			length, perr := parsePositiveInt(prefixText)
			if perr != nil {
				return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "invalid array prefix %q", prefixText)
			}
			prefixLen = &length
		}
	}

	trimmedBody := strings.TrimSpace(body)
	dim := dim1D
	if prefix2D {
		dim = dim2D
	} else if prefer2D {
		dim = dim2D
	} else if strings.HasPrefix(trimmedBody, "[") {
		// Heuristic: nested rows start with '['
		dim = dim2D
	}

	if dim == dim1D {
		elements, perr := parseElementSequence(row, column, body)
		if perr != nil {
			return nil, perr
		}
		if prefixLen != nil && len(elements) == 0 {
			return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "prefix form not allowed for empty arrays")
		}
		if prefix2D {
			return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "2D prefix cannot be applied to 1D data")
		}
		return &ArrayLiteral{Dim: dim1D, Elements: elements, PrefixLen: prefixLen}, nil
	}

	// Parse 2D array rows
	rows, cols, perr := parseRowMatrix(row, column, body)
	if perr != nil {
		return nil, perr
	}
	if prefixRows != nil && len(rows) == 0 {
		return nil, verr.New(row, column, verr.CodeInvalidPrefix.Code, "prefix form not allowed for empty arrays")
	}

	// Validate prefix dimensions match actual data
	if prefixRows != nil {
		if len(rows) != *prefixRows {
			return nil, verr.New(row, column, verr.CodeWrongShape.Code, "expected %d rows from prefix, got %d", *prefixRows, len(rows))
		}
	}
	if prefixCols != nil {
		if cols != *prefixCols {
			return nil, verr.New(row, column, verr.CodeWrongShape.Code, "expected %d columns from prefix, got %d", *prefixCols, cols)
		}
	}

	return &ArrayLiteral{Dim: dim2D, Rows: rows, PrefixRows: prefixRows, PrefixCols: prefixCols}, nil
}

func splitPrefix(inner string) (string, string, bool, error) {
	depth := 0
	inQuotes := false
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if ch == '"' {
			if inQuotes {
				if i+1 < len(inner) && inner[i+1] == '"' {
					i++
				} else {
					inQuotes = false
				}
			} else {
				inQuotes = true
			}
			continue
		}
		if inQuotes {
			continue
		}
		switch ch {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return "", "", false, fmt.Errorf("invalid structure: unexpected ']' in literal")
			}
			depth--
		case ':':
			if depth == 0 {
				prefix := strings.TrimSpace(inner[:i])
				body := strings.TrimSpace(inner[i+1:])
				return prefix, body, true, nil
			}
		}
	}
	if inQuotes || depth < 0 {
		return "", "", false, fmt.Errorf("invalid structure: unterminated literal")
	}
	return "", strings.TrimSpace(inner), false, nil
}

func parseElementSequence(row int, column, body string) ([]Element, *verr.ValidationError) {
	if strings.TrimSpace(body) == "" {
		return []Element{}, nil
	}
	tokens, err := splitScalarTokens(body)
	if err != nil {
		code := verr.CodeInvalidContainerSyntax.Code
		if errors.Is(err, errNestedContainer) {
			code = verr.CodeNestedContainerNotAllowed.Code
		}
		return nil, verr.New(row, column, code, "%s", err.Error())
	}
	elements := make([]Element, 0, len(tokens))
	for _, tok := range tokens {
		// NOTE: Do NOT trim tokens here! The container grammar enforces
		// its own whitespace rules via validateString(). Trimming here
		// would hide invalid leading/trailing whitespace in unquoted strings.
		elem, perr := parseElementToken(tok)
		if perr != nil {
			code := verr.CodeInvalidContainerSyntax.Code
			if errors.Is(perr, errNestedContainer) {
				code = verr.CodeNestedContainerNotAllowed.Code
			}
			return nil, verr.New(row, column, code, "%s", perr.Error())
		}
		elements = append(elements, elem)
	}
	return elements, nil
}

func splitScalarTokens(body string) ([]string, error) {
	var tokens []string
	start := 0
	inQuotes := false
	depth := 0
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch == '"' {
			if inQuotes {
				if i+1 < len(body) && body[i+1] == '"' {
					i++
				} else {
					inQuotes = false
				}
			} else {
				inQuotes = true
			}
			continue
		}
		if inQuotes {
			continue
		}
		switch ch {
		case '[':
			depth++
			return nil, errNestedContainer
		case ']':
			if depth == 0 {
				return nil, errNestedContainer
			}
			depth--
		case ',':
			if depth == 0 {
				tokens = append(tokens, body[start:i])
				start = i + 1
			}
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("invalid structure: unterminated quoted value")
	}
	tokens = append(tokens, body[start:])
	return tokens, nil
}

func parseElementToken(token string) (Element, error) {
	if token == "" {
		return Element{}, fmt.Errorf("empty value not allowed in container")
	}
	if token[0] == '"' {
		if len(token) < 2 || token[len(token)-1] != '"' {
			return Element{}, fmt.Errorf("invalid structure: unterminated quoted value")
		}
		val, err := unescapeQuoted(token[1 : len(token)-1])
		if err != nil {
			return Element{}, err
		}
		return Element{Value: val, Quoted: true}, nil
	}
	if strings.ContainsRune(token, '"') {
		return Element{}, fmt.Errorf("invalid structure: quote in unquoted value")
	}
	return Element{Value: token, Quoted: false}, nil
}

func parseRowMatrix(row int, column, body string) ([][]Element, int, *verr.ValidationError) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return [][]Element{}, 0, nil
	}
	rowsRaw, err := splitRows(trimmed)
	if err != nil {
		code := verr.CodeInvalidContainerSyntax.Code
		if errors.Is(err, errNestedContainer) {
			code = verr.CodeNestedContainerNotAllowed.Code
		}
		return nil, 0, verr.New(row, column, code, "%s", err.Error())
	}

	rows := make([][]Element, 0, len(rowsRaw))
	cols := 0
	for idx, rawRow := range rowsRaw {
		literal := strings.TrimSpace(rawRow)
		if literal == "" {
			return nil, 0, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "empty row literal")
		}
		if len(literal) < 2 || literal[0] != '[' || literal[len(literal)-1] != ']' {
			return nil, 0, verr.New(row, column, verr.CodeInvalidContainerSyntax.Code, "rows must be enclosed in []")
		}
		elems, perr := parseElementSequence(row, column, literal[1:len(literal)-1])
		if perr != nil {
			return nil, 0, perr
		}
		if idx == 0 {
			cols = len(elems)
		} else if len(elems) != cols {
			return nil, 0, verr.New(row, column, verr.CodeWrongShape.Code, "rows must all contain %d elements, row %d has %d", cols, idx+1, len(elems))
		}
		rows = append(rows, elems)
	}
	return rows, cols, nil
}

func splitRows(body string) ([]string, error) {
	var rows []string
	start := 0
	inQuotes := false
	depth := 0
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch == '"' {
			if inQuotes {
				if i+1 < len(body) && body[i+1] == '"' {
					i++
				} else {
					inQuotes = false
				}
			} else {
				inQuotes = true
			}
			continue
		}
		if inQuotes {
			continue
		}
		switch ch {
		case '[':
			depth++
			if depth > 1 {
				return nil, errNestedContainer
			}
		case ']':
			if depth == 0 {
				return nil, fmt.Errorf("invalid structure: unexpected ']' in array")
			}
			depth--
		case ',':
			if depth == 0 {
				rows = append(rows, body[start:i])
				start = i + 1
			}
		}
	}
	if inQuotes || depth != 0 {
		return nil, fmt.Errorf("invalid structure: unterminated array")
	}
	rows = append(rows, body[start:])
	return rows, nil
}

func unescapeQuoted(body string) (string, error) {
	if body == "" {
		return "", nil
	}
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch == '"' {
			if i+1 >= len(body) || body[i+1] != '"' {
				return "", fmt.Errorf("invalid structure: unterminated escape sequence")
			}
			b.WriteByte('"')
			i++
			continue
		}
		b.WriteByte(ch)
	}
	return b.String(), nil
}

func parsePositiveInt(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("missing integer")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", value)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid value: expected > 0")
	}
	return n, nil
}
