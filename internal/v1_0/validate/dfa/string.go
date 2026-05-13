// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// ValidateString is a passthrough validator - always accepts
// Strings are validated by the quote parsing layer, not the DFA layer
func ValidateString(value []byte) bool {
	return true
}

func StringError() string {
	return ""
}

// ValidateTimezone validates timezone strings with reasonable length limit
// Max 64 chars (covers longest IANA timezone names like "America/Argentina/ComodRivadavia")
func ValidateTimezone(value []byte) bool {
	// Defensive check: reasonable max length for timezone names
	if len(value) > 64 {
		return false
	}
	return true
}

func TimezoneError() string {
	return ""
}
