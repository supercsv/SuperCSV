// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// VersionDirectiveLine is the canonical v1.0 version directive with trailing newline.
const VersionDirectiveLine = "((SuperCSV v1.0))\n"

// HOT_PATH - Sentinel errors (zero allocation)
var (
	ErrTypeMismatch      = errors.New("type mismatch")
	ErrInvalidEnumValue  = errors.New("invalid enum value")
	ErrInvalidArraySize  = errors.New("invalid array size")
	ErrInvalidDimensions = errors.New("invalid array dimensions")
)

// HOT_PATH - Generic value dispatcher using internal types
// Type-driven dispatch - single type assertion per value
func AppendValue(dst []byte, v any, colType headerdef.ColumnType) ([]byte, error) {
	return appendValue(dst, v, colType)
}

func appendValue(dst []byte, v any, colType headerdef.ColumnType) ([]byte, error) {
	if v == nil {
		return append(dst, '_'), nil
	}

	switch colType.Kind {
	case headerdef.TypeKindScalar:
		return appendScalarValue(dst, v, colType.Scalar)
	case headerdef.TypeKindList:
		return appendList(dst, v, colType.List)
	case headerdef.TypeKindArray:
		return appendArray(dst, v, colType.Array)
	default:
		return dst, ErrTypeMismatch
	}
}

// HOT_PATH - Scalar value dispatcher
func appendScalarValue(dst []byte, v any, scalar headerdef.ScalarType) ([]byte, error) {
	// Enum is a special scalar
	if scalar.Kind == headerdef.ScalarEnum {
		return appendEnumValue(dst, v, scalar.Enum)
	}

	switch scalar.Kind {
	case headerdef.ScalarInt:
		if val, ok := v.(int64); ok {
			return AppendInt(dst, val), nil
		}
	case headerdef.ScalarFloat:
		if val, ok := v.(float64); ok {
			return AppendFloat(dst, val), nil
		}
	case headerdef.ScalarBool:
		if val, ok := v.(bool); ok {
			return AppendBool(dst, val), nil
		}
	case headerdef.ScalarString:
		if val, ok := v.(string); ok {
			return appendString(dst, val), nil
		}
	case headerdef.ScalarDecimal:
		if val, ok := v.(string); ok {
			return AppendDecimal(dst, val), nil
		}
	case headerdef.ScalarBytesHex:
		if val, ok := v.(string); ok {
			return AppendBytesHex(dst, val), nil
		}
	case headerdef.ScalarBytesB64:
		if val, ok := v.(string); ok {
			return AppendBytesB64(dst, val), nil
		}
	case headerdef.ScalarDate:
		switch val := v.(type) {
		case headerdef.Date:
			return appendDate(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarTime:
		switch val := v.(type) {
		case headerdef.Time:
			return appendTime(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarTimestamp:
		switch val := v.(type) {
		case headerdef.Timestamp:
			return appendTimestamp(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarDatetime:
		switch val := v.(type) {
		case headerdef.Datetime:
			return appendDatetimeVal(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarDatetimeTZ:
		switch val := v.(type) {
		case headerdef.DatetimeTZ:
			return appendDatetimeTZ(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarDuration:
		switch val := v.(type) {
		case headerdef.Duration:
			return appendDuration(dst, val), nil
		case string:
			return append(dst, val...), nil
		}
	case headerdef.ScalarTimezone:
		if val, ok := v.(string); ok {
			return AppendTimezone(dst, val), nil
		}
	case headerdef.ScalarUUID:
		if val, ok := v.(string); ok {
			return AppendUUID(dst, val), nil
		}
	}
	return dst, ErrTypeMismatch
}

// HOT_PATH - Enum value encoder
func appendEnumValue(dst []byte, v any, spec *headerdef.EnumSpec) ([]byte, error) {
	val, ok := v.(string)
	if !ok {
		return dst, ErrTypeMismatch
	}
	return append(dst, val...), nil
}

// HOT_PATH - List encoder
func appendList(dst []byte, v any, listType headerdef.ListType) ([]byte, error) {
	list, ok := v.([]any)
	if !ok {
		return dst, ErrTypeMismatch
	}

	dst = append(dst, '[')
	for i, elem := range list {
		if i > 0 {
			dst = append(dst, ',')
		}
		elemColType := headerdef.ColumnType{
			Kind:   headerdef.TypeKindScalar,
			Scalar: listType.Element,
		}
		var err error
		dst, err = appendValue(dst, elem, elemColType)
		if err != nil {
			return dst, err
		}
	}
	dst = append(dst, ']')
	return dst, nil
}

// HOT_PATH - Array encoder
func appendArray(dst []byte, v any, arrayType headerdef.ArrayType) ([]byte, error) {
	switch arrayType.Rank {
	case headerdef.ArrayRank1DFixed:
		return appendArray1D(dst, v, arrayType)
	case headerdef.ArrayRank2DFixed:
		return appendArray2D(dst, v, arrayType)
	case headerdef.ArrayRankFlexible:
		return appendArrayDynamic(dst, v, arrayType)
	default:
		return dst, ErrInvalidDimensions
	}
}

// HOT_PATH - Dynamic array encoder (arr<T>)
// Accepts 1D or 2D arrays with runtime dimension detection.
func appendArrayDynamic(dst []byte, v any, arrayType headerdef.ArrayType) ([]byte, error) {
	elemColType := headerdef.ColumnType{
		Kind:   headerdef.TypeKindScalar,
		Scalar: arrayType.Element,
	}

	switch val := v.(type) {
	case []any:
		if len(val) == 0 {
			// Empty array
			return append(dst, '[', ']'), nil
		}
		// Detect 1D vs 2D by checking first element
		if _, is2D := val[0].([]any); is2D {
			// 2D array - validate rectangularity
			expectedCols := len(val[0].([]any))
			dst = append(dst, '[')
			for i, rowAny := range val {
				if i > 0 {
					dst = append(dst, ',')
				}
				row, ok := rowAny.([]any)
				if !ok {
					return dst, ErrTypeMismatch
				}
				if len(row) != expectedCols {
					return dst, ErrInvalidDimensions
				}
				dst = append(dst, '[')
				for j, elem := range row {
					if j > 0 {
						dst = append(dst, ',')
					}
					var err error
					dst, err = appendValue(dst, elem, elemColType)
					if err != nil {
						return dst, err
					}
				}
				dst = append(dst, ']')
			}
			dst = append(dst, ']')
			return dst, nil
		}
		// 1D array
		dst = append(dst, '[')
		for i, elem := range val {
			if i > 0 {
				dst = append(dst, ',')
			}
			var err error
			dst, err = appendValue(dst, elem, elemColType)
			if err != nil {
				return dst, err
			}
		}
		dst = append(dst, ']')
		return dst, nil
	case [][]any:
		// Explicitly 2D - validate rectangularity
		if len(val) > 0 {
			expectedCols := len(val[0])
			for _, row := range val[1:] {
				if len(row) != expectedCols {
					return dst, ErrInvalidDimensions
				}
			}
		}
		dst = append(dst, '[')
		for i, row := range val {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = append(dst, '[')
			for j, elem := range row {
				if j > 0 {
					dst = append(dst, ',')
				}
				var err error
				dst, err = appendValue(dst, elem, elemColType)
				if err != nil {
					return dst, err
				}
			}
			dst = append(dst, ']')
		}
		dst = append(dst, ']')
		return dst, nil
	default:
		return dst, ErrTypeMismatch
	}
}

// HOT_PATH - 1D array encoder
func appendArray1D(dst []byte, v any, arrayType headerdef.ArrayType) ([]byte, error) {
	arr, ok := v.([]any)
	if !ok {
		return dst, ErrTypeMismatch
	}

	if len(arr) != arrayType.Length {
		return dst, ErrInvalidArraySize
	}

	dst = append(dst, '[')
	elemColType := headerdef.ColumnType{
		Kind:   headerdef.TypeKindScalar,
		Scalar: arrayType.Element,
	}
	for i, elem := range arr {
		if i > 0 {
			dst = append(dst, ',')
		}
		var err error
		dst, err = appendValue(dst, elem, elemColType)
		if err != nil {
			return dst, err
		}
	}
	dst = append(dst, ']')
	return dst, nil
}

// HOT_PATH - 2D array encoder
func appendArray2D(dst []byte, v any, arrayType headerdef.ArrayType) ([]byte, error) {
	// Accept [][]any or []any (and type-assert elements)
	var matrix [][]any
	switch val := v.(type) {
	case [][]any:
		matrix = val
	case []any:
		// Convert []any to [][]any by type-asserting each element
		matrix = make([][]any, len(val))
		for i, rowAny := range val {
			rowSlice, ok := rowAny.([]any)
			if !ok {
				return dst, ErrTypeMismatch
			}
			matrix[i] = rowSlice
		}
	default:
		return dst, ErrTypeMismatch
	}

	if len(matrix) != arrayType.Rows {
		return dst, ErrInvalidDimensions
	}

	dst = append(dst, '[')
	elemColType := headerdef.ColumnType{
		Kind:   headerdef.TypeKindScalar,
		Scalar: arrayType.Element,
	}
	for i, row := range matrix {
		if i > 0 {
			dst = append(dst, ',')
		}
		if len(row) != arrayType.Cols {
			return dst, ErrInvalidDimensions
		}
		dst = append(dst, '[')
		for j, elem := range row {
			if j > 0 {
				dst = append(dst, ',')
			}
			var err error
			dst, err = appendValue(dst, elem, elemColType)
			if err != nil {
				return dst, err
			}
		}
		dst = append(dst, ']')
	}
	dst = append(dst, ']')
	return dst, nil
}

// HOT_PATH - Scalar encoders

// AppendInt appends an int64 to dst
func AppendInt(dst []byte, v int64) []byte {
	return strconv.AppendInt(dst, v, 10)
}

// AppendBool appends a bool to dst as 1 or 0
func AppendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, '1')
	}
	return append(dst, '0')
}

// AppendFloat appends a float64 to dst
// Handles special values (nan, inf, -inf) per spec
func AppendFloat(dst []byte, v float64) []byte {
	if math.IsNaN(v) {
		return append(dst, "nan"...)
	}
	if math.IsInf(v, 1) {
		return append(dst, "inf"...)
	}
	if math.IsInf(v, -1) {
		return append(dst, "-inf"...)
	}
	start := len(dst)
	dst = strconv.AppendFloat(dst, v, 'g', -1, 64)
	// Ensure output always contains a decimal point to distinguish from int.
	// Check the appended portion for '.' or 'e'/'E'.
	hasDecimal := false
	for i := start; i < len(dst); i++ {
		if dst[i] == '.' || dst[i] == 'e' || dst[i] == 'E' {
			hasDecimal = true
			break
		}
	}
	if !hasDecimal {
		dst = append(dst, '.', '0')
	}
	return dst
}

// appendString appends a string to dst with quoting if needed
func appendString(dst []byte, s string) []byte {
	if needsQuotingBytes(s) {
		return appendQuotedString(dst, s)
	}
	return append(dst, s...)
}

// needsQuotingBytes returns true if string requires quoting (hot path version)
func needsQuotingBytes(s string) bool {
	// Empty string must be quoted to distinguish from null
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Control chars
		if c < 0x20 || c == 0x7F {
			return true
		}
		// Spec-forbidden unquoted chars: , # [ ] ( ) < > { } " ' ` ; : = ? / \ | @
		switch c {
		case ',', '#', '[', ']', '(', ')', '<', '>', '{', '}',
			'"', '\'', '`', ';', ':', '=', '?', '/', '\\', '|', '@':
			return true
		}
	}
	// Quote if starts or ends with whitespace
	if s[0] == ' ' || s[0] == '\t' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t' {
		return true
	}
	return false
}

// appendQuotedString appends a quoted string to dst, doubling internal quotes
func appendQuotedString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			dst = append(dst, '"', '"') // Double the quote
		} else {
			dst = append(dst, s[i])
		}
	}
	dst = append(dst, '"')
	return dst
}

// AppendDecimal appends a decimal string to dst
// TODO: Implement Decimal.AppendTo(dst) method for zero allocation
func AppendDecimal(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendBytesHex appends a hex-encoded byte string to dst
func AppendBytesHex(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendBytesB64 appends a base64-encoded byte string to dst
func AppendBytesB64(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendDate appends a date string (YYYY-MM-DD) to dst
func AppendDate(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendTime appends a time string (HH:MM:SS[.fff]) to dst
func AppendTime(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendTimestamp appends a timestamp string (RFC3339) to dst
func AppendTimestamp(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendDatetime appends a datetime string (ISO 8601 without timezone) to dst
func AppendDatetime(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendDatetimeTZ appends a datetimetz string (ISO 8601 with required timezone) to dst
func AppendDatetimeTZ(dst []byte, s string) []byte {
	return append(dst, s...)
}

// AppendDuration appends a duration string (ISO 8601) to dst
func AppendDuration(dst []byte, s string) []byte {
	return append(dst, s...)
}

// --- Canonical struct formatters for temporal types ---

// appendDate formats a Date struct as YYYY-MM-DD
func appendDate(dst []byte, d headerdef.Date) []byte {
	dst = appendPaddedInt(dst, d.Year, 4)
	dst = append(dst, '-')
	dst = appendPaddedInt(dst, d.Month, 2)
	dst = append(dst, '-')
	dst = appendPaddedInt(dst, d.Day, 2)
	return dst
}

// appendTime formats a Time struct as HH:MM:SS[.fffffffff]
func appendTime(dst []byte, t headerdef.Time) []byte {
	dst = appendPaddedInt(dst, t.Hour, 2)
	dst = append(dst, ':')
	dst = appendPaddedInt(dst, t.Minute, 2)
	dst = append(dst, ':')
	dst = appendPaddedInt(dst, t.Second, 2)
	if t.Nanos > 0 {
		dst = appendFractionalSeconds(dst, t.Nanos)
	}
	return dst
}

// appendTimestamp formats a Timestamp struct as YYYY-MM-DDTHH:MM:SS[.fff][+/-HH:MM|Z]
func appendTimestamp(dst []byte, ts headerdef.Timestamp) []byte {
	dst = appendDate(dst, headerdef.Date{Year: ts.Year, Month: ts.Month, Day: ts.Day})
	dst = append(dst, 'T')
	dst = appendTime(dst, headerdef.Time{Hour: ts.Hour, Minute: ts.Minute, Second: ts.Second, Nanos: ts.Nanos})
	if ts.OffsetSeconds != 0 {
		dst = appendTZOffset(dst, ts.OffsetSeconds)
	}
	return dst
}

// appendDatetimeVal formats a Datetime struct as YYYY-MM-DDTHH:MM:SS[.fff]
func appendDatetimeVal(dst []byte, dt headerdef.Datetime) []byte {
	dst = appendDate(dst, headerdef.Date{Year: dt.Year, Month: dt.Month, Day: dt.Day})
	dst = append(dst, 'T')
	dst = appendTime(dst, headerdef.Time{Hour: dt.Hour, Minute: dt.Minute, Second: dt.Second, Nanos: dt.Nanos})
	return dst
}

// appendDatetimeTZ formats a DatetimeTZ struct as YYYY-MM-DDTHH:MM:SS[.fff](+/-HH:MM|Z)
func appendDatetimeTZ(dst []byte, dt headerdef.DatetimeTZ) []byte {
	dst = appendDate(dst, headerdef.Date{Year: dt.Year, Month: dt.Month, Day: dt.Day})
	dst = append(dst, 'T')
	dst = appendTime(dst, headerdef.Time{Hour: dt.Hour, Minute: dt.Minute, Second: dt.Second, Nanos: dt.Nanos})
	dst = appendTZOffset(dst, dt.OffsetSeconds)
	return dst
}

// appendDuration formats a Duration struct as PnDTnHnMnS (ISO 8601)
func appendDuration(dst []byte, d headerdef.Duration) []byte {
	dst = append(dst, 'P')
	if d.Days > 0 {
		dst = strconv.AppendInt(dst, int64(d.Days), 10)
		dst = append(dst, 'D')
	}
	if d.Hours > 0 || d.Minutes > 0 || d.Seconds > 0 || d.Nanos > 0 {
		dst = append(dst, 'T')
		if d.Hours > 0 {
			dst = strconv.AppendInt(dst, int64(d.Hours), 10)
			dst = append(dst, 'H')
		}
		if d.Minutes > 0 {
			dst = strconv.AppendInt(dst, int64(d.Minutes), 10)
			dst = append(dst, 'M')
		}
		if d.Seconds > 0 || d.Nanos > 0 {
			dst = strconv.AppendInt(dst, int64(d.Seconds), 10)
			if d.Nanos > 0 {
				dst = appendFractionalSeconds(dst, d.Nanos)
			}
			dst = append(dst, 'S')
		}
	}
	return dst
}

// appendPaddedInt appends an integer zero-padded to width digits
func appendPaddedInt(dst []byte, v int, width int) []byte {
	// For temporal formats, width is always 2 or 4
	if width == 4 {
		dst = append(dst, byte('0'+v/1000%10), byte('0'+v/100%10), byte('0'+v/10%10), byte('0'+v%10))
	} else { // width == 2
		dst = append(dst, byte('0'+v/10%10), byte('0'+v%10))
	}
	return dst
}

// appendFractionalSeconds appends .fffffffff (trimming trailing zeros)
func appendFractionalSeconds(dst []byte, nanos int) []byte {
	dst = append(dst, '.')
	// Format as 9-digit string, then trim trailing zeros
	digits := [9]byte{}
	for i := 8; i >= 0; i-- {
		digits[i] = byte('0' + nanos%10)
		nanos /= 10
	}
	// Find last non-zero digit
	last := 8
	for last > 0 && digits[last] == '0' {
		last--
	}
	dst = append(dst, digits[:last+1]...)
	return dst
}

// appendTZOffset appends a timezone offset as Z, +HH:MM, or -HH:MM
func appendTZOffset(dst []byte, offsetSeconds int) []byte {
	if offsetSeconds == 0 {
		return append(dst, 'Z')
	}
	if offsetSeconds < 0 {
		dst = append(dst, '-')
		offsetSeconds = -offsetSeconds
	} else {
		dst = append(dst, '+')
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	dst = appendPaddedInt(dst, hours, 2)
	dst = append(dst, ':')
	dst = appendPaddedInt(dst, minutes, 2)
	return dst
}

// AppendTimezone appends a timezone string to dst (may need quoting)
func AppendTimezone(dst []byte, s string) []byte {
	return appendString(dst, s)
}

// AppendUUID appends a UUID string to dst
func AppendUUID(dst []byte, s string) []byte {
	return append(dst, s...)
}

// EncodeNull returns the null literal.
func EncodeNull() string {
	return "_"
}

// EncodeInt encodes an int64 value.
func EncodeInt(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	val, ok := v.(int64)
	if !ok {
		return "", fmt.Errorf("expected int64, got %T", v)
	}
	return strconv.FormatInt(val, 10), nil
}

// EncodeFloat encodes a float64 value.
// Special values (Inf, -Inf, NaN) are handled per spec - no leading '+' allowed.
func EncodeFloat(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	val, ok := v.(float64)
	if !ok {
		return "", fmt.Errorf("expected float64, got %T", v)
	}

	// Handle special values per spec (lowercase, no leading '+')
	if math.IsNaN(val) {
		return "nan", nil
	}
	if math.IsInf(val, 1) {
		return "inf", nil // Positive infinity
	}
	if math.IsInf(val, -1) {
		return "-inf", nil
	}

	// Use 'g' format for normal numbers to handle scientific notation.
	// Always include a decimal point to distinguish from int.
	s := strconv.FormatFloat(val, 'g', -1, 64)
	hasDecimal := false
	for i := 0; i < len(s); i++ {
		if s[i] == '.' || s[i] == 'e' || s[i] == 'E' {
			hasDecimal = true
			break
		}
	}
	if !hasDecimal {
		s += ".0"
	}
	return s, nil
}

// EncodeBool encodes a bool value.
// Outputs numeric format: 1 for true, 0 for false.
func EncodeBool(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	val, ok := v.(bool)
	if !ok {
		return "", fmt.Errorf("expected bool, got %T", v)
	}
	if val {
		return "1", nil
	}
	return "0", nil
}

// EncodeString encodes a string value, quoting if necessary.
func EncodeString(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}

	// Check if quoting is needed
	if needsQuoting(str) {
		return quoteString(str), nil
	}
	return str, nil
}

// needsQuoting checks if a string needs to be quoted.
// Per spec: quote if contains any forbidden unquoted character, control characters,
// or has leading/trailing whitespace that must be preserved.
// Forbidden: , # [ ] ( ) < > { } " ' ` ; : = ? / \ | @
func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true // Empty string must be quoted as ""
	}

	// Check for leading/trailing whitespace
	trimmed := strings.TrimSpace(s)
	if len(trimmed) != len(s) {
		return true // Preserve whitespace
	}

	// Check for forbidden characters
	for _, ch := range s {
		switch ch {
		case ',', '#', '[', ']', '(', ')', '<', '>', '{', '}',
			'"', '\'', '`', ';', ':', '=', '?', '/', '\\', '|', '@':
			return true
		}
		// Control characters (tab, newline, etc.)
		if ch < 32 {
			return true
		}
	}

	return false
}

// quoteString quotes a string for CSV output.
func quoteString(s string) string {
	// Double any quotes in the string
	escaped := strings.ReplaceAll(s, `"`, `""`)
	return `"` + escaped + `"`
}

// EncodeDecimal encodes a decimal string value.
// Decimals are stored as strings to preserve precision.
func EncodeDecimal(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (decimal), got %T", v)
	}
	return str, nil
}

// EncodeBytesHex encodes a hex-encoded byte string.
func EncodeBytesHex(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (bytes-hex), got %T", v)
	}
	return str, nil
}

// EncodeBytesB64 encodes a base64-encoded byte string.
func EncodeBytesB64(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (bytes-b64), got %T", v)
	}
	return str, nil
}

// EncodeDate encodes a date string (YYYY-MM-DD).
func EncodeDate(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (date), got %T", v)
	}
	return str, nil
}

// EncodeTime encodes a time string (HH:MM:SS[.fff]).
func EncodeTime(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (time), got %T", v)
	}
	return str, nil
}

// EncodeTimestamp encodes a timestamp string (RFC3339).
func EncodeTimestamp(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (timestamp), got %T", v)
	}
	return str, nil
}

// EncodeDatetime encodes a datetime string (ISO 8601 without timezone).
func EncodeDatetime(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (datetime), got %T", v)
	}
	return str, nil
}

// EncodeDatetimeTZ encodes a datetimetz string (ISO 8601 with required timezone).
func EncodeDatetimeTZ(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (datetimetz), got %T", v)
	}
	return str, nil
}

// EncodeDuration encodes a duration string (ISO 8601).
func EncodeDuration(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (duration), got %T", v)
	}
	return str, nil
}

// EncodeTimezone encodes a timezone string (+/-HH:MM or Z).
func EncodeTimezone(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (timezone), got %T", v)
	}
	// Timezone may be quoted (follows string rules per spec)
	if needsQuoting(str) {
		return quoteString(str), nil
	}
	return str, nil
}

// EncodeUUID encodes a UUID string.
func EncodeUUID(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (uuid), got %T", v)
	}
	return str, nil
}

// EncodeEnum encodes an enum value.
// The value must be a string - either an enum name or value (Identifier).
func EncodeEnum(v any) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}
	val, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string (enum name or value), got %T", v)
	}
	return val, nil
}

// EncodeList encodes a list value.
// elemEncoder is a function that encodes individual elements.
func EncodeList(v any, elemEncoder func(any) (string, error)) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}

	list, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("expected []any (list), got %T", v)
	}

	// Empty list
	if len(list) == 0 {
		return "[]", nil
	}

	// Encode elements
	var sb strings.Builder
	sb.WriteByte('[')
	for i, elem := range list {
		if i > 0 {
			sb.WriteByte(',')
		}
		encoded, err := elemEncoder(elem)
		if err != nil {
			return "", fmt.Errorf("list element %d: %w", i, err)
		}
		sb.WriteString(encoded)
	}
	sb.WriteByte(']')

	return sb.String(), nil
}

// EncodeArray1D encodes a 1D array value.
// elemEncoder is a function that encodes individual elements.
// size is the expected array length (0 for dynamic arrays).
func EncodeArray1D(v any, elemEncoder func(any) (string, error), size int) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}

	arr, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("expected []any (array), got %T", v)
	}

	// Validate size for fixed-size arrays
	if size > 0 && len(arr) != size {
		return "", fmt.Errorf("array size mismatch: expected %d, got %d", size, len(arr))
	}

	// Empty array (only valid for dynamic arrays)
	if len(arr) == 0 {
		if size > 0 {
			return "", fmt.Errorf("empty array invalid for fixed-size array[%d]", size)
		}
		return "[]", nil
	}

	// Encode elements
	var sb strings.Builder
	sb.WriteByte('[')
	for i, elem := range arr {
		if i > 0 {
			sb.WriteByte(',')
		}
		encoded, err := elemEncoder(elem)
		if err != nil {
			return "", fmt.Errorf("array element %d: %w", i, err)
		}
		sb.WriteString(encoded)
	}
	sb.WriteByte(']')

	return sb.String(), nil
}

// EncodeArray2D encodes a 2D array value.
// elemEncoder is a function that encodes individual elements.
// rows and cols are the expected dimensions (0 for dynamic arrays).
func EncodeArray2D(v any, elemEncoder func(any) (string, error), rows, cols int) (string, error) {
	if v == nil {
		return EncodeNull(), nil
	}

	// Accept [][]any or []any (and type-assert elements)
	var arr [][]any
	switch val := v.(type) {
	case [][]any:
		arr = val
	case []any:
		// Convert []any to [][]any by type-asserting each element
		arr = make([][]any, len(val))
		for i, rowAny := range val {
			rowSlice, ok := rowAny.([]any)
			if !ok {
				return "", fmt.Errorf("array row %d: expected []any, got %T", i, rowAny)
			}
			arr[i] = rowSlice
		}
	default:
		return "", fmt.Errorf("expected [][]any or []any (2D array), got %T", v)
	}

	// Validate dimensions for fixed-size arrays
	if rows > 0 && len(arr) != rows {
		return "", fmt.Errorf("array row count mismatch: expected %d, got %d", rows, len(arr))
	}

	// Empty array (only valid for dynamic arrays)
	if len(arr) == 0 {
		if rows > 0 {
			return "", fmt.Errorf("empty array invalid for fixed-size array[%d][%d]", rows, cols)
		}
		return "[]", nil
	}

	// Validate column uniformity and size
	for i, row := range arr {
		if cols > 0 && len(row) != cols {
			return "", fmt.Errorf("array row %d: expected %d columns, got %d", i, cols, len(row))
		}
		if i > 0 && len(row) != len(arr[0]) {
			return "", fmt.Errorf("array row %d: non-rectangular array (expected %d columns, got %d)", i, len(arr[0]), len(row))
		}
	}

	// Encode rows
	var sb strings.Builder
	sb.WriteByte('[')
	for i, row := range arr {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('[')
		for j, elem := range row {
			if j > 0 {
				sb.WriteByte(',')
			}
			encoded, err := elemEncoder(elem)
			if err != nil {
				return "", fmt.Errorf("array element [%d][%d]: %w", i, j, err)
			}
			sb.WriteString(encoded)
		}
		sb.WriteByte(']')
	}
	sb.WriteByte(']')

	return sb.String(), nil
}
