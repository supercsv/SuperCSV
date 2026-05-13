# SuperCSV Encoder Model

## Scope

This document defines the behavioral model for SuperCSV encoders.
It describes output requirements, type encoding rules, and conformance constraints.
It does **not** redefine the grammar or the type table; those remain canonical in the v1.0 spec.

---

# 1. Output requirements

## 1.1 Version directive

Encoders MUST include a version directive (`((SuperCSV v1.0))`) as the first line of output.
The version MUST match the specification version the encoder implements.

## 1.2 Header

The header row MUST follow the version directive. It declares column names and types in the format `Name:type`.

Header modes are **implementation-defined** and not normative:

- **Canonical** (default): full type names (`int`, `string`, `list<string>`, etc.). This is the only mode guaranteed to round-trip across all implementations.
- **Small**: abbreviated keywords where aliases exist (e.g., `lst` for `list`, `a` for `arr`)
- **Tiny**: maximally abbreviated

All modes produce valid SuperCSV per the spec.

## 1.3 Data rows

Each data row MUST:

- Have exactly as many fields as the header declares columns
- Encode each field according to its declared type
- Use `_` for null values
- Separate fields with commas

## 1.4 File encoding

Output MUST be UTF-8 without BOM.

---

# 2. Type encoding rules

## 2.1 Scalars

| Type | Encoding rule |
|------|---------------|
| `int` | Decimal integer literal, no leading zeros (except `0` itself). |
| `float` | Decimal floating-point or scientific notation (`e`/`E`). Special values: `nan`, `inf`, `-inf`. Output MUST always contain a decimal point or exponent to distinguish from `int` (e.g., `42` → `42.0`). |
| `decimal` | Exact decimal literal as provided. No exponent notation (per type table). |
| `bool` | `true` or `false`. |
| `string` | Unquoted if safe; quoted with `"` if value contains reserved characters (see spec §String Values). |
| `bytes<hex>` | Lowercase hex string. |
| `bytes<b64>` | Standard padded base64 encoding. |
| `date` | `YYYY-MM-DD`. |
| `time` | `HH:MM:SS` or `HH:MM:SS.nnnnnnnnn` (trailing zeros trimmed). |
| `datetime` | `YYYY-MM-DDTHH:MM:SS[.nnn]` — timezone forbidden. |
| `datetimetz` | `YYYY-MM-DDTHH:MM:SS[.nnn]Z` or `YYYY-MM-DDTHH:MM:SS[.nnn]±HH:MM` — timezone required. |
| `timestamp` | `YYYY-MM-DDTHH:MM:SS[.nnn]` with optional timezone (`Z` or `±HH:MM`). |
| `duration` | ISO-8601 duration (`P[nD]T[nH][nM][nS]`). |
| `timezone` | Timezone identifier string. |
| `uuid` | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` (lowercase). |

## 2.2 Strings

A string value MUST be quoted if it contains any character forbidden in unquoted strings (see spec §String Values for the complete list).

The encoder does not perform trimming — values are encoded exactly as provided. If the caller supplies leading/trailing whitespace that must be preserved, the encoder MUST quote the value.

Quoting rules:

- Quoted strings use `"` delimiters
- Internal `"` is doubled: `""`
- Empty string is `""`

## 2.3 Enums

Encoders accept: the enum name as a `string`.

Encoding rules:

- Encoders MUST use **name-form** when the EnumType contains duplicate values (to avoid ambiguity).
- Encoders MAY use **value-form** only when all values are unique within the EnumType.
- Encoders MUST NOT emit numeric forms unless the EnumType explicitly defines numeric values.

## 2.4 Containers

### Lists

- Empty: `[]`
- Non-empty: `[item1,item2,item3]`
- Fixed-size: element count MUST match declared size

### Arrays (1D)

- `[item1,item2,...,itemN]`
- Fixed-size: element count MUST match declared size

### Arrays (2D)

- `[[r1c1,r1c2],[r2c1,r2c2]]`
- All rows MUST have the same length (rectangular)
- Fixed-size: dimensions MUST match declared `[R,C]`
- Whitespace MUST NOT be inserted between brackets, commas, or elements

### Null elements

Null elements inside containers are encoded as `_`.

## 2.5 Null

The null literal is `_`. Never quoted.

---

# 3. Conformance

Encoder output MUST:

- Be valid SuperCSV under standard mode (no permissive extensions required)
- Round-trip through decode → encode without data loss
- Not emit constructs outside the declared version

---

# 4. Pointers to canonical docs

- Grammar: `internal/v1_0/spec/grammar.ebnf`
- Main spec: `internal/v1_0/spec/supercsv-spec-v1.0.md`
- Type table: `internal/v1_0/spec/supercsv-type-table-v1.0.md`
- Validator/decoder model: `internal/v1_0/spec/supercsv-model-v1.0.md`
