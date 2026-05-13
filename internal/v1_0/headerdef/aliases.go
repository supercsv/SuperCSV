// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package headerdef

// Type name aliases for canonical, small, and tiny header modes.
// Per spec: aliases are lowercase only and apply to header type declarations.
// Aliases are resolved to ScalarKind enum values before parsing.

// Bidirectional lookup tables for type aliases
var (
	// aliasToScalarKind maps type name aliases to ScalarKind enum.
	// Includes canonical names, small aliases, and tiny aliases.
	// All keys are lowercase.
	aliasToScalarKind = map[string]ScalarKind{
		// Canonical names
		"int":        ScalarInt,
		"float":      ScalarFloat,
		"decimal":    ScalarDecimal,
		"bool":       ScalarBool,
		"string":     ScalarString,
		"bytes<hex>": ScalarBytesHex,
		"bytes<b64>": ScalarBytesB64,
		"date":       ScalarDate,
		"time":       ScalarTime,
		"timestamp":  ScalarTimestamp,
		"datetime":   ScalarDatetime,
		"datetimetz": ScalarDatetimeTZ,
		"duration":   ScalarDuration,
		"timezone":   ScalarTimezone,
		"uuid":       ScalarUUID,

		// Small aliases
		"flt": ScalarFloat,
		"dec": ScalarDecimal,
		"bl":  ScalarBool,
		"str": ScalarString,
		"hex": ScalarBytesHex, // bytes<hex> -> hex in small mode
		"b64": ScalarBytesB64, // bytes<b64> -> b64 in small mode
		"dat": ScalarDate,
		"tm":  ScalarTime,
		"ts":  ScalarTimestamp,
		"dt":  ScalarDatetime,
		"dtz": ScalarDatetimeTZ,
		"dur": ScalarDuration,
		"tz":  ScalarTimezone,
		"uu":  ScalarUUID,

		// Tiny aliases
		"i":  ScalarInt,
		"f":  ScalarFloat,
		"d":  ScalarDecimal,
		"b":  ScalarBool,
		"s":  ScalarString,
		"bx": ScalarBytesHex, // bytes<hex> -> bx in tiny mode
		"b6": ScalarBytesB64, // bytes<b64> -> b6 in tiny mode
		"da": ScalarDate,
		// "dt": already mapped (same as small)
		// "dtz": already mapped (same as small)
		"t": ScalarTime,
		// "ts": already mapped (same as small)
		"du": ScalarDuration,
		"z":  ScalarTimezone,
		"u":  ScalarUUID,
	}

	// scalarKindToCanonical maps ScalarKind to canonical type name strings.
	scalarKindToCanonical = map[ScalarKind]string{
		ScalarInt:        "int",
		ScalarFloat:      "float",
		ScalarDecimal:    "decimal",
		ScalarBool:       "bool",
		ScalarString:     "string",
		ScalarBytesHex:   "bytes<hex>",
		ScalarBytesB64:   "bytes<b64>",
		ScalarDate:       "date",
		ScalarTime:       "time",
		ScalarTimestamp:  "timestamp",
		ScalarDatetime:   "datetime",
		ScalarDatetimeTZ: "datetimetz",
		ScalarDuration:   "duration",
		ScalarTimezone:   "timezone",
		ScalarUUID:       "uuid",
	}

	// scalarKindToSmall maps ScalarKind to small-mode alias strings.
	scalarKindToSmall = map[ScalarKind]string{
		ScalarInt:        "int",
		ScalarFloat:      "flt",
		ScalarDecimal:    "dec",
		ScalarBool:       "bl",
		ScalarString:     "str",
		ScalarBytesHex:   "hex",
		ScalarBytesB64:   "b64",
		ScalarDate:       "dat",
		ScalarTime:       "tm",
		ScalarTimestamp:  "ts",
		ScalarDatetime:   "dt",
		ScalarDatetimeTZ: "dtz",
		ScalarDuration:   "dur",
		ScalarTimezone:   "tz",
		ScalarUUID:       "uu",
	}

	// scalarKindToTiny maps ScalarKind to tiny-mode alias strings.
	scalarKindToTiny = map[ScalarKind]string{
		ScalarInt:        "i",
		ScalarFloat:      "f",
		ScalarDecimal:    "d",
		ScalarBool:       "b",
		ScalarString:     "s",
		ScalarBytesHex:   "bx",
		ScalarBytesB64:   "b6",
		ScalarDate:       "da",
		ScalarTime:       "t",
		ScalarTimestamp:  "ts",
		ScalarDatetime:   "dt",
		ScalarDatetimeTZ: "dtz",
		ScalarDuration:   "du",
		ScalarTimezone:   "z",
		ScalarUUID:       "u",
	}

	// containerKeywordAliasToCanonical maps container keyword aliases to canonical forms.
	// All keys and values are lowercase.
	containerKeywordAliasToCanonical = map[string]string{
		// Canonical keywords
		"list": "list",
		"arr":  "arr",
		"enum": "enum",

		// Small aliases
		"li": "list",
		"ar": "arr",
		"en": "enum",

		// Tiny aliases
		"l": "list",
		"a": "arr",
		"e": "enum",
	}

	// containerKeywordToSmall maps canonical container keywords to small form.
	containerKeywordToSmall = map[string]string{
		"list": "li",
		"arr":  "ar",
		"enum": "en",
	}

	// containerKeywordToTiny maps canonical container keywords to tiny form.
	containerKeywordToTiny = map[string]string{
		"list": "l",
		"arr":  "a",
		"enum": "e",
	}
)

// resolveTypeAlias resolves a type name alias to its ScalarKind.
// Returns the ScalarKind and true if the input is a valid scalar type or alias.
// Returns zero value and false if the input is not a recognized scalar type.
// Input must be lowercase (already validated by parser).
func resolveTypeAlias(typeName string) (kind ScalarKind, ok bool) {
	kind, ok = aliasToScalarKind[typeName]
	return kind, ok
}

// resolveContainerKeywordAlias resolves a container keyword alias to its canonical form.
// Returns the canonical keyword and true if the input is valid.
// Returns empty string and false if the input is not a recognized container keyword.
// Input must be lowercase (already validated by parser).
func resolveContainerKeywordAlias(keyword string) (canonical string, ok bool) {
	canonical, ok = containerKeywordAliasToCanonical[keyword]
	return canonical, ok
}

// FormatScalarCanonical returns the canonical representation of a ScalarKind.
func FormatScalarCanonical(kind ScalarKind) string {
	if name, ok := scalarKindToCanonical[kind]; ok {
		return name
	}
	return "" // Should never happen for valid ScalarKind
}

// FormatScalarSmall returns the small-mode representation of a ScalarKind.
func FormatScalarSmall(kind ScalarKind) string {
	if name, ok := scalarKindToSmall[kind]; ok {
		return name
	}
	return FormatScalarCanonical(kind) // Fallback to canonical
}

// FormatScalarTiny returns the tiny-mode representation of a ScalarKind.
func FormatScalarTiny(kind ScalarKind) string {
	if name, ok := scalarKindToTiny[kind]; ok {
		return name
	}
	return FormatScalarCanonical(kind) // Fallback to canonical
}

// FormatContainerKeywordSmall returns the small-mode representation of a container keyword.
// Input must be a canonical container keyword ("list", "arr", "enum").
func FormatContainerKeywordSmall(canonical string) string {
	if small, ok := containerKeywordToSmall[canonical]; ok {
		return small
	}
	return canonical // Fallback to canonical if not found
}

// FormatContainerKeywordTiny returns the tiny-mode representation of a container keyword.
// Input must be a canonical container keyword ("list", "arr", "enum").
func FormatContainerKeywordTiny(canonical string) string {
	if tiny, ok := containerKeywordToTiny[canonical]; ok {
		return tiny
	}
	return canonical // Fallback to canonical if not found
}
