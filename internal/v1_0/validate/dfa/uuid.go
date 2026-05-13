// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateUUID validates UUIDs in 8-4-4-4-12 format
// Pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars total)
// Uses 2-char lookup tables for fast validation (16 lookups vs 32 single-char checks)
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateUUID(value []byte) bool {
	// Must be exactly 36 characters: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(value) != 36 {
		return false
	}

	// Validate dashes and all hex pairs in one expression (no early returns for valid UUIDs)
	return value[8] == '-' && value[13] == '-' && value[18] == '-' && value[23] == '-' &&
		validHexPair[value[0]][value[1]] && validHexPair[value[2]][value[3]] &&
		validHexPair[value[4]][value[5]] && validHexPair[value[6]][value[7]] &&
		validHexPair[value[9]][value[10]] && validHexPair[value[11]][value[12]] &&
		validHexPair[value[14]][value[15]] && validHexPair[value[16]][value[17]] &&
		validHexPair[value[19]][value[20]] && validHexPair[value[21]][value[22]] &&
		validHexPair[value[24]][value[25]] && validHexPair[value[26]][value[27]] &&
		validHexPair[value[28]][value[29]] && validHexPair[value[30]][value[31]] &&
		validHexPair[value[32]][value[33]] && validHexPair[value[34]][value[35]]
}

func UUIDError() string {
	return "invalid UUID format (expected xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)"
}
