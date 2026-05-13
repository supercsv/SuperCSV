// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateBytesB64 validates base64-encoded byte strings
// Format: RFC 4648 standard base64 (padded or unpadded)
// NO URL-safe characters (- and _ not allowed)
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func ValidateBytesB64(value []byte) bool {
	if len(value) == 0 {
		return false
	}

	// len%4==1 is always invalid: 1 base64 char = 6 bits, can't form a complete byte.
	// Safe early reject for both padded and unpadded (RFC 4648).
	if len(value)%4 == 1 {
		return false
	}

	// Scan characters: validate alphabet, track padding position and count.
	padCount := 0
	seenPadding := false

	for _, b := range value {
		// Check if valid base64 character (includes padding '=')
		if !isBase64Char[b] {
			return false
		}

		// Track padding
		if b == '=' {
			seenPadding = true
			padCount++
		} else if seenPadding {
			// Non-padding char after padding = invalid
			return false
		}
	}

	// Valid if multiple of 4, with max 2 padding chars at end
	if len(value)%4 == 0 && padCount <= 2 {
		return true
	}

	// Also valid if unpadded (not multiple of 4, no padding)
	if padCount == 0 {
		return true
	}

	return false
}

func BytesB64Error() string {
	return "invalid bytes<b64> format (expected RFC 4648 base64)"
}
