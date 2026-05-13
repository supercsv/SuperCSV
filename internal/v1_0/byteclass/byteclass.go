// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package byteclass

// Character classification flags for the shared combined lookup table.
// The first extraction pass mirrors the current validate/decode behavior exactly.
const (
	CharWhitespace    uint8 = 1 << 0 // 0x01 - space, tab, \n, \r
	CharForbidden     uint8 = 1 << 1 // 0x02 - , # " [ ] ( ) < > { } ' ` ; : = ? / \ | @ \n \r
	CharContinuation  uint8 = 1 << 2 // 0x04 - UTF-8 continuation bytes 0x80-0xBF
	CharUnicodeWSLead uint8 = 1 << 3 // 0x08 - lead bytes of edge-invalid unicode whitespace/invisible chars not caught by ASCII trim
)

// Shared lookup tables. These are initialized once and are intended to replace
// the duplicated validate/decode tables in a later mechanical switch-over.
var (
	CharClass           [256]uint8
	IsWhitespace        [256]bool
	IsForbiddenInString [256]bool
)

// IsUnicodeEdgeWS reports whether the bytes starting at b[0] encode an
// edge-invalid unicode whitespace or invisible format codepoint that must be
// quoted when it appears at an unquoted string edge.
//
// The retained name is compatibility-oriented: the helper now covers the
// existing unicode whitespace set plus the approved invisible edge-invalid
// characters U+200B, U+200C, U+200D, U+2060, and U+FEFF.
func IsUnicodeEdgeWS(b []byte) bool {
	switch b[0] {
	case 0x0B, 0x0C: // \v, \f - single-byte, always whitespace
		return true
	case 0xC2: // U+0085 NEL (C2 85), U+00A0 NBSP (C2 A0)
		return len(b) >= 2 && (b[1] == 0x85 || b[1] == 0xA0)
	case 0xE1: // U+1680 OGHAM SPACE MARK (E1 9A 80)
		return len(b) >= 3 && b[1] == 0x9A && b[2] == 0x80
	case 0xE2:
		if len(b) < 3 {
			return false
		}
		switch b[1] {
		case 0x80: // U+2000-U+200D, U+2028, U+2029, U+202F
			return b[2] <= 0x8D || b[2] == 0xA8 || b[2] == 0xA9 || b[2] == 0xAF
		case 0x81: // U+205F (E2 81 9F), U+2060 (E2 81 A0)
			return b[2] == 0x9F || b[2] == 0xA0
		}
		return false
	case 0xE3: // U+3000 IDEOGRAPHIC SPACE (E3 80 80)
		return len(b) >= 3 && b[1] == 0x80 && b[2] == 0x80
	case 0xEF: // U+FEFF ZERO WIDTH NO-BREAK SPACE / BOM (EF BB BF)
		return len(b) >= 3 && b[1] == 0xBB && b[2] == 0xBF
	}
	return false
}

// TrimASCIISpace trims ASCII whitespace (space, tab, \n, \r) from both ends.
func TrimASCIISpace(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	start := 0
	for start < len(data) && CharClass[data[start]]&CharWhitespace != 0 {
		start++
	}

	end := len(data)
	for end > start && CharClass[data[end-1]]&CharWhitespace != 0 {
		end--
	}

	return data[start:end]
}

func init() {
	CharClass[' '] |= CharWhitespace
	CharClass['\t'] |= CharWhitespace
	CharClass['\n'] |= CharWhitespace | CharForbidden
	CharClass['\r'] |= CharWhitespace | CharForbidden

	CharClass[','] |= CharForbidden
	CharClass['#'] |= CharForbidden
	CharClass['"'] |= CharForbidden
	CharClass['['] |= CharForbidden
	CharClass[']'] |= CharForbidden
	CharClass['('] |= CharForbidden
	CharClass[')'] |= CharForbidden
	CharClass['<'] |= CharForbidden
	CharClass['>'] |= CharForbidden
	CharClass['{'] |= CharForbidden
	CharClass['}'] |= CharForbidden
	CharClass['\''] |= CharForbidden
	CharClass['`'] |= CharForbidden
	CharClass[';'] |= CharForbidden
	CharClass[':'] |= CharForbidden
	CharClass['='] |= CharForbidden
	CharClass['?'] |= CharForbidden
	CharClass['/'] |= CharForbidden
	CharClass['\\'] |= CharForbidden
	CharClass['|'] |= CharForbidden
	CharClass['@'] |= CharForbidden

	for b := 0x80; b <= 0xBF; b++ {
		CharClass[b] |= CharContinuation
	}

	CharClass[0x0B] |= CharUnicodeWSLead // \v
	CharClass[0x0C] |= CharUnicodeWSLead // \f
	CharClass[0xC2] |= CharUnicodeWSLead // U+0085, U+00A0
	CharClass[0xE1] |= CharUnicodeWSLead // U+1680
	CharClass[0xE2] |= CharUnicodeWSLead // U+2000-U+2060
	CharClass[0xE3] |= CharUnicodeWSLead // U+3000
	CharClass[0xEF] |= CharUnicodeWSLead // U+FEFF

	IsWhitespace[' '] = true
	IsWhitespace['\t'] = true
	IsWhitespace['\n'] = true
	IsWhitespace['\r'] = true

	IsForbiddenInString[','] = true
	IsForbiddenInString['#'] = true
	IsForbiddenInString['"'] = true
	IsForbiddenInString['['] = true
	IsForbiddenInString[']'] = true
	IsForbiddenInString['('] = true
	IsForbiddenInString[')'] = true
	IsForbiddenInString['<'] = true
	IsForbiddenInString['>'] = true
	IsForbiddenInString['{'] = true
	IsForbiddenInString['}'] = true
	IsForbiddenInString['\''] = true
	IsForbiddenInString['`'] = true
	IsForbiddenInString[';'] = true
	IsForbiddenInString[':'] = true
	IsForbiddenInString['='] = true
	IsForbiddenInString['?'] = true
	IsForbiddenInString['/'] = true
	IsForbiddenInString['\\'] = true
	IsForbiddenInString['|'] = true
	IsForbiddenInString['@'] = true
	IsForbiddenInString['\n'] = true
	IsForbiddenInString['\r'] = true
}
