# SuperCSV v1.0 Quick Reference

A concise reference card for the SuperCSV v1.0 format.
For full details, see [supercsv-spec-v1.0.md](supercsv-spec-v1.0.md) and [supercsv-type-table-v1.0.md](supercsv-type-table-v1.0.md).

---

## File Structure

- **Extension**: `.supr`
- **Encoding**: UTF-8 without BOM
- **Version declaration**: `((SuperCSV v1.0))` — must be line 1 in strict mode
- **Line comments**: `# text` or `( text )`
- **Line metadata syntax**: `(( text ))` — reserved syntax in v1.0; only `((SuperCSV v1.0))` has defined metadata meaning
- **Inline annotations**: `( text )` or `(( text ))` before or after a field value (max 1 per field); non-version metadata syntax has no defined meaning in v1.0
- **Blank lines**: allowed and ignored everywhere
- **Header**: first non-comment, non-blank, non-metadata-syntax line after the version declaration defines the schema
- **Header continuation**: line ending with `,` continues on the next line (header lines only; trailing `,` must be last non-whitespace character)
- **Data rows**: all subsequent non-comment, non-blank, non-line-metadata-syntax lines
- **Structural whitespace**: ASCII space and tab only; leading/trailing trimmed from unquoted scalar fields only (not inside containers, quotes, or annotations)

---

## Identifiers

Used for column names and enum names/values.

- Pattern: `[A-Za-z0-9][A-Za-z0-9_-]*`
- ASCII only; max 255 characters
- `_` alone is reserved (null literal) — invalid as an identifier

## Column Names

- Column names are Identifiers
- Case-preserving; uniqueness checked case-insensitively

---

## Scalar Types

| Type | Description | Notes |
|------|-------------|-------|
| `int` | 64-bit signed integer | No leading zeros, no `-0`, no leading `+`. **Must not be quoted.** |
| `float` | IEEE 754 double | `nan`, `inf`, `-inf` (case-insensitive). **Must not be quoted.** |
| `decimal` | Arbitrary-precision decimal | Max 128 chars, 64 fractional digits. No `-0`, no exponent. **Must not be quoted.** |
| `bool` | `true`, `false`, `1`, `0` | Case-insensitive. **Must not be quoted.** |
| `string` | UTF-8 text | Unquoted (if no special chars) or `"double-quoted"`. `""` = empty string. |
| `bytes<hex>` | Hex-encoded binary | Even-length `[0-9A-Fa-f]+`. No `0x` prefix. **Must not be quoted.** |
| `bytes<b64>` | Base64-encoded binary | RFC 4648 §4 standard alphabet. Padded or unpadded. **Must not be quoted.** |
| `date` | `YYYY-MM-DD` or `YYYY/MM/DD` | Full date validation (month, day, leap year). **Must not be quoted.** |
| `time` | `HH:MM:SS[.fraction]` | 0–9 fractional digits. No timezone. **Must not be quoted.** |
| `datetime` | ISO 8601 datetime | No timezone. Date `T` or space Time. **Must not be quoted.** |
| `datetimetz` | ISO 8601 datetime + tz | Required timezone: `Z` or `±HH:MM`. **Must not be quoted.** |
| `timestamp` | ISO 8601 timestamp | Optional timezone. **Must not be quoted.** |
| `duration` | ISO 8601 duration | Days + time only (no years, months, or weeks). `P[n]DT[n]H[n]M[n]S`. **Must not be quoted.** |
| `timezone` | IANA timezone identifier | Case-sensitive (per IANA). **Only non-string type that may be quoted or unquoted.** |
| `uuid` | Canonical UUID `8-4-4-4-12` | Hex digits + hyphens. **Must not be quoted.** |

**Null**: `_` (unquoted) is valid for all types. `"_"` in quotes is a string, not null.

---

## Container Types

| Syntax | Description |
|--------|-------------|
| `list<T>` | Variable-length list |
| `list<T>[N]` | Fixed-length list (exactly N elements) |
| `arr<T>` | Variable-length 1D or 2D array |
| `arr<T>[N]` | Fixed-length 1D array (N elements) |
| `arr<T>[R,C]` | Fixed-shape 2D array (R rows, C columns) |

- Element type `T` must be a scalar or enum (no nesting)
- Empty container: `[]` (variable-length only)
- 2D arrays must be rectangular (all rows same length)
- Null `_` allowed at container level and element level
- **Must not be quoted**

**Prefix notation** (variable-length only, not for empty or fixed-size):
- 1D: `[N][e1,e2,...eN]`
- 2D: `[R,C][[row1],[row2],...]`

---

## Enum Types

| Syntax | Description |
|--------|-------------|
| `enum<label1,label2,...>` | Name-only enum |
| `enum<0=label1,1=label2,...>` | Value=name enum with codes |

- All items must use the same form (no mixing name-only with value=name)
- Names and values are Identifiers (see [Identifiers](#identifiers)); must not be quoted
- Names must be unique (case-insensitive)
- A value may match its own item's name but must not match any other item's name (case-insensitive)
- Matching is case-insensitive
- `_` cannot be used as an enum name or value in the header definition

---

## Special Values

- `_` — Null literal (unquoted; valid for all types)
- `"_"` — The string `_`, not null
- `""` — Empty string
- `[]` — Empty container (variable-length list/array only)
- Whitespace around commas in containers is ignored and trimmed
