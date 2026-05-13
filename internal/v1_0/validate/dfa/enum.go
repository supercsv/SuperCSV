// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

import (
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// toLowerString returns s with ASCII uppercase letters folded to lowercase.
// Used only at construction time (not hot path).
func toLowerString(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := []byte(s)
			for ; i < len(b); i++ {
				if b[i] >= 'A' && b[i] <= 'Z' {
					b[i] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s // already lowercase - no allocation
}

// EnumValidator validates enum values using map-based O(1) lookup
// Matches against both names and values defined in the enum specification
// Constant ~9ns performance regardless of enum size or access pattern
type EnumValidator struct {
	spec     *headerdef.EnumSpec
	nameMap  map[string]int32
	valueMap map[string]int32 // nil for name-only enums
}

// NewEnumValidator creates an enum validator with map-based lookup tables
func NewEnumValidator(spec *headerdef.EnumSpec) *EnumValidator {
	nameMap := make(map[string]int32, len(spec.Values))
	var valueMap map[string]int32

	for i, val := range spec.Values {
		// Always use declaration index as the returned int32
		idx := int32(i)

		// Add to name map with case-insensitive key (first-wins for declaration order)
		nameKey := toLowerString(val.Name)
		if _, exists := nameMap[nameKey]; !exists {
			nameMap[nameKey] = idx
		}

		// Add to value map if present (first-wins preserves declaration order for duplicate values)
		if val.Value != "" {
			if valueMap == nil {
				valueMap = make(map[string]int32, len(spec.Values))
			}
			valKey := toLowerString(val.Value)
			if _, exists := valueMap[valKey]; !exists {
				valueMap[valKey] = idx
			}
		}
	}

	return &EnumValidator{
		spec:     spec,
		nameMap:  nameMap,
		valueMap: valueMap,
	}
}

// Lookup finds an enum value and returns (declaration index, true) if found, (0, false) if not.
// Checks names first, then values (per spec lookup order). Per spec, no value may match a
// different item's name (cross-item collision forbidden), so there is no ambiguity - the ordering
// is a performance optimisation (names are the common case in data rows). A value matching its
// own item's name is explicitly allowed by spec and does not cause ambiguity here.
// Case-insensitive: input is ASCII-lowered before map lookup.
// HOT_PATH - zero allocations for inputs <=128 bytes (all practical enum identifiers).
func (v *EnumValidator) Lookup(b []byte) (int32, bool) {
	n := len(b)
	if n == 0 {
		return 0, false
	}

	// Stack-allocated buffer for ASCII lowercasing - avoids heap allocation.
	// Go optimizes map[string([]byte)] to not allocate when the slice is stack-local.
	var buf [128]byte
	if n <= len(buf) {
		for i := 0; i < n; i++ {
			c := b[i]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			buf[i] = c
		}
		lower := buf[:n]
		if val, ok := v.nameMap[string(lower)]; ok {
			return val, true
		}
		if v.valueMap != nil {
			if val, ok := v.valueMap[string(lower)]; ok {
				return val, true
			}
		}
		return 0, false
	}

	// Fallback for inputs >128 bytes (cannot be valid enum identifiers, but handle correctly)
	lower := make([]byte, n)
	for i := 0; i < n; i++ {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower[i] = c
	}
	if val, ok := v.nameMap[string(lower)]; ok {
		return val, true
	}
	if v.valueMap != nil {
		if val, ok := v.valueMap[string(lower)]; ok {
			return val, true
		}
	}
	return 0, false
}

// Validate checks if value is a valid enum (label or code)
// HOT_PATH -> zero allocations, O(1) constant time
func (v *EnumValidator) Validate(value []byte) bool {
	_, ok := v.Lookup(value)
	return ok
}

// LookupIndex finds the index of an enum value (by name or value).
// Returns (index, true) if found, (0, false) if not found.
// Index corresponds to position in spec.Values array.
// Comparison is case-insensitive. Zero allocations (delegates to Lookup).
func (v *EnumValidator) LookupIndex(value []byte) (int, bool) {
	idx, ok := v.Lookup(value)
	if !ok {
		return 0, false
	}
	return int(idx), true
}

// Spec returns the underlying EnumSpec
func (v *EnumValidator) Spec() *headerdef.EnumSpec {
	return v.spec
}

func EnumError() string {
	return "invalid enum value (not in allowed names or values)"
}
