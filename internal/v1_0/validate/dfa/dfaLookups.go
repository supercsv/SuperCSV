// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// Shared lookup tables for fast validation across all DFAs

// Digit lookup table for O(1) validation
var isDigit = [256]bool{
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,
}

// 2-char validation lookup tables [firstDigit][secondDigit]bool
// Valid months: 01-12
var validMonth = [256][256]bool{
	'0': {'1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'1': {'0': true, '1': true, '2': true},
}

// Valid days: 01-31
var validDay = [256][256]bool{
	'0': {'1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'1': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'2': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'3': {'0': true, '1': true},
}

// Valid hours: 00-23
var validHour = [256][256]bool{
	'0': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'1': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'2': {'0': true, '1': true, '2': true, '3': true},
}

// Valid minutes/seconds: 00-59
var validMinSec = [256][256]bool{
	'0': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'1': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'2': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'3': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'4': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
	'5': {'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true},
}

// Valid hex pairs for UUID validation (00-FF)
var validHexPair = [256][256]bool{}

func init() {
	// Initialize validHexPair for all combinations of hex digits
	hexDigits := "0123456789abcdefABCDEF"
	for _, first := range hexDigits {
		for _, second := range hexDigits {
			validHexPair[first][second] = true
		}
	}
}

// Date separator lookup (- or /)
var isDateSep = [256]bool{'-': true, '/': true}

// DateTime separator lookup (T or space)
var isDateTimeSep = [256]bool{'T': true, ' ': true}

// Timezone sign lookup (+ or -)
var isTZSign = [256]bool{'+': true, '-': true}

// Hex character lookup (0-9, A-F, a-f)
var isHexChar = [256]bool{
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true,
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true,
}

// Base64 character lookup (A-Z, a-z, 0-9, +, /, =)
var isBase64Char = [256]bool{
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true,
	'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true,
	'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true,
	'Y': true, 'Z': true,
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true,
	'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true,
	'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true,
	'y': true, 'z': true,
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,
	'+': true, '/': true, '=': true,
}
