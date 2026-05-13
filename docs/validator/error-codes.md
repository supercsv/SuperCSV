# SuperCSV Error Code Catalog

**Version**: v1.0.0  
**Last Updated**: 2026-04-18

## Overview

This document catalogs all error codes in the SuperCSV validator. Error codes provide stable, machine-readable identifiers for programmatic error handling.

For teaching examples, see [error-examples.md](error-examples.md).

## Programmatic Usage

```go
import (
    "errors"
    "github.com/supercsv/supercsv/supr"
)

// Check for validation errors using errors.As
var ve *supr.ValidationError
if errors.As(err, &ve) {
    fmt.Printf("row %d, col %d: %s\n", ve.Row, ve.Col, ve.Reason)
}

// Check error category using errors.Is
if errors.Is(err, supr.ErrValidation) {
    // Handle validation error
}
if errors.Is(err, supr.ErrMaxErrors) {
    // Validation stopped at error limit
}
```

## Error Categories

- [Numeric Errors](#numeric-errors) - Type validation for numbers
- [String Errors](#string-errors) - String format and encoding
- [Temporal Errors](#temporal-errors) - Date, time, timestamp, duration
- [Enum Errors](#enum-errors) - Enumeration validation
- [Container Errors](#container-errors) - Lists, arrays, matrices
- [Structural Errors](#structural-errors) - CSV/container syntax
- [System Errors](#system-errors) - File-level and system issues

---

## Numeric Errors

### CodeInvalidScalar
- **Code**: `invalid_scalar`
- **Message Template**: `invalid {type} value: {value}`
- **Category**: numeric
- **Added**: v1.0.0
- **Description**: Parser could not interpret the scalar literal for the declared type
- **Example**: Field declared as `int` but contains `"abc"`

### CodeInvalidBool
- **Code**: `invalid_bool`
- **Message Template**: `invalid bool value: {value}`
- **Category**: numeric
- **Added**: v1.0.0
- **Description**: Boolean column contained a value other than true/false/1/0
- **Example**: Field declared as `bool` but contains `"maybe"`

### CodeInvalidDecimal
- **Code**: `invalid_decimal`
- **Message Template**: `invalid decimal: {value}`
- **Category**: numeric
- **Added**: v1.0.0
- **Description**: Decimal literal was malformed or violated scale/precision rules
- **Example**: Field declared as `decimal(10,2)` but contains `"12.345"`

---

## String Errors

### CodeInvalidBytesHex
- **Code**: `invalid_bytes_hex`
- **Message Template**: `invalid bytes<hex> value: {reason}`
- **Category**: string
- **Added**: v1.0.0
- **Description**: bytes\<hex\> literal was not valid hexadecimal
- **Example**: Field declared as `bytes<hex>` but contains `"ZZZZ"` (not hex)

### CodeInvalidBytesB64
- **Code**: `invalid_bytes_b64`
- **Message Template**: `invalid bytes<b64> value: {reason}`
- **Category**: string
- **Added**: v1.0.0
- **Description**: bytes\<b64\> literal was not valid base64
- **Example**: Field declared as `bytes<b64>` but contains `"!!!"` (not base64)

### CodeInvalidUUID
- **Code**: `invalid_uuid`
- **Message Template**: `invalid UUID format: {value}`
- **Category**: string
- **Added**: v1.0.0
- **Description**: Value was not a canonical UUID string
- **Example**: Field declared as `uuid` but contains `"not-a-uuid"`

### CodeInvalidUnquotedChar
- **Code**: `invalid_unquoted_char`
- **Message Template**: `forbidden character {char} in unquoted string`
- **Category**: string
- **Added**: v1.0.0
- **Description**: Forbidden character in unquoted string (hash, semicolon, brackets, quotes, newlines)
- **Example**: Unquoted field contains `Jo"hn` (quote not allowed)

### CodeQuotedScalarForbidden
- **Code**: `quoted_scalar_forbidden`
- **Message Template**: `{type} values must not be quoted`
- **Category**: string
- **Added**: v1.0.0
- **Description**: Non-string scalar type quoted when quoting not allowed
- **Example**: Integer field contains `"123"` when schema disallows quoting

### CodeQuotedContainerForbidden
- **Code**: `quoted_container_forbidden`
- **Message Template**: `container values must not be quoted`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Container values must not be quoted
- **Example**: Array field contains `"[1,2,3]"` instead of `[1,2,3]`

---

## Temporal Errors

### CodeInvalidDate
- **Code**: `invalid_date`
- **Message Template**: `invalid date format: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: Date literal did not match YYYY-MM-DD or supported variants
- **Example**: Field declared as `date` but contains `"2026-13-45"` (invalid month/day)

### CodeInvalidTime
- **Code**: `invalid_time`
- **Message Template**: `invalid time format: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: Time literal had an invalid HH:MM(:SS) component
- **Example**: Field declared as `time` but contains `"25:99:00"` (invalid hours/minutes)

### CodeInvalidTimestamp
- **Code**: `invalid_timestamp`
- **Message Template**: `invalid timestamp: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: Timestamp literal was malformed or missing timezone info
- **Example**: Field declared as `timestamp` but contains `"2026-01-18"` (missing time)

### CodeInvalidDuration
- **Code**: `invalid_duration`
- **Message Template**: `invalid duration: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: Duration literal was not a valid ISO-8601 duration
- **Example**: Field declared as `duration` but contains `"5 hours"` (not ISO-8601)

### CodeInvalidTimezone
- **Code**: `invalid_timezone`
- **Message Template**: `invalid timezone: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: Timezone value could not be parsed
- **Example**: Field contains `"InvalidTZ"` instead of `"America/New_York"`

### CodeInvalidDatetime
- **Code**: `invalid_datetime`
- **Message Template**: `invalid datetime: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: datetime literal was malformed or contained timezone
- **Example**: Field declared as `datetime` but contains timezone info

### CodeInvalidDatetimeTZ
- **Code**: `invalid_datetimetz`
- **Message Template**: `invalid datetimetz: {value}`
- **Category**: temporal
- **Added**: v1.0.0
- **Description**: datetimetz literal was malformed or missing required timezone
- **Example**: Field declared as `datetimetz` but contains `"2026-01-18T10:30:00"` (no timezone)

---

## Enum Errors

### CodeInvalidEnumName
- **Code**: `invalid_enum_name`
- **Message Template**: `invalid enum name: {value}`
- **Category**: enum
- **Added**: v1.0.0
- **Description**: Enum name did not match any declared schema member
- **Example**: Enum field declares names `"red"`, `"green"`, `"blue"` but contains `"purple"`

### CodeInvalidEnumValue
- **Code**: `invalid_enum_value`
- **Message Template**: `invalid enum value: {value}`
- **Category**: enum
- **Added**: v1.0.0
- **Description**: Enum numeric value did not map to a defined member
- **Example**: Enum with values 1, 2, 3 but contains `99`

### CodeInvalidEnumCodeNonNumeric
- **Code**: `invalid_enum_code_non_numeric`
- **Message Template**: `invalid enum code: non-numeric {code}`
- **Category**: enum
- **Added**: v1.0.0
- **Description**: Enum code must be numeric
- **Example**: Schema defines enum with code `"abc"` instead of number

---

## Container Errors

### CodeInvalidContainerSyntax
- **Code**: `invalid_container_syntax`
- **Message Template**: `invalid container syntax: {details}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Container literal syntax was malformed
- **Example**: Array contains `[1,2,` (missing closing bracket)

### CodeInvalidPrefix
- **Code**: `invalid_prefix`
- **Message Template**: `invalid prefix: {details}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Container prefix was illegal or contradictory
- **Example**: Array contains `[0][1,2,3]` (zero prefix invalid)

### CodeWrongLength
- **Code**: `wrong_length`
- **Message Template**: `expected {expected} elements, got {actual}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Container element count differed from schema or prefix
- **Example**: Array declared as `[3]int` but contains `[1,2]` (2 elements)

### CodeWrongShape
- **Code**: `wrong_shape`
- **Message Template**: `expected shape [{expected}], got [{actual}]`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Matrix dimensions differed from the declared shape
- **Example**: Matrix declared as `[2,3]int` but actual data is 3x2

### CodeNestedContainerNotAllowed
- **Code**: `nested_container_not_allowed`
- **Message Template**: `nested containers not allowed`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Nested containers are not allowed by the schema
- **Example**: List contains `[[1,2],[3,4]]` (nested arrays forbidden in lists)

### CodeContainerNestingTooDeep
- **Code**: `container_nesting_too_deep`
- **Message Template**: `container nesting too deep: {details}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Containers nested beyond maximum depth (lists cannot nest; arrays may be 1D or 2D only)
- **Example**: Array contains `[[[1]]]` (3D not allowed)

### CodePrefixOnFixedArray
- **Code**: `prefix_on_fixed_array`
- **Message Template**: `prefix notation cannot be used with fixed-size arrays`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix notation used on fixed-size array or list
- **Example**: Field declared as `[3]int` but contains `[3][1,2,3]` (prefix redundant)

### CodePrefixOnEmptyArray
- **Code**: `prefix_on_empty_array`
- **Message Template**: `prefix notation cannot be used on empty arrays`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix notation used on zero-length array
- **Example**: Field contains `[0][]` (prefix on empty array forbidden)

### CodePrefixDimensionInvalid
- **Code**: `prefix_dimension_invalid`
- **Message Template**: `prefix dimension must be a positive integer`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix dimension is not a positive integer (zero, negative, or non-numeric)
- **Example**: Field contains `-1:[1,2]` or `abc:[1,2]`

### CodePrefixDimensionMismatch
- **Code**: `prefix_dimension_mismatch`
- **Message Template**: `prefix length {prefix} does not match actual length {actual}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix-declared dimension does not match actual element count
- **Example**: Field contains `[5][1,2,3]` (prefix says 5, actual is 3)

### CodeInvalidPrefixNonPositiveLength
- **Code**: `invalid_prefix_non_positive_length`
- **Message Template**: `invalid prefix: non-positive length`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix length is zero or negative
- **Example**: Field contains `[0][1,2]` or `[-5][1,2]`

### CodeInvalidPrefixNonPositiveDimensions
- **Code**: `invalid_prefix_non_positive_dimensions`
- **Message Template**: `invalid prefix: non-positive dimensions`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Prefix dimensions are zero or negative
- **Example**: 2D array contains `2x0:[[1,2]]` (zero columns invalid)

### CodeMissingComma
- **Code**: `missing_comma`
- **Message Template**: `expected comma between elements`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Missing comma between container elements
- **Example**: Array contains `[1 2 3]` instead of `[1,2,3]`

### CodeTrailingComma
- **Code**: `trailing_comma`
- **Message Template**: `trailing comma not allowed in container`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Trailing comma at end of container
- **Example**: Array contains `[1,2,3,]` (trailing comma after 3)

### CodeDoubleComma
- **Code**: `double_comma`
- **Message Template**: `consecutive commas not allowed (missing element)`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Consecutive commas in container (missing element)
- **Example**: Array contains `[1,,3]` (double comma implies missing element)

### CodeRowLengthMismatch2D
- **Code**: `row_length_mismatch_2d`
- **Message Template**: `row {row} has {actual} columns, expected {expected}`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Rows in 2D array have inconsistent lengths (jagged arrays not allowed)
- **Example**: 2D array `[[1,2],[3,4,5]]` (first row has 2, second has 3)

### CodeEmptyRowIn2DArray
- **Code**: `empty_row_in_2d_array`
- **Message Template**: `2D array cannot contain empty rows`
- **Category**: container
- **Added**: v1.0.0
- **Description**: 2D array contains an empty row
- **Example**: 2D array contains `[[1,2],[],[3,4]]` (middle row empty)

### CodeZeroColumnArray
- **Code**: `zero_column_array`
- **Message Template**: `array cannot have zero columns`
- **Category**: container
- **Added**: v1.0.0
- **Description**: Array declared with zero columns in schema
- **Example**: Schema declares `[2,0]int` (zero columns forbidden)

---

## Structural Errors

### CodeCSVParseError
- **Code**: `csv_parse_error`
- **Message Template**: `CSV parse error: {error}`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: CSV structure is broken (stray quote, unclosed field, etc.)
- **Example**: Row contains unmatched quotes or malformed CSV syntax

### CodeColumnCountMismatch
- **Code**: `column_count_mismatch`
- **Message Template**: `expected {expected} columns, got {actual}`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Row had a different number of fields than the header
- **Example**: Header has 3 columns but data row has 4

### CodeHeaderContinuationInvalid
- **Code**: `header_continuation_invalid`
- **Message Template**: `header continuation line cannot be blank or comment`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Header continuation cannot include blank or comment lines
- **Example**: Multi-line header with blank line in continuation

### CodeHeaderContinuationMalformed
- **Code**: `header_continuation_malformed`
- **Message Template**: `unexpected EOF after header continuation comma`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Unexpected EOF after header continuation comma
- **Example**: Header ends with `,` but no continuation line follows

### CodeHeaderMidTypeSplit
- **Code**: `header_mid_type_split`
- **Message Template**: `type declaration cannot be split across lines`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Type declaration cannot be split across continuation lines
- **Example**: Header has `Name:li` on line 1, `st<int>` on line 2 (type split)

### CodeInvalidStructureQuoteInUnquoted
- **Code**: `invalid_structure_quote_in_unquoted`
- **Message Template**: `invalid structure: quote in unquoted field`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Quote character found inside unquoted field
- **Example**: Field contains `Jo"hn` (quote in unquoted text)

### CodeInvalidStructureUnclosedQuoted
- **Code**: `invalid_structure_unclosed_quoted`
- **Message Template**: `invalid structure: unclosed quoted field`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Quoted field not properly closed
- **Example**: Field contains `"John` (missing closing quote)

### CodeInvalidStructureUnexpectedChar
- **Code**: `invalid_structure_unexpected_char`
- **Message Template**: `invalid structure: unexpected character after quote`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Unexpected character after closing quote
- **Example**: Field contains `"John"xyz` (xyz after closing quote)

### CodeInvalidStructureMissingBrackets
- **Code**: `invalid_structure_missing_brackets`
- **Message Template**: `invalid structure: missing brackets`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: List or array missing opening/closing brackets
- **Example**: Field contains `1,2,3` instead of `[1,2,3]`

### CodeInvalidStructureUnterminatedLiteral
- **Code**: `invalid_structure_unterminated_literal`
- **Message Template**: `invalid structure: unterminated literal`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Array or list literal not properly terminated
- **Example**: Field contains `[1,2,3` (missing closing bracket)

### CodeInvalidStructureUnterminatedQuoted
- **Code**: `invalid_structure_unterminated_quoted`
- **Message Template**: `invalid structure: unterminated quoted value`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Quoted value in container not properly terminated
- **Example**: Container element contains `"abc` (missing closing quote)

### CodeInvalidStructureUnterminatedArray
- **Code**: `invalid_structure_unterminated_array`
- **Message Template**: `invalid structure: unterminated array`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Array not properly terminated (missing closing bracket)
- **Example**: 2D array contains `[[1,2],[3,4]` (missing outer closing bracket)

### CodeInvalidStructureUnterminatedEscape
- **Code**: `invalid_structure_unterminated_escape`
- **Message Template**: `invalid structure: unterminated escape sequence`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Escape sequence in string not completed
- **Example**: String contains `"abc\` (backslash at end, no escape char)

### CodeInvalidStructureStrayQuote
- **Code**: `invalid_structure_stray_quote`
- **Message Template**: `invalid structure: stray quote`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Stray quote character in quoted string (not properly escaped)
- **Example**: Quoted string contains `"John"s"` (stray quote after "John")

### CodeInvalidStructureMissingQuotes
- **Code**: `invalid_structure_missing_quotes`
- **Message Template**: `invalid structure: missing quotes`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Quoted string missing opening or closing quotes
- **Example**: Field requires quoting but contains `John Doe` instead of `"John Doe"`

### CodeInvalidStructureQuotedTooShort
- **Code**: `invalid_structure_quoted_too_short`
- **Message Template**: `invalid structure: quoted string too short`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Quoted string too short to be valid (minimum 2 chars for `""`)
- **Example**: Field contains `"` (single quote, not a valid quoted string)

### CodeInvalidFieldSize
- **Code**: `invalid_field_size`
- **Message Template**: `invalid field size: {size} (expected >= 0)`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Field size parameter out of valid range
- **Example**: Schema declares field with negative size

### CodeInvalidHeaderContinuation
- **Code**: `invalid_header_continuation`
- **Message Template**: `header continuation: unexpected EOF after comma`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Header continuation contains blank lines or comments (forbidden by spec)
- **Example**: Multi-line header has blank line in continuation

### CodeInvalidHeaderMidTypeSplit
- **Code**: `invalid_header_mid_type_split`
- **Message Template**: `header mid-type split: unclosed angle bracket at line {line}`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Type declaration split across header continuation lines
- **Example**: Type `list<int>` split across two header lines

---

## System Errors

### CodeRowStillInUse
- **Code**: `row_still_in_use`
- **Message Template**: `row still in use`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Row buffer still in use when trying to read next row
- **Example**: Previous row not released before calling `Next()`

### CodeInlineCommentsNotAllowed
- **Code**: `inline_comments_not_allowed`
- **Message Template**: `inline comments not allowed`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Inline comments are not allowed in SuperCSV format
- **Example**: Data row contains `123 # this is a comment` (inline comment after value)

### CodeNULByteNotAllowed
- **Code**: `nul_byte_not_allowed`
- **Message Template**: `NUL byte not allowed`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: NUL byte (0x00) not allowed in SuperCSV data
- **Example**: File contains binary null byte in data

### CodeBOMNotAllowed
- **Code**: `bom_not_allowed`
- **Message Template**: `utf-8 BOM not allowed`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: UTF-8 BOM (byte order mark) not allowed
- **Example**: File starts with UTF-8 BOM bytes (EF BB BF)

### CodeFieldTooLarge
- **Code**: `field_too_large`
- **Message Template**: `field size exceeds maximum {max}`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Field data exceeds the configured maximum field size
- **Example**: Field exceeds `--max-field-size` limit

### CodeInvalidHeaderSchema
- **Code**: `invalid_header_schema`
- **Message Template**: `invalid header: {error}`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Header schema is malformed or invalid
- **Example**: Header contains `Name:` (missing type) or `Name:invalidtype`

### CodeMaxErrorsReached
- **Code**: `max_errors_reached`
- **Message Template**: `validation stopped max errors limit reached`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Validation stopped: maximum error limit reached
- **Example**: `--max-errors 10` set and 10 errors found

### CodeDuplicateAnnotation
- **Code**: `duplicate_annotation`
- **Message Template**: `multiple {type} blocks on field`
- **Category**: structural
- **Added**: v1.0.0
- **Description**: Field has more than one comment or metadata annotation block
- **Example**: Header field has two comment blocks before it

---

## Error Code Summary

**Total Error Codes**: 64

**By Category**:
- Numeric: 3
- String: 5
- Temporal: 7
- Enum: 3
- Container: 19
- Structural: 27

**By Version**:
- v1.0.0: 64 codes (initial public release)

---

## See Also

- [error-examples.md](error-examples.md) — Teaching examples for each category

---

*Last updated: 2026-04-18*
*SuperCSV v1.0.0*
