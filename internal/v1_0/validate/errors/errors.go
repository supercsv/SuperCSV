// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package errors

import "fmt"

// ErrorCode documents a structured validation error code.
type ErrorCode struct {
	Code            string
	Description     string
	Category        string // "string", "container", "numeric", "structural", "temporal", "enum"
	AddedIn         string // version when code was introduced (e.g. "v1.0.0")
	MessageTemplate string // "invalid {type} value: {value}"
}

var (
	CodeInvalidScalar = ErrorCode{
		Code:            "invalid_scalar",
		Description:     "parser could not interpret the scalar literal for the declared type",
		Category:        "numeric",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid {type} value: {value}",
	}
	CodeInvalidBool = ErrorCode{
		Code:            "invalid_bool",
		Description:     "bool column contained a value other than true/false/1/0",
		Category:        "numeric",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid bool value: {value}",
	}
	CodeCSVParseError = ErrorCode{
		Code:            "csv_parse_error",
		Description:     "CSV structure is broken (stray quote, unclosed field, etc.)",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "CSV parse error: {error}",
	}
	CodeInvalidBytesHex = ErrorCode{
		Code:            "invalid_bytes_hex",
		Description:     "bytes<hex> literal was not valid hexadecimal",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid bytes<hex> value: {reason}",
	}
	CodeInvalidBytesB64 = ErrorCode{
		Code:            "invalid_bytes_b64",
		Description:     "bytes<b64> literal was not valid base64",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid bytes<b64> value: {reason}",
	}
	CodeInvalidDecimal = ErrorCode{
		Code:            "invalid_decimal",
		Description:     "decimal literal was malformed or violated scale/precision rules",
		Category:        "numeric",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid decimal: {value}",
	}
	CodeInvalidDate = ErrorCode{
		Code:            "invalid_date",
		Description:     "date literal did not match YYYY-MM-DD or supported variants",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid date format: {value}",
	}
	CodeInvalidTime = ErrorCode{
		Code:            "invalid_time",
		Description:     "time literal had an invalid HH:MM(:SS) component",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid time format: {value}",
	}
	CodeInvalidTimestamp = ErrorCode{
		Code:            "invalid_timestamp",
		Description:     "timestamp literal was malformed or missing timezone info",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid timestamp: {value}",
	}
	CodeInvalidDatetime = ErrorCode{
		Code:            "invalid_datetime",
		Description:     "datetime literal was malformed or contained timezone",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid datetime: {value}",
	}
	CodeInvalidDatetimeTZ = ErrorCode{
		Code:            "invalid_datetimetz",
		Description:     "datetimetz literal was malformed or missing required timezone",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid datetimetz: {value}",
	}
	CodeInvalidDuration = ErrorCode{
		Code:            "invalid_duration",
		Description:     "duration literal was not a valid ISO-8601 duration",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid duration: {value}",
	}
	CodeInvalidUUID = ErrorCode{
		Code:            "invalid_uuid",
		Description:     "value was not a canonical UUID string",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid UUID format: {value}",
	}
	CodeInvalidTimezone = ErrorCode{
		Code:            "invalid_timezone",
		Description:     "timezone value could not be parsed",
		Category:        "temporal",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid timezone: {value}",
	}
	CodeInvalidEnumName = ErrorCode{
		Code:            "invalid_enum_name",
		Description:     "enum name did not match any declared schema member",
		Category:        "enum",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid enum name: {value}",
	}
	CodeInvalidEnumValue = ErrorCode{
		Code:            "invalid_enum_value",
		Description:     "enum numeric value did not map to a defined member",
		Category:        "enum",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid enum value: {value}",
	}
	CodeInvalidContainerSyntax = ErrorCode{
		Code:            "invalid_container_syntax",
		Description:     "container literal syntax was malformed",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid container syntax: {details}",
	}
	CodeInvalidPrefix = ErrorCode{
		Code:            "invalid_prefix",
		Description:     "container prefix was illegal or contradictory",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid prefix: {details}",
	}
	CodeWrongLength = ErrorCode{
		Code:            "wrong_length",
		Description:     "container element count differed from schema or prefix",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "expected {expected} elements, got {actual}",
	}
	CodeWrongShape = ErrorCode{
		Code:            "wrong_shape",
		Description:     "matrix dimensions differed from the declared shape",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "expected shape [{expected}], got [{actual}]",
	}
	CodeNestedContainerNotAllowed = ErrorCode{
		Code:            "nested_container_not_allowed",
		Description:     "nested containers are not allowed by the schema",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "nested containers not allowed",
	}
	CodeColumnCountMismatch = ErrorCode{
		Code:            "column_count_mismatch",
		Description:     "row had a different number of fields than the header",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "expected {expected} columns, got {actual}",
	}
	CodeHeaderContinuationInvalid = ErrorCode{
		Code:            "header_continuation_invalid",
		Description:     "header continuation cannot include blank or comment lines",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "header continuation line cannot be blank or comment",
	}
	CodeHeaderContinuationMalformed = ErrorCode{
		Code:            "header_continuation_malformed",
		Description:     "unexpected EOF after header continuation comma",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "unexpected EOF after header continuation comma",
	}
	CodeHeaderMidTypeSplit = ErrorCode{
		Code:            "header_mid_type_split",
		Description:     "type declaration cannot be split across continuation lines",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "type declaration cannot be split across lines",
	}

	// Quoting and character validation errors
	CodeInvalidUnquotedChar = ErrorCode{
		Code:            "invalid_unquoted_char",
		Description:     "Forbidden character in unquoted string (hash, semicolon, brackets, quotes, newlines)",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "forbidden character {char} in unquoted string",
	}
	CodeQuotedScalarForbidden = ErrorCode{
		Code:            "quoted_scalar_forbidden",
		Description:     "Non-string scalar type quoted when quoting not allowed",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "{type} values must not be quoted",
	}
	CodeQuotedContainerForbidden = ErrorCode{
		Code:            "quoted_container_forbidden",
		Description:     "Container values must not be quoted",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "container values must not be quoted",
	}
	CodeContainerNestingTooDeep = ErrorCode{
		Code:            "container_nesting_too_deep",
		Description:     "Containers nested beyond maximum depth (lists cannot nest; arrays may be 1D or 2D only)",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "container nesting too deep: {details}",
	}

	// Prefix errors
	CodePrefixOnFixedArray = ErrorCode{
		Code:            "prefix_on_fixed_array",
		Description:     "Prefix notation used on fixed-size array or list",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "prefix notation cannot be used with fixed-size arrays",
	}
	CodePrefixOnEmptyArray = ErrorCode{
		Code:            "prefix_on_empty_array",
		Description:     "Prefix notation used on zero-length array",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "prefix notation cannot be used on empty arrays",
	}
	CodePrefixDimensionInvalid = ErrorCode{
		Code:            "prefix_dimension_invalid",
		Description:     "Prefix dimension is not a positive integer (zero, negative, or non-numeric)",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "prefix dimension must be a positive integer",
	}
	CodePrefixDimensionMismatch = ErrorCode{
		Code:            "prefix_dimension_mismatch",
		Description:     "Prefix-declared dimension does not match actual element count",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "prefix length {prefix} does not match actual length {actual}",
	}

	// Comma/delimiter errors
	CodeMissingComma = ErrorCode{
		Code:            "missing_comma",
		Description:     "Missing comma between container elements",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "expected comma between elements",
	}
	CodeTrailingComma = ErrorCode{
		Code:            "trailing_comma",
		Description:     "Trailing comma at end of container",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "trailing comma not allowed in container",
	}
	CodeDoubleComma = ErrorCode{
		Code:            "double_comma",
		Description:     "Consecutive commas in container (missing element)",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "consecutive commas not allowed (missing element)",
	}

	// Array shape errors
	CodeRowLengthMismatch2D = ErrorCode{
		Code:            "row_length_mismatch_2d",
		Description:     "Rows in 2D array have inconsistent lengths (jagged arrays not allowed)",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "row {row} has {actual} columns, expected {expected}",
	}
	CodeEmptyRowIn2DArray = ErrorCode{
		Code:            "empty_row_in_2d_array",
		Description:     "2D array contains an empty row",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "2D array cannot contain empty rows",
	}
	CodeZeroColumnArray = ErrorCode{
		Code:            "zero_column_array",
		Description:     "Array declared with zero columns in schema",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "array cannot have zero columns",
	}

	// Structural errors (CSV/container syntax)
	CodeInvalidStructureQuoteInUnquoted = ErrorCode{
		Code:            "invalid_structure_quote_in_unquoted",
		Description:     "Quote character found inside unquoted field",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: quote in unquoted field",
	}
	CodeInvalidStructureUnclosedQuoted = ErrorCode{
		Code:            "invalid_structure_unclosed_quoted",
		Description:     "Quoted field not properly closed",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unclosed quoted field",
	}
	CodeInvalidStructureUnexpectedChar = ErrorCode{
		Code:            "invalid_structure_unexpected_char",
		Description:     "Unexpected character after closing quote",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unexpected character after quote",
	}
	CodeInvalidStructureMissingBrackets = ErrorCode{
		Code:            "invalid_structure_missing_brackets",
		Description:     "List or array missing opening/closing brackets",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: missing brackets",
	}
	CodeInvalidStructureUnterminatedLiteral = ErrorCode{
		Code:            "invalid_structure_unterminated_literal",
		Description:     "Array or list literal not properly terminated",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unterminated literal",
	}
	CodeInvalidStructureUnterminatedQuoted = ErrorCode{
		Code:            "invalid_structure_unterminated_quoted",
		Description:     "Quoted value in container not properly terminated",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unterminated quoted value",
	}
	CodeInvalidStructureUnterminatedArray = ErrorCode{
		Code:            "invalid_structure_unterminated_array",
		Description:     "Array not properly terminated (missing closing bracket)",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unterminated array",
	}
	CodeInvalidStructureUnterminatedEscape = ErrorCode{
		Code:            "invalid_structure_unterminated_escape",
		Description:     "Escape sequence in string not completed",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: unterminated escape sequence",
	}
	CodeInvalidStructureStrayQuote = ErrorCode{
		Code:            "invalid_structure_stray_quote",
		Description:     "Stray quote character in quoted string (not properly escaped)",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: stray quote",
	}
	CodeInvalidStructureMissingQuotes = ErrorCode{
		Code:            "invalid_structure_missing_quotes",
		Description:     "Quoted string missing opening or closing quotes",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: missing quotes",
	}
	CodeInvalidStructureQuotedTooShort = ErrorCode{
		Code:            "invalid_structure_quoted_too_short",
		Description:     "Quoted string too short to be valid (minimum 2 chars)",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid structure: quoted string too short",
	}

	// Prefix validation errors
	CodeInvalidPrefixNonPositiveLength = ErrorCode{
		Code:            "invalid_prefix_non_positive_length",
		Description:     "Prefix length is zero or negative",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid prefix: non-positive length",
	}
	CodeInvalidPrefixNonPositiveDimensions = ErrorCode{
		Code:            "invalid_prefix_non_positive_dimensions",
		Description:     "Prefix dimensions are zero or negative",
		Category:        "container",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid prefix: non-positive dimensions",
	}

	// Field/schema validation errors
	CodeFieldTooLarge = ErrorCode{
		Code:            "field_too_large",
		Description:     "Field data exceeds the configured maximum field size",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "field size exceeds maximum {max}",
	}
	CodeInvalidFieldSize = ErrorCode{
		Code:            "invalid_field_size",
		Description:     "Field size parameter out of valid range",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid field size: {size} (expected >= 0)",
	}
	CodeInvalidEnumCodeNonNumeric = ErrorCode{
		Code:            "invalid_enum_code_non_numeric",
		Description:     "Enum code must be numeric",
		Category:        "enum",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid enum code: non-numeric {code}",
	}

	// File/header validation errors
	CodeInvalidHeaderContinuation = ErrorCode{
		Code:            "invalid_header_continuation",
		Description:     "Header continuation contains blank lines or comments (forbidden by spec)",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "header continuation: unexpected EOF after comma",
	}
	CodeInvalidHeaderMidTypeSplit = ErrorCode{
		Code:            "invalid_header_mid_type_split",
		Description:     "Type declaration split across header continuation lines",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "header mid-type split: unclosed angle bracket at line {line}",
	}

	// System-level errors
	CodeInvalidHeaderSchema = ErrorCode{
		Code:            "invalid_header_schema",
		Description:     "Header schema is malformed or invalid",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid header: {error}",
	}
	CodeMaxErrorsReached = ErrorCode{
		Code:            "max_errors_reached",
		Description:     "Validation stopped: maximum error limit reached",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "validation stopped max errors limit reached",
	}
	CodeRowStillInUse = ErrorCode{
		Code:            "row_still_in_use",
		Description:     "Row buffer still in use when trying to read next row",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "row still in use",
	}
	CodeInlineCommentsNotAllowed = ErrorCode{
		Code:            "inline_comments_not_allowed",
		Description:     "Inline comments are not allowed in SuperCSV format",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "inline comments not allowed",
	}
	CodeNULByteNotAllowed = ErrorCode{
		Code:            "nul_byte_not_allowed",
		Description:     "NUL byte (0x00) not allowed in SuperCSV data",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "NUL byte not allowed",
	}
	CodeBOMNotAllowed = ErrorCode{
		Code:            "bom_not_allowed",
		Description:     "UTF-8 BOM (byte order mark) not allowed",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "utf-8 BOM not allowed",
	}

	// Annotation errors
	CodeDuplicateAnnotation = ErrorCode{
		Code:            "duplicate_annotation",
		Description:     "Field has more than one comment or metadata annotation block",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "multiple {type} blocks on field",
	}
	// Enum schema errors (header-level, fatal)
	CodeInvalidEnumSchema = ErrorCode{
		Code:            "invalid_enum_schema",
		Description:     "Enum type definition in the header is structurally invalid (duplicate names, mixed styles, cross-pair collision, etc.)",
		Category:        "enum",
		AddedIn:         "v1.0.0",
		MessageTemplate: "invalid enum schema: {error}",
	}
	CodeAnnotationAfterTrailingComma = ErrorCode{
		Code:            "annotation_after_trailing_comma",
		Description:     "Inline annotation appears after the trailing continuation comma with no field value on the same line",
		Category:        "structural",
		AddedIn:         "v1.0.0",
		MessageTemplate: "annotation after trailing comma: annotation must appear on the continuation line, not before it",
	}
	CodeUnicodeWhitespaceInUnquoted = ErrorCode{
		Code:            "unicode_whitespace_in_unquoted",
		Description:     "Unicode whitespace (e.g. NBSP U+00A0) appears at the start or end of an unquoted string field; use a quoted string",
		Category:        "string",
		AddedIn:         "v1.0.0",
		MessageTemplate: "unexpected whitespace character at the start or end of an unquoted string; use a quoted string",
	}
)

// ValidationError matches the canonical error payload described in validator-spec.md.
// File may be left empty for now; Row refers to the physical line number.
// RowEnd is the ending line number for multi-line constructs (e.g., multi-line headers).
// If RowEnd is 0 or equal to Row, the error refers to a single line.
type ValidationError struct {
	File   string
	Row    int
	RowEnd int
	Column string
	Code   string
	Msg    string
	Fatal  bool // True if this error is fatal (stops validation)
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.File != "" {
		if e.RowEnd > 0 && e.RowEnd != e.Row {
			return fmt.Sprintf("%s:%d-%d: %s", e.File, e.Row, e.RowEnd, e.Msg)
		}
		return fmt.Sprintf("%s:%d: %s", e.File, e.Row, e.Msg)
	}
	if e.RowEnd > 0 && e.RowEnd != e.Row {
		return fmt.Sprintf("line %d-%d: %s", e.Row, e.RowEnd, e.Msg)
	}
	return fmt.Sprintf("line %d: %s", e.Row, e.Msg)
}

// New creates a validation error with formatted message.
func New(row int, column, code, format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Row:    row,
		Column: column,
		Code:   code,
		Msg:    fmt.Sprintf(format, args...),
	}
}
