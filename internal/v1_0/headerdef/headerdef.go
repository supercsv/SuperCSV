// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package headerdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Row struct {
	Fields                       []string
	CommentCounts                []int8
	MetaCounts                   []int8
	AnnotationAfterTrailingComma bool
	Line                         int
	EndLine                      int
}

type HeaderDef struct {
	Columns []Column
	Line    int
}

type Column struct {
	Index int
	Name  string
	Type  ColumnType
}

type ColumnType struct {
	Kind   TypeKind
	Scalar ScalarType
	List   ListType
	Array  ArrayType
}

type TypeKind int

const (
	TypeKindScalar TypeKind = iota
	TypeKindList
	TypeKindArray
)

type ScalarKind int

const (
	ScalarString ScalarKind = iota
	ScalarInt
	ScalarFloat
	ScalarDecimal
	ScalarBool
	ScalarBytesHex
	ScalarBytesB64
	ScalarDate
	ScalarTime
	ScalarTimestamp
	ScalarDatetime
	ScalarDatetimeTZ
	ScalarDuration
	ScalarTimezone
	ScalarUUID
	ScalarEnum
)

type ScalarType struct {
	Kind ScalarKind
	Enum *EnumSpec
}

type EnumSpec struct {
	Values []EnumValue
	names  map[string]struct{}
	values map[string]struct{}
}

type EnumValue struct {
	Index int32 // 0-based declaration position; set once at parse time
	Name  string
	Value string
}

type ListType struct {
	Element     ScalarType
	FixedLength int
}

type ArrayType struct {
	Element ScalarType
	Rank    ArrayRank
	Length  int
	Rows    int
	Cols    int
}

type ArrayRank int

const (
	ArrayRankFlexible ArrayRank = iota
	ArrayRank1DFixed
	ArrayRank2DFixed
)

// Identifier validation lookup tables for zero-alloc validation.
// Used for column names, enum labels, and type parameters.
const maxIdentifierLength = 255

var isIdentifierStart [256]bool
var isIdentifierRest [256]bool

func init() {
	for c := byte('A'); c <= byte('Z'); c++ {
		isIdentifierStart[c] = true
		isIdentifierStart[c|0x20] = true
	}
	for c := byte('0'); c <= byte('9'); c++ {
		isIdentifierStart[c] = true
	}

	for c := byte('A'); c <= byte('Z'); c++ {
		isIdentifierRest[c] = true
		isIdentifierRest[c|0x20] = true
	}
	for c := byte('0'); c <= byte('9'); c++ {
		isIdentifierRest[c] = true
	}
	isIdentifierRest['_'] = true
	isIdentifierRest['-'] = true
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	if len(s) > maxIdentifierLength {
		return false
	}
	if !isIdentifierStart[s[0]] {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentifierRest[s[i]] {
			return false
		}
	}
	return true
}

type ErrorCode string

const (
	CodeInvalidColumnName         ErrorCode = "invalid_column_name"
	CodeMissingType               ErrorCode = "missing_type"
	CodeUnknownType               ErrorCode = "unknown_type"
	CodeInvalidContainerSyntax    ErrorCode = "invalid_container_syntax"
	CodeInvalidPrefix             ErrorCode = "invalid_prefix"
	CodeNestedContainerNotAllowed ErrorCode = "nested_container_not_allowed"
	CodeInvalidEnum               ErrorCode = "invalid_enum"
	CodeDuplicateAnnotation       ErrorCode = "duplicate_annotation"
	CodeAnnotationAfterTrailing   ErrorCode = "annotation_after_trailing_comma"
)

type Error struct {
	Line    int
	EndLine int
	Column  string
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	lineInfo := fmt.Sprintf("%d", e.Line)
	if e.EndLine > 0 && e.EndLine != e.Line {
		lineInfo = fmt.Sprintf("%d-%d", e.Line, e.EndLine)
	}
	if e.Column != "" {
		return fmt.Sprintf("line %s column %s: %s", lineInfo, e.Column, e.Message)
	}
	return fmt.Sprintf("line %s: %s", lineInfo, e.Message)
}

func newError(line int, column string, code ErrorCode, msg string, args ...interface{}) *Error {
	return &Error{
		Line:    line,
		Column:  column,
		Code:    code,
		Message: fmt.Sprintf(msg, args...),
	}
}

func newErrorRow(row *Row, column string, code ErrorCode, msg string, args ...interface{}) *Error {
	err := &Error{
		Line:    row.Line,
		Column:  column,
		Code:    code,
		Message: fmt.Sprintf(msg, args...),
	}
	if row.EndLine > 0 && row.EndLine != row.Line {
		err.EndLine = row.EndLine
	}
	return err
}

var ErrMissingHeader = errors.New("no header row found")

func ParseHeader(row *Row) (*HeaderDef, error) {
	if row == nil {
		return nil, newError(0, "", CodeInvalidColumnName, "nil header row")
	}
	if len(row.Fields) == 0 {
		return nil, newErrorRow(row, "", CodeMissingType, "header row contains no columns")
	}
	if row.AnnotationAfterTrailingComma {
		return nil, newErrorRow(row, "", CodeAnnotationAfterTrailing, "annotation after trailing comma: annotation must appear on the continuation line, not before it")
	}
	cols := make([]Column, 0, len(row.Fields))
	// Case-insensitive uniqueness check, but preserve original casing
	seen := make(map[string]struct{})
	for idx, raw := range row.Fields {
		cell := strings.TrimSpace(raw)
		if cell == "" {
			return nil, newErrorRow(row, fmt.Sprintf("col%d", idx), CodeMissingType, "empty header cell")
		}
		name, typeExpr, err := splitCell(cell, row)
		if err != nil {
			return nil, err
		}
		if idx < len(row.CommentCounts) && row.CommentCounts[idx] > 1 {
			return nil, newErrorRow(row, name, CodeDuplicateAnnotation, "multiple comment blocks on field")
		}
		if idx < len(row.MetaCounts) && row.MetaCounts[idx] > 1 {
			return nil, newErrorRow(row, name, CodeDuplicateAnnotation, "multiple metadata blocks on field")
		}
		// Check for duplicates using lowercase, but preserve original casing
		lowerName := strings.ToLower(name)
		if _, ok := seen[lowerName]; ok {
			return nil, newErrorRow(row, name, CodeInvalidColumnName, "duplicate column name (case-insensitive)")
		}
		seen[lowerName] = struct{}{}
		colType, err := parseColumnType(typeExpr, row.Line, name)
		if err != nil {
			// Update error with EndLine if it's a headerdef.Error
			if schemaErr, ok := err.(*Error); ok && row.EndLine > 0 && row.EndLine != row.Line {
				schemaErr.EndLine = row.EndLine
			}
			return nil, err
		}
		cols = append(cols, Column{Index: idx, Name: name, Type: colType})
	}
	return &HeaderDef{Columns: cols, Line: row.Line}, nil
}

func splitCell(cell string, row *Row) (string, string, error) {
	parts := strings.SplitN(cell, ":", 2)
	if len(parts) != 2 {
		return "", "", newErrorRow(row, "", CodeMissingType, "header cell missing type declaration")
	}
	name := strings.TrimSpace(parts[0])
	typeExpr := strings.TrimSpace(parts[1])
	if name == "" {
		return "", "", newErrorRow(row, "", CodeInvalidColumnName, "missing column name")
	}
	if typeExpr == "" {
		return "", "", newErrorRow(row, name, CodeMissingType, "missing column type")
	}
	if !isValidIdentifier(name) {
		return "", "", newErrorRow(row, name, CodeInvalidColumnName, "invalid column name")
	}
	return name, typeExpr, nil
}

func parseColumnType(expr string, line int, column string) (ColumnType, error) {
	expr = strings.TrimSpace(expr)

	// Extract potential container keyword (before <)
	keyword := extractContainerKeyword(expr)
	canonical, isContainer := resolveContainerKeywordAlias(keyword)

	if isContainer {
		// It's a container type (list/li/l, arr/ar/a, enum/en/e)
		switch canonical {
		case "list":
			return parseListType(expr, line, column)
		case "arr":
			return parseArrayType(expr, line, column)
		case "enum":
			scalar, err := parseEnumType(expr, line, column)
			if err != nil {
				return ColumnType{}, err
			}
			return ColumnType{Kind: TypeKindScalar, Scalar: scalar}, nil
		}
	}

	// Not a container - try scalar type
	scalar, err := parseScalar(expr, line, column)
	if err != nil {
		return ColumnType{}, err
	}
	return ColumnType{Kind: TypeKindScalar, Scalar: scalar}, nil
}

func parseScalar(expr string, line int, column string) (ScalarType, error) {
	expr = strings.TrimSpace(expr)

	// Special handling for bytes<encoding>
	if strings.HasPrefix(expr, "bytes<") {
		return parseBytesType(expr, line, column)
	}

	// Reject bare "bytes"
	if expr == "bytes" {
		return ScalarType{}, newError(line, column, CodeUnknownType,
			"bytes type requires encoding parameter: bytes<hex> or bytes<b64>")
	}

	// Type names must be lowercase - resolve using alias tables
	if kind, ok := resolveTypeAlias(expr); ok {
		return ScalarType{Kind: kind}, nil
	}
	return ScalarType{}, newError(line, column, CodeUnknownType, "unknown type %q (type names must be lowercase)", expr)
}

func parseBytesType(expr string, line int, column string) (ScalarType, error) {
	// Extract encoding parameter using existing extractGeneric function
	body, rest, err := extractGeneric(expr, "bytes", line, column)
	if err != nil {
		return ScalarType{}, err
	}

	// No suffix allowed (e.g., bytes<hex>[5] is invalid)
	if strings.TrimSpace(rest) != "" {
		return ScalarType{}, newError(line, column, CodeInvalidContainerSyntax,
			"unexpected suffix after bytes type: %q", rest)
	}

	// Encoding must be lowercase - no normalization
	encoding := strings.TrimSpace(body)

	switch encoding {
	case "hex":
		return ScalarType{Kind: ScalarBytesHex}, nil
	case "b64":
		return ScalarType{Kind: ScalarBytesB64}, nil
	default:
		return ScalarType{}, newError(line, column, CodeUnknownType,
			"unknown bytes encoding %q (expected hex or b64, encoding must be lowercase)", body)
	}
}

func parseListType(expr string, line int, column string) (ColumnType, error) {
	// Extract using the actual keyword in the input (might be alias)
	keyword := extractContainerKeyword(expr)
	body, rest, err := extractGeneric(expr, keyword, line, column)
	if err != nil {
		return ColumnType{}, err
	}
	elem, err := parseElement(body, line, column)
	if err != nil {
		return ColumnType{}, err
	}
	rest = strings.TrimSpace(rest)
	fixedLen := 0
	if rest != "" {
		length, perr := parseSingleLength(rest, line, column)
		if perr != nil {
			return ColumnType{}, perr
		}
		fixedLen = length
	}
	return ColumnType{
		Kind: TypeKindList,
		List: ListType{Element: elem, FixedLength: fixedLen},
	}, nil
}

func parseArrayType(expr string, line int, column string) (ColumnType, error) {
	// Extract using the actual keyword in the input (might be alias)
	keyword := extractContainerKeyword(expr)
	body, rest, err := extractGeneric(expr, keyword, line, column)
	if err != nil {
		return ColumnType{}, err
	}
	elem, err := parseElement(body, line, column)
	if err != nil {
		return ColumnType{}, err
	}
	rest = strings.TrimSpace(rest)
	array := ArrayType{Element: elem, Rank: ArrayRankFlexible}
	if rest != "" {
		if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
			return ColumnType{}, newError(line, column, CodeInvalidContainerSyntax, "invalid array suffix %q", rest)
		}
		inner := strings.TrimSpace(rest[1 : len(rest)-1])
		if strings.Contains(inner, ",") {
			parts := strings.Split(inner, ",")
			if len(parts) != 2 {
				return ColumnType{}, newError(line, column, CodeInvalidContainerSyntax, "array shape must specify two integers")
			}
			r, err := parsePositiveInt(strings.TrimSpace(parts[0]))
			if err != nil {
				return ColumnType{}, newError(line, column, CodeInvalidPrefix, "invalid row count: %v", err)
			}
			c, err := parsePositiveInt(strings.TrimSpace(parts[1]))
			if err != nil {
				return ColumnType{}, newError(line, column, CodeInvalidPrefix, "invalid column count: %v", err)
			}
			array.Rank = ArrayRank2DFixed
			array.Rows = r
			array.Cols = c
		} else {
			length, err := parsePositiveInt(inner)
			if err != nil {
				return ColumnType{}, newError(line, column, CodeInvalidPrefix, "invalid array length: %v", err)
			}
			array.Rank = ArrayRank1DFixed
			array.Length = length
		}
	}
	return ColumnType{Kind: TypeKindArray, Array: array}, nil
}

func parseElement(expr string, line int, column string) (ScalarType, error) {
	expr = strings.TrimSpace(expr)

	// Check for container keywords (including aliases)
	keyword := extractContainerKeyword(expr)
	canonical, _ := resolveContainerKeywordAlias(keyword)
	if canonical == "" {
		canonical = keyword
	}

	switch canonical {
	case "list", "arr":
		return ScalarType{}, newError(line, column, CodeNestedContainerNotAllowed, "containers may not nest")
	case "enum":
		return parseEnumType(expr, line, column)
	}

	return parseScalar(expr, line, column)
}

func parseEnumType(expr string, line int, column string) (ScalarType, error) {
	// Extract using the actual keyword in the input (might be alias)
	keyword := extractContainerKeyword(expr)
	body, rest, err := extractGeneric(expr, keyword, line, column)
	if err != nil {
		return ScalarType{}, err
	}
	if strings.TrimSpace(rest) != "" {
		return ScalarType{}, newError(line, column, CodeInvalidEnum, "unexpected suffix after enum declaration")
	}
	entries := strings.Split(body, ",")
	vals := make([]EnumValue, 0, len(entries))
	namesMap := make(map[string]struct{})
	valuesMap := make(map[string]struct{})
	mappedStyle := -1 // -1=unknown, 0=name-only, 1=value=name
	for _, entry := range entries {
		token := strings.TrimSpace(entry)
		if token == "" {
			return ScalarType{}, newError(line, column, CodeInvalidEnum, "empty enum entry")
		}
		if strings.Contains(token, "=") {
			if mappedStyle == 0 {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "mixed enum style: cannot mix name-only and value=name items")
			}
			mappedStyle = 1
			parts := strings.SplitN(token, "=", 2)
			value := strings.TrimSpace(parts[0])
			name := strings.TrimSpace(parts[1])
			if name == "" || value == "" {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "invalid enum mapping %q", token)
			}
			if !isValidIdentifier(name) {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "invalid enum name %q", name)
			}
			if name == "_" {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "null literal '_' cannot be used as enum name")
			}
			if !isValidIdentifier(value) {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "invalid enum value %q", value)
			}
			if value == "_" {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "null literal '_' cannot be used as enum value")
			}
			lowerName := strings.ToLower(name)
			lowerValue := strings.ToLower(value)
			if _, ok := namesMap[lowerName]; ok {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "duplicate enum name %q (case-insensitive)", name)
			}
			// Cross-item collision checks. Each item's name and value are added to the maps
			// AFTER the checks below, so this item's own name/value are never in the maps yet.
			// That means a value matching its own item's name (a "self-pair") is structurally
			// impossible to catch here - which is correct: spec allows self-pairs. Only
			// cross-item collisions are forbidden (value of item i matching name of item j, i!=j).
			// Value must not collide with any existing name (case-insensitive)
			if _, ok := namesMap[lowerValue]; ok {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "enum value %q collides with existing name (case-insensitive)", value)
			}
			// Name must not collide with any existing value (case-insensitive)
			if _, ok := valuesMap[lowerName]; ok {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "enum name %q collides with existing value (case-insensitive)", name)
			}
			// Duplicate values are allowed (spec: values MAY duplicate; first-match decoding applies)
			namesMap[lowerName] = struct{}{}
			valuesMap[lowerValue] = struct{}{}
			vals = append(vals, EnumValue{Index: int32(len(vals)), Name: name, Value: value})
		} else {
			if mappedStyle == 1 {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "mixed enum style: cannot mix name-only and value=name items")
			}
			mappedStyle = 0
			name := token
			if !isValidIdentifier(name) {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "invalid enum name %q", name)
			}
			if name == "_" {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "null literal '_' cannot be used as enum name")
			}
			lowerName := strings.ToLower(name)
			if _, ok := namesMap[lowerName]; ok {
				return ScalarType{}, newError(line, column, CodeInvalidEnum, "duplicate enum name %q (case-insensitive)", name)
			}
			namesMap[lowerName] = struct{}{}
			vals = append(vals, EnumValue{Index: int32(len(vals)), Name: name})
		}
	}
	if len(vals) == 0 {
		return ScalarType{}, newError(line, column, CodeInvalidEnum, "enum must declare at least one value")
	}
	return ScalarType{Kind: ScalarEnum, Enum: &EnumSpec{Values: vals, names: namesMap, values: valuesMap}}, nil
}

func (e *EnumSpec) HasName(value string) bool {
	if e == nil {
		return false
	}
	if e.names == nil {
		return false
	}
	_, ok := e.names[strings.ToLower(value)]
	return ok
}

func (e *EnumSpec) HasValue(value string) bool {
	if e == nil {
		return false
	}
	if e.values == nil {
		return false
	}
	_, ok := e.values[strings.ToLower(value)]
	return ok
}

// HasValues returns true if this enum has explicit values (name=value enum)
func (e *EnumSpec) HasValues() bool {
	if e == nil {
		return false
	}
	return len(e.values) > 0
}

// NameToValue converts a name to its value (for compact encoding)
// Comparison is case-insensitive.
// Returns (value, true) if found, ("", false) if not found or name-only enum
func (e *EnumSpec) NameToValue(name string) (string, bool) {
	if e == nil || len(e.Values) == 0 {
		return "", false
	}
	lowerName := strings.ToLower(name)
	for _, ev := range e.Values {
		if strings.ToLower(ev.Name) == lowerName && ev.Value != "" {
			return ev.Value, true
		}
	}
	return "", false
}

// extractContainerKeyword extracts the keyword from a container type expression.
// Examples: "list<int>" -> "list", "li<str>" -> "li", "en<a,b>" -> "en"
func extractContainerKeyword(expr string) string {
	expr = strings.TrimSpace(expr)
	idx := strings.IndexByte(expr, '<')
	if idx == -1 {
		return expr // No angle bracket, return whole expr
	}
	return strings.TrimSpace(expr[:idx])
}

func extractGeneric(expr, prefix string, line int, column string) (string, string, error) {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, prefix) {
		return "", "", newError(line, column, CodeInvalidContainerSyntax, "%s type must start with %s<...>", prefix, prefix)
	}
	start := len(prefix)
	for start < len(expr) && expr[start] == ' ' {
		start++
	}
	if start >= len(expr) || expr[start] != '<' {
		return "", "", newError(line, column, CodeInvalidContainerSyntax, "%s type missing '<'", prefix)
	}
	depth := 0
	var body strings.Builder
	i := start
	for ; i < len(expr); i++ {
		ch := expr[i]
		if ch == '<' {
			if depth > 0 {
				body.WriteByte(ch)
			}
			depth++
			continue
		}
		if ch == '>' {
			depth--
			if depth == 0 {
				i++
				break
			}
			body.WriteByte(ch)
			continue
		}
		if depth > 0 {
			body.WriteByte(ch)
		}
	}
	if depth != 0 {
		return "", "", newError(line, column, CodeInvalidContainerSyntax, "unterminated %s<>", prefix)
	}
	rest := strings.TrimSpace(expr[i:])
	return body.String(), rest, nil
}

func parseSingleLength(rest string, line int, column string) (int, error) {
	if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
		return 0, newError(line, column, CodeInvalidContainerSyntax, "length suffix must be [N]")
	}
	inner := strings.TrimSpace(rest[1 : len(rest)-1])
	if inner == "" {
		return 0, newError(line, column, CodeInvalidPrefix, "length suffix must contain a positive integer")
	}
	length, err := parsePositiveInt(inner)
	if err != nil {
		return 0, newError(line, column, CodeInvalidPrefix, "invalid length: %v", err)
	}
	return length, nil
}

func parsePositiveInt(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be > 0")
	}
	return n, nil
}
