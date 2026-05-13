// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
	"github.com/supercsv/supercsv/internal/v1_0/validate/dfa"
)

// decodeScalarValue dispatches to the correct scalar decoder based on type.
// This is the glue between Phase 1 (scalars) and Phase 2 (containers).
// For enums, returns headerdef.EnumField (a pointer wrapper into the column's EnumSpec.Values slice).
func decodeScalarValue(b []byte, st headerdef.ScalarType, ev *dfa.EnumValidator) (any, error) {
	// Handle null literal '_' (SuperCSV v1.0 feature)
	// Must be checked before type-specific decoding
	if isNullElement(b) {
		return nil, nil
	}

	switch st.Kind {
	case headerdef.ScalarInt:
		val, ok := decodeInt(b)
		if !ok {
			return nil, fmt.Errorf("invalid int")
		}
		return val, nil

	case headerdef.ScalarFloat:
		val, ok := decodeFloat(b)
		if !ok {
			return nil, fmt.Errorf("invalid float")
		}
		return val, nil

	case headerdef.ScalarBool:
		val, ok := decodeBool(b)
		if !ok {
			return nil, fmt.Errorf("invalid bool")
		}
		return val, nil

	case headerdef.ScalarDecimal:
		val, ok := decodeDecimal(b)
		if !ok {
			return nil, fmt.Errorf("invalid decimal")
		}
		return val, nil

	case headerdef.ScalarString:
		return decodeString(b), nil

	case headerdef.ScalarBytesHex:
		val, ok := decodeBytesHex(b)
		if !ok {
			return nil, fmt.Errorf("invalid bytes:hex")
		}
		return val, nil

	case headerdef.ScalarBytesB64:
		val, ok := decodeBytesB64(b)
		if !ok {
			return nil, fmt.Errorf("invalid bytes:b64")
		}
		return val, nil

	case headerdef.ScalarDate:
		val, ok := decodeDate(b)
		if !ok {
			return nil, fmt.Errorf("invalid date")
		}
		return val, nil

	case headerdef.ScalarTime:
		val, ok := decodeTime(b)
		if !ok {
			return nil, fmt.Errorf("invalid time")
		}
		return val, nil

	case headerdef.ScalarTimestamp:
		val, ok := decodeTimestamp(b)
		if !ok {
			return nil, fmt.Errorf("invalid timestamp")
		}
		return val, nil

	case headerdef.ScalarDatetime:
		val, ok := decodeDatetime(b)
		if !ok {
			return nil, fmt.Errorf("invalid datetime")
		}
		return val, nil

	case headerdef.ScalarDatetimeTZ:
		val, ok := decodeDatetimeTZ(b)
		if !ok {
			return nil, fmt.Errorf("invalid datetimetz")
		}
		return val, nil

	case headerdef.ScalarDuration:
		val, ok := decodeDuration(b)
		if !ok {
			return nil, fmt.Errorf("invalid duration")
		}
		return val, nil

	case headerdef.ScalarUUID:
		_, ok := decodeUUID(b)
		if !ok {
			return nil, fmt.Errorf("invalid uuid")
		}
		// Return as string (the original validated bytes)
		return string(b), nil

	case headerdef.ScalarTimezone:
		// Timezone is just a string identifier (no parsing)
		return decodeString(b), nil

	case headerdef.ScalarEnum:
		// Enum decoding: returns EnumField - a thin pointer wrapper into the column's
		// EnumSpec.Values slice. Zero allocation. Caller asserts .(supr.EnumField).
		if ev == nil {
			return nil, fmt.Errorf("internal error: enum validator not provided")
		}

		idx, ok := ev.LookupIndex(b)
		if !ok {
			return nil, fmt.Errorf("invalid enum value")
		}

		return headerdef.NewEnumField(&ev.Spec().Values[idx]), nil

	default:
		return nil, fmt.Errorf("unknown scalar type: %v", st.Kind)
	}
}
