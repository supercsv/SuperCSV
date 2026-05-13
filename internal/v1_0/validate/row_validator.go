// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"
	"strings"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/validate/dfa"
	"github.com/supercsv/supercsv/internal/v1_0/validate/errors"
)

// columnValidator bundles all validation data for a single column
// Pre-binds function pointers, error codes, and enum validators to eliminate runtime lookups
// Size: 48 bytes - fits in single 64-byte cache line for optimal locality
type columnValidator struct {
	validateFn    func([]byte) bool    // 8 bytes - scalar validator (nil for containers)
	errorFn       func([]byte) string  // 8 bytes - error message generator
	errorCode     string               // 16 bytes - pre-bound error code (hot path optimization)
	enumValidator *dfa.EnumValidator   // 8 bytes - enum validator (nil for non-enum types)
	scalarKind    headerdef.ScalarKind // 8 bytes - element type for error mapping
}

// RowValidator orchestrates validation for CSV data rows using column headerdef.
// It dispatches to appropriate validators (scalar/list/array) based on column types
// and collects validation errors with row/column context.
type RowValidator struct {
	headerDef    *headerdef.HeaderDef
	validators   []columnValidator // Consolidated validation data (48 bytes per column)
	maxFieldSize int
}

// alwaysTrue is used for string type (validation done by quote parsing layer)
func alwaysTrue([]byte) bool { return true }

// Thin errorFn adapters for types whose error message does not depend on the value.
// These satisfy func([]byte) string while discarding the unused argument.
func boolErrorFor([]byte) string       { return dfa.BoolError() }
func floatErrorFor([]byte) string      { return dfa.FloatError() }
func decimalErrorFor([]byte) string    { return dfa.DecimalError() }
func dateErrorFor([]byte) string       { return dfa.DateError() }
func timeErrorFor([]byte) string       { return dfa.TimeError() }
func timestampErrorFor([]byte) string  { return dfa.TimestampError() }
func datetimeErrorFor([]byte) string   { return dfa.DatetimeError() }
func datetimeTZErrorFor([]byte) string { return dfa.DatetimeTZError() }
func durationErrorFor([]byte) string   { return dfa.DurationError() }
func uuidErrorFor([]byte) string       { return dfa.UUIDError() }
func bytesHexErrorFor([]byte) string   { return dfa.BytesHexError() }
func bytesB64ErrorFor([]byte) string   { return dfa.BytesB64Error() }
func timezoneErrorFor([]byte) string   { return dfa.TimezoneError() }
func enumErrorFor([]byte) string       { return dfa.EnumError() }
func emptyErrorFor([]byte) string      { return "" }

// NewRowValidator creates a validator for the given headerdef.
// Pre-binds per-column validator function pointers, error codes, and enum validators.
// Optimized for cache locality: all validation data in 48-byte structs.
func NewRowValidator(hdef *headerdef.HeaderDef, maxFieldSize int) *RowValidator {
	rv := &RowValidator{
		headerDef:    hdef,
		validators:   make([]columnValidator, len(hdef.Columns)),
		maxFieldSize: maxFieldSize,
	}

	// Build column validators: pre-bind all validation data at construction time
	for colIdx, col := range hdef.Columns {
		validator := &rv.validators[colIdx]

		// Set scalarKind for error code mapping (ALL column types - scalar and container elements)
		if col.Type.Kind == headerdef.TypeKindScalar {
			validator.scalarKind = col.Type.Scalar.Kind
		} else if col.Type.Kind == headerdef.TypeKindList {
			validator.scalarKind = col.Type.List.Element.Kind
		} else if col.Type.Kind == headerdef.TypeKindArray {
			validator.scalarKind = col.Type.Array.Element.Kind
		}

		// Build enum validator if needed (scalar, list<enum>, arr<enum>)
		var enumSpec *headerdef.EnumSpec
		if col.Type.Kind == headerdef.TypeKindScalar && col.Type.Scalar.Kind == headerdef.ScalarEnum {
			enumSpec = col.Type.Scalar.Enum
		} else if col.Type.Kind == headerdef.TypeKindList && col.Type.List.Element.Kind == headerdef.ScalarEnum {
			enumSpec = col.Type.List.Element.Enum
		} else if col.Type.Kind == headerdef.TypeKindArray && col.Type.Array.Element.Kind == headerdef.ScalarEnum {
			enumSpec = col.Type.Array.Element.Enum
		}

		if enumSpec != nil {
			validator.enumValidator = dfa.NewEnumValidator(enumSpec)
		}

		// Pre-bind scalar validators, error functions, AND error codes (only for scalar columns).
		// validateFn is later invoked as the primary scalar type check in validateScalarField,
		// avoiding per-row type switches in the hot path.
		if col.Type.Kind == headerdef.TypeKindScalar {
			switch col.Type.Scalar.Kind {
			case headerdef.ScalarBool:
				validator.validateFn = dfa.ValidateBool
				validator.errorFn = boolErrorFor
				validator.errorCode = errors.CodeInvalidBool.Code
			case headerdef.ScalarInt:
				validator.validateFn = dfa.ValidateInt
				validator.errorFn = dfa.IntErrorFor
				validator.errorCode = errors.CodeInvalidScalar.Code
			case headerdef.ScalarFloat:
				validator.validateFn = dfa.ValidateFloat
				validator.errorFn = floatErrorFor
				validator.errorCode = errors.CodeInvalidScalar.Code
			case headerdef.ScalarDecimal:
				validator.validateFn = dfa.ValidateDecimal
				validator.errorFn = decimalErrorFor
				validator.errorCode = errors.CodeInvalidDecimal.Code
			case headerdef.ScalarDate:
				validator.validateFn = dfa.ValidateDate
				validator.errorFn = dateErrorFor
				validator.errorCode = errors.CodeInvalidDate.Code
			case headerdef.ScalarTime:
				validator.validateFn = dfa.ValidateTime
				validator.errorFn = timeErrorFor
				validator.errorCode = errors.CodeInvalidTime.Code
			case headerdef.ScalarTimestamp:
				validator.validateFn = dfa.ValidateTimestamp
				validator.errorFn = timestampErrorFor
				validator.errorCode = errors.CodeInvalidTimestamp.Code
			case headerdef.ScalarDatetime:
				validator.validateFn = dfa.ValidateDatetime
				validator.errorFn = datetimeErrorFor
				validator.errorCode = errors.CodeInvalidDatetime.Code
			case headerdef.ScalarDatetimeTZ:
				validator.validateFn = dfa.ValidateDatetimeTZ
				validator.errorFn = datetimeTZErrorFor
				validator.errorCode = errors.CodeInvalidDatetimeTZ.Code
			case headerdef.ScalarDuration:
				validator.validateFn = dfa.ValidateDuration
				validator.errorFn = durationErrorFor
				validator.errorCode = errors.CodeInvalidDuration.Code
			case headerdef.ScalarUUID:
				validator.validateFn = dfa.ValidateUUID
				validator.errorFn = uuidErrorFor
				validator.errorCode = errors.CodeInvalidUUID.Code
			case headerdef.ScalarBytesHex:
				validator.validateFn = dfa.ValidateBytesHex
				validator.errorFn = bytesHexErrorFor
				validator.errorCode = errors.CodeInvalidBytesHex.Code
			case headerdef.ScalarBytesB64:
				validator.validateFn = dfa.ValidateBytesB64
				validator.errorFn = bytesB64ErrorFor
				validator.errorCode = errors.CodeInvalidBytesB64.Code
			case headerdef.ScalarString:
				// String validation done by quote parsing layer
				validator.validateFn = alwaysTrue
				validator.errorFn = emptyErrorFor
				validator.errorCode = errors.CodeInvalidScalar.Code
			case headerdef.ScalarTimezone:
				validator.validateFn = dfa.ValidateTimezone
				validator.errorFn = timezoneErrorFor
				validator.errorCode = errors.CodeInvalidScalar.Code
			case headerdef.ScalarEnum:
				// Enum validator already pre-built above
				validator.validateFn = validator.enumValidator.Validate
				validator.errorFn = enumErrorFor
				validator.errorCode = errors.CodeInvalidEnumValue.Code
			}
		}
	}

	return rv
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateCol validates a single column field from a row
// This is the core per-column validation logic extracted for unrolling optimization
func (v *RowValidator) validateCol(field []byte, isQuoted bool, colIdx int, rowIndex int, errs *[]*errors.ValidationError) {
	col := v.headerDef.Columns[colIdx]

	// 1. FIRST: Check quoting violations
	if isQuoted {
		switch col.Type.Kind {
		case headerdef.TypeKindScalar:
			// Check if it's a string-like scalar (string or timezone)
			// Per spec: timezone follows string rules and may be quoted
			if col.Type.Scalar.Kind == headerdef.ScalarString || col.Type.Scalar.Kind == headerdef.ScalarTimezone {
				// Strings and timezones MAY be quoted - allowed
			} else {
				// Non-string scalars (int, float, bool, etc.) MUST NOT be quoted
				*errs = append(*errs, &errors.ValidationError{
					Row:    rowIndex,
					Column: col.Name,
					Code:   errors.CodeQuotedScalarForbidden.Code,
					Msg:    "scalar values must not be quoted",
				})
				return
			}
		case headerdef.TypeKindList, headerdef.TypeKindArray:
			// Containers MUST NOT be quoted
			*errs = append(*errs, &errors.ValidationError{
				Row:    rowIndex,
				Column: col.Name,
				Code:   errors.CodeQuotedContainerForbidden.Code,
				Msg:    "container values must not be quoted",
			})
			return
		}
	}

	// 2. SECOND: Handle null literal (only "_" is null)
	if v.isNullField(field) {
		// Null is allowed for all types (spec v1.0)
		return // Skip type validation for null
	}

	// 3. THIRD: Check field size limits
	if v.maxFieldSize > 0 && len(field) > v.maxFieldSize {
		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: col.Name,
			Code:   errors.CodeFieldTooLarge.Code,
			Msg:    fmt.Sprintf("field size %d exceeds maximum %d", len(field), v.maxFieldSize),
		})
		return
	}

	// 4. FINALLY: Type-specific validation (DFA, container parsing, etc.)
	v.validateField(field, isQuoted, col, rowIndex, errs)
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// ValidateRow validates a single data row against the headerdef.
// Returns a slice of validation errors (nil if row is valid).
// Each error includes row number, column name, and descriptive message.
//
// Parameters:
//   - fields: the field values as byte slices (deferred materialization)
//   - quoted: parallel slice indicating which fields were quoted in the CSV
//   - commentCounts: per-field comment annotation block counts (nil to skip check)
//   - metaCounts: per-field metadata annotation block counts (nil to skip check)
//   - rowIndex: the line number of this row (for error reporting)
func (v *RowValidator) ValidateRow(fields [][]byte, quoted []bool, commentCounts []int8, metaCounts []int8, rowIndex int) []*errors.ValidationError {
	// Check field count matches schema
	if len(fields) != len(v.headerDef.Columns) {
		return []*errors.ValidationError{{
			Row:    rowIndex,
			Column: "rowErr",
			Code:   errors.CodeColumnCountMismatch.Code,
			Msg:    fmt.Sprintf("expected %d fields, got %d", len(v.headerDef.Columns), len(fields)),
		}}
	}

	var errs []*errors.ValidationError

	// Check for duplicate annotation blocks (at most one comment, one metadata per field)
	if commentCounts != nil {
		for i, cc := range commentCounts {
			if cc > 1 {
				errs = append(errs, &errors.ValidationError{
					Row:    rowIndex,
					Column: v.headerDef.Columns[i].Name,
					Code:   errors.CodeDuplicateAnnotation.Code,
					Msg:    "multiple comment blocks on field",
				})
			}
		}
	}
	if metaCounts != nil {
		for i, mc := range metaCounts {
			if mc > 1 {
				errs = append(errs, &errors.ValidationError{
					Row:    rowIndex,
					Column: v.headerDef.Columns[i].Name,
					Code:   errors.CodeDuplicateAnnotation.Code,
					Msg:    "multiple metadata blocks on field",
				})
			}
		}
	}

	// Validate each field according to its column type
	for i := range fields {
		v.validateCol(fields[i], quoted[i], i, rowIndex, &errs)
	}

	return errs
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateField dispatches to appropriate validator based on column type
func (v *RowValidator) validateField(fieldValue []byte, isQuoted bool, col headerdef.Column, rowIndex int, errs *[]*errors.ValidationError) {
	switch col.Type.Kind {
	case headerdef.TypeKindScalar:
		v.validateScalarField(fieldValue, isQuoted, col, rowIndex, errs)
	case headerdef.TypeKindList:
		v.validateListField(fieldValue, col, rowIndex, errs)
	case headerdef.TypeKindArray:
		v.validateArrayField(fieldValue, col, rowIndex, errs)
	default:
		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: col.Name,
			Code:   "unknown_type",
			Msg:    fmt.Sprintf("unknown type kind %d", col.Type.Kind),
		})
	}
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateScalarField validates a scalar field (int, float, string, etc.)
func (v *RowValidator) validateScalarField(fieldValue []byte, isQuoted bool, col headerdef.Column, rowIndex int, errs *[]*errors.ValidationError) {
	// Apply type-aware cleanup
	var cleaned []byte
	var err error

	// String and timezone follow the same quoting rules (spec v1.0 section 8.1/8.11)
	if col.Type.Scalar.Kind == headerdef.ScalarString || col.Type.Scalar.Kind == headerdef.ScalarTimezone {
		// String/timezone validation - use quoting info to call correct validator
		if isQuoted {
			cleaned, err = validateQuotedString(fieldValue)
		} else {
			cleaned, err = validateUnquotedString(fieldValue)
		}
	} else {
		// Other scalar validation (int, float, bool, date, etc.)
		// Quoting already checked earlier in ValidateRow
		cleaned, err = validateUnquotedScalar(fieldValue, col.Type.Scalar.Kind)
	}

	if err != nil {
		// Determine appropriate error code based on type and error message
		errCode := errors.CodeInvalidScalar.Code
		if col.Type.Scalar.Kind == headerdef.ScalarString || col.Type.Scalar.Kind == headerdef.ScalarTimezone {
			// String/timezone follow the same quoting rules - both can produce unquoted char errors
			if strings.Contains(err.Error(), "unexpected whitespace character") {
				errCode = errors.CodeUnicodeWhitespaceInUnquoted.Code
			} else if strings.Contains(err.Error(), "invalid character in unquoted") {
				errCode = errors.CodeInvalidUnquotedChar.Code
			}
		}
		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: col.Name,
			Code:   errCode,
			Msg:    err.Error(),
		})
		return
	}

	// Stage 2: primary scalar type validation via pre-bound function pointers.
	// The route is determined at construction time by scalar kind (int/bool/date/...)
	// and executes here without a per-row switch.
	validator := v.validators[col.Index]
	// validateFn is the primary scalar type check for cleaned input.
	if validator.validateFn != nil && !validator.validateFn(cleaned) {
		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: col.Name,
			Code:   validator.errorCode,        // Pre-bound, no function call, no switch
			Msg:    validator.errorFn(cleaned), // Pre-bound, value passed for classifiers (e.g. int)
		})
	}
}

// validateListField validates a list column (list<T> or list<T>[N])
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func (v *RowValidator) validateListField(fieldValue []byte, col headerdef.Column, rowIndex int, errs *[]*errors.ValidationError) {
	// HOT_PATH: Single array access, pre-bound validator, no conditional lookup
	enumValidator := v.validators[col.Index].enumValidator // nil for non-enum, zero-cost

	// Use existing validateListValue which handles container parsing
	err := validateListValue(fieldValue, col.Type.List.Element, col.Type.List.FixedLength, rowIndex, col.Name, enumValidator)
	if err != nil {
		// Extract position info for element errors
		column := col.Name
		isElementErr := false
		if elemErr, ok := err.(*ContainerElementError); ok {
			column = fmt.Sprintf("%s(%d)", col.Name, elemErr.Index+1) // Convert to 1-based
			isElementErr = true
		}

		// Map error messages to specific error codes.
		// Element errors default to invalid_scalar; structural errors default to invalid_container_syntax.
		code := errors.CodeInvalidContainerSyntax.Code
		if isElementErr {
			code = errors.CodeInvalidScalar.Code
		}
		errMsg := err.Error()

		// Structural errors (container syntax)
		if strings.Contains(errMsg, "invalid structure: missing brackets") {
			code = errors.CodeInvalidStructureMissingBrackets.Code
		} else if strings.Contains(errMsg, "invalid structure: quoted string too short") {
			code = errors.CodeInvalidStructureQuotedTooShort.Code
		} else if strings.Contains(errMsg, "invalid structure: missing quotes") {
			code = errors.CodeInvalidStructureMissingQuotes.Code
		} else if strings.Contains(errMsg, "invalid structure: stray quote") {
			code = errors.CodeInvalidStructureStrayQuote.Code
		} else if strings.Contains(errMsg, "invalid structure: unterminated") {
			code = errors.CodeInvalidStructureUnterminatedLiteral.Code
		} else if strings.Contains(errMsg, "empty unquoted string") {
			code = errors.CodeInvalidScalar.Code // Empty string is a scalar validation issue
		} else if strings.Contains(errMsg, "unexpected whitespace character") {
			code = errors.CodeUnicodeWhitespaceInUnquoted.Code
		} else if strings.Contains(errMsg, "forbidden character") {
			code = errors.CodeInvalidUnquotedChar.Code
		} else if strings.Contains(errMsg, "cannot contain nested containers") {
			code = errors.CodeContainerNestingTooDeep.Code
		} else if strings.Contains(errMsg, "invalid bytes<hex>") {
			code = errors.CodeInvalidBytesHex.Code
		} else if strings.Contains(errMsg, "invalid bytes<b64>") {
			code = errors.CodeInvalidBytesB64.Code
		} else if strings.Contains(errMsg, "prefix notation cannot be used with fixed-size arrays") {
			code = errors.CodePrefixOnFixedArray.Code
		} else if strings.Contains(errMsg, "invalid prefix: non-positive") {
			code = errors.CodeInvalidPrefixNonPositiveLength.Code
		} else if strings.Contains(errMsg, "does not match actual length") {
			code = errors.CodePrefixDimensionMismatch.Code
		} else if strings.Contains(errMsg, "trailing comma not allowed") {
			code = errors.CodeTrailingComma.Code
		} else if strings.Contains(errMsg, "expected comma between elements") {
			code = errors.CodeMissingComma.Code
		} else if strings.Contains(errMsg, "consecutive commas not allowed") {
			code = errors.CodeDoubleComma.Code
		}

		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: column,
			Code:   code,
			Msg:    err.Error(),
		})
	}
}

// validateArrayField validates an array column (arr<T>, arr<T>[N], arr<T>[R,C])
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func (v *RowValidator) validateArrayField(fieldValue []byte, col headerdef.Column, rowIndex int, errs *[]*errors.ValidationError) {
	// HOT_PATH: Single array access, pre-bound validator, no conditional lookup
	enumValidator := v.validators[col.Index].enumValidator // nil for non-enum, zero-cost

	// Determine if 2D is preferred based on rank
	prefer2D := col.Type.Array.Rank == headerdef.ArrayRank2DFixed

	// Use existing validateArrayValue which handles array parsing and validation
	err := validateArrayValue(fieldValue, col.Type.Array.Element, col.Type.Array.Length, col.Type.Array.Rows, col.Type.Array.Cols, rowIndex, col.Name, prefer2D, enumValidator)
	if err != nil {
		// Extract position info for element errors
		column := col.Name
		isElementErr := false
		if arrErr, ok := err.(*Array2DElementError); ok {
			// 2D array element error - format as Matrix(row,col) with 1-based indices
			column = fmt.Sprintf("%s(%d,%d)", col.Name, arrErr.Row+1, arrErr.Col+1)
			isElementErr = true
		} else if elemErr, ok := err.(*ContainerElementError); ok {
			// 1D array element error - format as Array(index) with 1-based index
			column = fmt.Sprintf("%s(%d)", col.Name, elemErr.Index+1)
			isElementErr = true
		}

		// Map error messages to specific error codes.
		// Element errors (scalar failures at a known position) default to invalid_scalar;
		// structural errors default to invalid_container_syntax.
		code := errors.CodeInvalidContainerSyntax.Code
		if isElementErr {
			code = errors.CodeInvalidScalar.Code
		}
		errMsg := err.Error()

		// Structural errors (container syntax)
		if strings.Contains(errMsg, "invalid structure: missing brackets") {
			code = errors.CodeInvalidStructureMissingBrackets.Code
		} else if strings.Contains(errMsg, "invalid structure: quoted string too short") {
			code = errors.CodeInvalidStructureQuotedTooShort.Code
		} else if strings.Contains(errMsg, "invalid structure: missing quotes") {
			code = errors.CodeInvalidStructureMissingQuotes.Code
		} else if strings.Contains(errMsg, "invalid structure: stray quote") {
			code = errors.CodeInvalidStructureStrayQuote.Code
		} else if strings.Contains(errMsg, "invalid structure: unterminated array") {
			code = errors.CodeInvalidStructureUnterminatedArray.Code
		} else if strings.Contains(errMsg, "invalid structure: unterminated") {
			code = errors.CodeInvalidStructureUnterminatedLiteral.Code
		} else if strings.Contains(errMsg, "empty unquoted string") {
			code = errors.CodeInvalidScalar.Code
		} else if strings.Contains(errMsg, "unexpected whitespace character") {
			code = errors.CodeUnicodeWhitespaceInUnquoted.Code
		} else if strings.Contains(errMsg, "forbidden character") {
			code = errors.CodeInvalidUnquotedChar.Code
		} else if strings.Contains(errMsg, "may be 1D or 2D only") || strings.Contains(errMsg, "cannot contain nested containers") {
			code = errors.CodeContainerNestingTooDeep.Code
		} else if strings.Contains(errMsg, "invalid bytes<hex>") {
			code = errors.CodeInvalidBytesHex.Code
		} else if strings.Contains(errMsg, "invalid bytes<b64>") {
			code = errors.CodeInvalidBytesB64.Code
		} else if strings.Contains(errMsg, "prefix notation cannot be used with fixed-size arrays") {
			code = errors.CodePrefixOnFixedArray.Code
		} else if strings.Contains(errMsg, "invalid prefix: non-positive") {
			code = errors.CodeInvalidPrefixNonPositiveDimensions.Code
		} else if strings.Contains(errMsg, "does not match actual length") || strings.Contains(errMsg, "expected") && strings.Contains(errMsg, "from prefix") || strings.Contains(errMsg, "unexpected data after prefix rows") {
			code = errors.CodePrefixDimensionMismatch.Code
		} else if strings.Contains(errMsg, "trailing comma not allowed") {
			code = errors.CodeTrailingComma.Code
		} else if strings.Contains(errMsg, "expected comma between elements") {
			code = errors.CodeMissingComma.Code
		} else if strings.Contains(errMsg, "consecutive commas not allowed") {
			code = errors.CodeDoubleComma.Code
		} else if strings.Contains(errMsg, "has") && strings.Contains(errMsg, "columns, expected") {
			code = errors.CodeRowLengthMismatch2D.Code
		} else if strings.Contains(errMsg, "cannot contain empty rows") {
			code = errors.CodeEmptyRowIn2DArray.Code
		}

		*errs = append(*errs, &errors.ValidationError{
			Row:    rowIndex,
			Column: column,
			Code:   code,
			Msg:    err.Error(),
		})
	}
}

// isNullField checks if a field should be treated as null.
// Only "_" is null.
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func (v *RowValidator) isNullField(field []byte) bool {
	// Only "_" is null
	return len(field) == 1 && field[0] == '_'
}
