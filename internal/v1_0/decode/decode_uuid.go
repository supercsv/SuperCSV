// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeUUID validates and decodes UUIDs in 8-4-4-4-12 format
// Pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars total)
// Returns ([16]byte, true) on success, ([16]byte{}, false) on failure
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func decodeUUID(value []byte) ([16]byte, bool) {
	// Must be exactly 36 characters: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(value) != 36 {
		return [16]byte{}, false
	}

	// Validate dashes and all hex pairs
	if !(value[8] == '-' && value[13] == '-' && value[18] == '-' && value[23] == '-' &&
		validHexPair[value[0]][value[1]] && validHexPair[value[2]][value[3]] &&
		validHexPair[value[4]][value[5]] && validHexPair[value[6]][value[7]] &&
		validHexPair[value[9]][value[10]] && validHexPair[value[11]][value[12]] &&
		validHexPair[value[14]][value[15]] && validHexPair[value[16]][value[17]] &&
		validHexPair[value[19]][value[20]] && validHexPair[value[21]][value[22]] &&
		validHexPair[value[24]][value[25]] && validHexPair[value[26]][value[27]] &&
		validHexPair[value[28]][value[29]] && validHexPair[value[30]][value[31]] &&
		validHexPair[value[32]][value[33]] && validHexPair[value[34]][value[35]]) {
		return [16]byte{}, false
	}

	// Parse UUID into [16]byte
	var result [16]byte
	positions := []int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}

	for i, pos := range positions {
		high := hexCharToNibble(value[pos])
		low := hexCharToNibble(value[pos+1])
		result[i] = (high << 4) | low
	}

	return result, true
}

// hexCharToNibble converts a hex character to its 4-bit value
func hexCharToNibble(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 0 // Should never happen if validation passed
}
