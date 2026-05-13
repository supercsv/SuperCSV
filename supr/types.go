// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr

import (
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// Type represents a SuperCSV type (scalar, enum, or container).
// Types are immutable and safe to share across goroutines.
//
// Call Kind() to determine the type category:
//   - KindScalar: call ScalarKind() for the specific scalar type
//   - KindEnum: call EnumSpec() for enum values
//   - KindList/KindArray1D/KindArray2D: call Container() for element type info
type Type interface {
	// Kind returns the top-level type category.
	Kind() Kind

	// ScalarKind returns the scalar type.
	// Only valid when Kind() == KindScalar.
	// Panics if called on enum or container types.
	ScalarKind() ScalarKind

	// EnumSpec returns the enum definition.
	// Only valid when Kind() == KindEnum.
	// Panics if called on scalar or container types.
	EnumSpec() *EnumSpec

	// Container returns the container type information.
	// Only valid when Kind() is KindList, KindArray1D, KindArray2D, or KindArrayDynamic.
	// Panics if called on scalar or enum types.
	Container() ContainerInfo

	// String returns the canonical type string (e.g., "int", "list<string>", "arr<float>[3,4]").
	String() string
}

// Kind represents the top-level type category.
type Kind uint8

const (
	KindScalar       Kind = 1 // Scalar types (int, string, bool, etc.)
	KindEnum         Kind = 2 // Enum types
	KindList         Kind = 3 // list<T>
	KindArray1D      Kind = 4 // array<T>[N]
	KindArray2D      Kind = 5 // array<T>[N][M]
	KindArrayDynamic Kind = 6 // arr<T> (dynamic size, 1D or 2D)
)

// String returns the canonical name of the Kind.
func (k Kind) String() string {
	switch k {
	case KindScalar:
		return "scalar"
	case KindEnum:
		return "enum"
	case KindList:
		return "list"
	case KindArray1D:
		return "array1d"
	case KindArray2D:
		return "array2d"
	case KindArrayDynamic:
		return "array-dynamic"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

// ScalarKind represents one of the 15 scalar types in SuperCSV.
type ScalarKind uint8

const (
	KindString     ScalarKind = 1  // string
	KindInt        ScalarKind = 2  // int (64-bit signed)
	KindFloat      ScalarKind = 3  // float (64-bit IEEE 754)
	KindDecimal    ScalarKind = 4  // decimal (arbitrary precision)
	KindBool       ScalarKind = 5  // bool
	KindBytesHex   ScalarKind = 6  // bytes-hex
	KindBytesB64   ScalarKind = 7  // bytes-b64
	KindDate       ScalarKind = 8  // date (YYYY-MM-DD)
	KindTime       ScalarKind = 9  // time (HH:MM:SS[.fff])
	KindTimestamp  ScalarKind = 10 // timestamp (RFC3339)
	KindDatetime   ScalarKind = 11 // datetime (no timezone)
	KindDatetimeTZ ScalarKind = 12 // datetimetz (timezone required)
	KindDuration   ScalarKind = 13 // duration (ISO 8601)
	KindTimezone   ScalarKind = 14 // timezone (+/-HH:MM or Z)
	KindUUID       ScalarKind = 15 // uuid (RFC 4122)
)

// String returns the canonical scalar type name.
func (sk ScalarKind) String() string {
	switch sk {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindDecimal:
		return "decimal"
	case KindBool:
		return "bool"
	case KindBytesHex:
		return "bytes-hex"
	case KindBytesB64:
		return "bytes-b64"
	case KindDate:
		return "date"
	case KindTime:
		return "time"
	case KindTimestamp:
		return "timestamp"
	case KindDatetime:
		return "datetime"
	case KindDatetimeTZ:
		return "datetimetz"
	case KindDuration:
		return "duration"
	case KindTimezone:
		return "timezone"
	case KindUUID:
		return "uuid"
	default:
		return fmt.Sprintf("ScalarKind(%d)", sk)
	}
}

// ContainerInfo describes the element type and dimensions of a container type.
// Future versions may add NamedFields for record/struct types without breaking this interface.
type ContainerInfo struct {
	// ElementType is the type of elements in the container (scalar, enum, or nested container).
	ElementType Type

	// Rank describes the dimensionality: List, Array1D, or Array2D.
	Rank ArrayRank

	// Dim1 is the size of the first dimension (arrays only, 0 for lists).
	Dim1 int

	// Dim2 is the size of the second dimension (2D arrays only, 0 otherwise).
	Dim2 int
}

// ArrayRank describes the dimensionality of a container.
type ArrayRank uint8

const (
	RankList         ArrayRank = 0 // list<T>
	RankArray1D      ArrayRank = 1 // arr<T>[N]
	RankArray2D      ArrayRank = 2 // arr<T>[R,C]
	RankListFixed    ArrayRank = 3 // list<T>[N]
	RankArrayDynamic ArrayRank = 4 // arr<T>
)

// String returns the canonical rank name.
func (r ArrayRank) String() string {
	switch r {
	case RankList:
		return "list"
	case RankArray1D:
		return "array1d"
	case RankArray2D:
		return "array2d"
	case RankListFixed:
		return "list-fixed"
	case RankArrayDynamic:
		return "arr-dynamic"
	default:
		return fmt.Sprintf("ArrayRank(%d)", r)
	}
}

// EnumSpec defines an enum type with its allowed values.
//
// Enums use one of two uniform styles (mixing is forbidden):
//   - Name-only:  enum<low,medium,high>
//   - Value=Name: enum<0=low,1=medium,2=high>  (value may be any Identifier, not just digits)
//
// When decoding, enum fields are returned as int32 - the 0-based declaration index into
// Values. Use that index to look up the original name or value:
//
//	idx := row[col].(int32)
//	name := spec.Values[idx].Name
//	value := spec.Values[idx].Value
//
// Lookup is case-insensitive and checks names first, then values in declaration order.
// EnumSpec is immutable and safe to share across goroutines.
type EnumSpec struct {
	Values []EnumValue
}

// EnumValue represents one item declared in an enum.
type EnumValue struct {
	Name  string // The enum item name (always present), e.g. "low", "active".
	Value string // The optional value prefix (empty for name-only enums), e.g. "0", "ERR".
}

// EnumField is the decoded type for enum columns in Full and Shallow decode modes.
// Use type assertion: row[col].(supr.EnumField), then call .Name(), .Value(), or .Index().
type EnumField = headerdef.EnumField

// IsNumeric returns true if this enum uses the value=name style (i.e. Value is set).
// Note: values are Identifiers and may be non-numeric (e.g. "ERR", "V1").
func (e *EnumSpec) IsNumeric() bool {
	if len(e.Values) == 0 {
		return false
	}
	return e.Values[0].Value != ""
}

// String returns the canonical enum type string.
func (e *EnumSpec) String() string {
	return "enum" // Simplified for now; full serialization in encoder
}

// File represents a decoded SuperCSV file with header and rows.
type File struct {
	Header *Header
	Rows   [][]interface{} // Each row matches Header.Columns length
}

// --- Scalar Type Constructors ---

var (
	// Singleton scalar types (immutable, safe to reuse)
	String     = newScalarType(KindString)
	Int        = newScalarType(KindInt)
	Float      = newScalarType(KindFloat)
	Decimal    = newScalarType(KindDecimal)
	Bool       = newScalarType(KindBool)
	BytesHex   = newScalarType(KindBytesHex)
	BytesB64   = newScalarType(KindBytesB64)
	Date       = newScalarType(KindDate)
	Time       = newScalarType(KindTime)
	Timestamp  = newScalarType(KindTimestamp)
	Datetime   = newScalarType(KindDatetime)
	DatetimeTZ = newScalarType(KindDatetimeTZ)
	Duration   = newScalarType(KindDuration)
	Timezone   = newScalarType(KindTimezone)
	UUID       = newScalarType(KindUUID)
)

// --- Decoded Value Types ---
//
// When the decoder returns rows in DecodeFull mode (the default),
// temporal and decimal fields are returned as the struct types below.
// Use type assertions to access the decoded fields:
//
//	date := row[col].(supr.DateValue)
//	fmt.Println(date.Year, date.Month, date.Day)
//
// These are type aliases for the internal decode types, so they work
// directly with type assertions on decoded rows.
type (
	DateValue       = headerdef.Date       // Decoded date: Year, Month, Day int
	TimeValue       = headerdef.Time       // Decoded time: Hour, Minute, Second, Nanos int
	TimestampValue  = headerdef.Timestamp  // Decoded timestamp: Year..Nanos, OffsetSeconds int
	DatetimeValue   = headerdef.Datetime   // Decoded datetime (no TZ): Year..Nanos int
	DatetimeTZValue = headerdef.DatetimeTZ // Decoded datetime with TZ: Year..Nanos, OffsetSeconds int
	DurationValue   = headerdef.Duration   // Decoded duration: Days, Hours, Minutes, Seconds, Nanos int
	DecimalValue    = headerdef.Decimal    // Decoded decimal: Value string (literal-preserving)
)

// --- Enum Constructor ---

// Enum creates a new enum type with the specified values.
// Returns a fresh EnumSpec for each call.
//
// Example (name-only):
//
//	enum := supr.Enum([]supr.EnumValue{
//	    {Name: "low"},
//	    {Name: "medium"},
//	    {Name: "high"},
//	})
//
// Example (value=name):
//
//	enum := supr.Enum([]supr.EnumValue{
//	    {Name: "low", Value: "0"},
//	    {Name: "medium", Value: "5"},
//	    {Name: "high", Value: "10"},
//	})
func Enum(values []EnumValue) Type {
	// Create a fresh copy
	copied := make([]EnumValue, len(values))
	copy(copied, values)
	return &enumType{spec: &EnumSpec{Values: copied}}
}

// --- Container Constructors ---

// List creates a list<T> type.
// Returns a fresh Type for each call.
//
// Example:
//
//	listType := supr.List(supr.Int)  // list<int>
func List(elementType Type) Type {
	return &containerType{
		info: ContainerInfo{
			ElementType: elementType,
			Rank:        RankList,
			Dim1:        0,
			Dim2:        0,
		},
	}
}

// ListFixed creates a list<T>[N] type (fixed-size list).
// Returns a fresh Type for each call.
//
// Example:
//
//	listType := supr.ListFixed(supr.String, 3)  // list<string>[3]
func ListFixed(elementType Type, size int) Type {
	return &containerType{
		info: ContainerInfo{
			ElementType: elementType,
			Rank:        RankListFixed,
			Dim1:        size,
			Dim2:        0,
		},
	}
}

// Arr creates an arr<T> type (dynamic-size array).
// Returns a fresh Type for each call.
//
// Example:
//
//	arrType := supr.Arr(supr.Int)  // arr<int>
func Arr(elementType Type) Type {
	return &containerType{
		info: ContainerInfo{
			ElementType: elementType,
			Rank:        RankArrayDynamic,
			Dim1:        0,
			Dim2:        0,
		},
	}
}

// ArrFixed1D creates an arr<T>[N] type.
// Returns a fresh Type for each call.
//
// Example:
//
//	arrType := supr.ArrFixed1D(supr.String, 5)  // arr<string>[5]
func ArrFixed1D(elementType Type, size int) Type {
	return &containerType{
		info: ContainerInfo{
			ElementType: elementType,
			Rank:        RankArray1D,
			Dim1:        size,
			Dim2:        0,
		},
	}
}

// ArrFixed2D creates an arr<T>[R,C] type.
// Returns a fresh Type for each call.
//
// Example:
//
//	arrType := supr.ArrFixed2D(supr.Float, 3, 4)  // arr<float>[3,4]
func ArrFixed2D(elementType Type, rows, cols int) Type {
	return &containerType{
		info: ContainerInfo{
			ElementType: elementType,
			Rank:        RankArray2D,
			Dim1:        rows,
			Dim2:        cols,
		},
	}
}

// --- Internal Type Implementations ---

// scalarType implements Type for scalar types.
type scalarType struct {
	kind ScalarKind
}

func newScalarType(kind ScalarKind) Type {
	return &scalarType{kind: kind}
}

func (t *scalarType) Kind() Kind {
	return KindScalar
}

func (t *scalarType) ScalarKind() ScalarKind {
	return t.kind
}

func (t *scalarType) EnumSpec() *EnumSpec {
	panic("EnumSpec() called on scalar type")
}

func (t *scalarType) Container() ContainerInfo {
	panic("Container() called on scalar type")
}

func (t *scalarType) String() string {
	return t.kind.String()
}

// enumType implements Type for enum types.
type enumType struct {
	spec *EnumSpec
}

func (t *enumType) Kind() Kind {
	return KindEnum
}

func (t *enumType) ScalarKind() ScalarKind {
	panic("ScalarKind() called on enum type")
}

func (t *enumType) EnumSpec() *EnumSpec {
	return t.spec
}

func (t *enumType) Container() ContainerInfo {
	panic("Container() called on enum type")
}

func (t *enumType) String() string {
	return t.spec.String()
}

// containerType implements Type for container types.
type containerType struct {
	info ContainerInfo
}

// Kind returns the broad category of this container type.
// Per spec, list<T> and list<T>[N] are the same container type with an optional
// size constraint - both return KindList. Use Container().Rank to distinguish
// RankList (dynamic) from RankListFixed (fixed-size).
func (t *containerType) Kind() Kind {
	switch t.info.Rank {
	case RankList:
		return KindList
	case RankListFixed:
		return KindList // Same type as list - size constraint only. Use Container().Rank to distinguish.
	case RankArrayDynamic:
		return KindArrayDynamic
	case RankArray1D:
		return KindArray1D
	case RankArray2D:
		return KindArray2D
	default:
		panic(fmt.Sprintf("invalid ArrayRank: %d", t.info.Rank))
	}
}

func (t *containerType) ScalarKind() ScalarKind {
	panic("ScalarKind() called on container type")
}

func (t *containerType) EnumSpec() *EnumSpec {
	panic("EnumSpec() called on container type")
}

func (t *containerType) Container() ContainerInfo {
	return t.info
}

func (t *containerType) String() string {
	switch t.info.Rank {
	case RankList:
		return fmt.Sprintf("list<%s>", t.info.ElementType.String())
	case RankListFixed:
		return fmt.Sprintf("list<%s>[%d]", t.info.ElementType.String(), t.info.Dim1)
	case RankArrayDynamic:
		return fmt.Sprintf("arr<%s>", t.info.ElementType.String())
	case RankArray1D:
		return fmt.Sprintf("arr<%s>[%d]", t.info.ElementType.String(), t.info.Dim1)
	case RankArray2D:
		return fmt.Sprintf("arr<%s>[%d,%d]", t.info.ElementType.String(), t.info.Dim1, t.info.Dim2)
	default:
		panic(fmt.Sprintf("invalid ArrayRank: %d", t.info.Rank))
	}
}
