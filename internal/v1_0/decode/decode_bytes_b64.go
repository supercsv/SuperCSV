// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import "encoding/base64"

// decodeBytesB64 validates and decodes base64-encoded byte strings
// Format: RFC 4648 standard base64 (padded or unpadded)
// NO URL-safe characters (- and _ not allowed)
// Returns ([]byte, true) on success, (nil, false) on failure
func decodeBytesB64(value []byte) ([]byte, bool) {
	if len(value) == 0 {
		return nil, false
	}

	// Base64 must be multiple of 4 characters (when padded)
	// OR valid unpadded length
	padCount := 0
	seenPadding := false

	for _, b := range value {
		// Check if valid base64 character (includes padding '=')
		if !isBase64Char[b] {
			return nil, false
		}

		// Track padding
		if b == '=' {
			seenPadding = true
			padCount++
		} else if seenPadding {
			// Non-padding char after padding = invalid
			return nil, false
		}
	}

	// Valid if multiple of 4, with max 2 padding chars at end
	isValidPadded := len(value)%4 == 0 && padCount <= 2
	// Also valid if unpadded (not multiple of 4, no padding)
	isValidUnpadded := padCount == 0

	if !isValidPadded && !isValidUnpadded {
		return nil, false
	}

	// Decode using standard library
	// Use RawStdEncoding if unpadded, StdEncoding if padded
	var result []byte
	var err error

	if padCount == 0 && len(value)%4 != 0 {
		// Unpadded
		result, err = base64.RawStdEncoding.DecodeString(string(value))
	} else {
		// Padded or properly sized
		result, err = base64.StdEncoding.DecodeString(string(value))
	}

	if err != nil {
		return nil, false
	}

	return result, true
}
