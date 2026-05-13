# **SuperCSV v1.0 — Type Table**

A minimal, deterministic, human‑friendly type system for SuperCSV v1.0.  
All types listed here may appear in header declarations.
Their corresponding values may appear in data rows.

---

## Overview

- Defines all built-in types available in SuperCSV v1.0 header declarations
- Each type entry specifies its accepted value format, constraints, and null literal behaviour
- Three type categories: scalar types, container types (`list`, `arr`), and the inline enum type
- Global Rules (null literal, quoting) apply to all types and are stated once at the top
- Type aliases provide short and tiny alternate names for all types; aliases are normative

---

## Global Rules

Rules that apply to all types without exception.

### Null Literal

- **`_`** — represents missing or undefined data
- Valid in all value positions for all types
- Never quoted
- `_` inside quotes is a string value, not null

### Quoting Constraints

- Numeric, boolean, enum, container, and binary values **must not be quoted**
- String values may be quoted or unquoted (see [string](#string) rules)
- Quoted values carry a leading `"` after whitespace trimming

Comment and metadata syntax are defined by the main v1.0 spec. In this type table, only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning; any other metadata syntax has no defined effect on type interpretation.

---

## Type System Overview

### ScalarType  
Includes:

- **numeric types**: `int`, `float`, `decimal`
- **boolean**: `bool`
- **string**: `string` (scalar, with its own quoting and syntax rules)
- **binary**: `bytes`
- **time types**: `date`, `time`, `datetime`, `datetimetz`, `timestamp`, `duration`, `timezone`
- **UUID**: `uuid`

---

### EnumType  
Defined inline using `enum<...>`.  
Its EnumItems define the complete set of allowed EnumValues.

- **EnumType**: `enum<low,medium,high>`, `enum<0=red,1=green,2=blue>`

---

### ContainerType  
Parameterized collection types:

- **list<T>** — ordered, dynamic or fixed size via list<T>[N]
- **arr<T>** — dynamic or fixed size  

Where **T** is a **ScalarType** or **EnumType**

Containers cannot nest.

---

## Scalar Types

### Numeric

#### `int`  
64‑bit signed integer (int64).
Range: `−9223372036854775808` to `9223372036854775807`.
Optional leading minus sign.  
No leading plus sign.  
No leading zeros except for zero itself.
Negative zero (`-0`) is not allowed — use `0` instead.
Digits 0–9 only.  
**Values must not be quoted.**  
Null literal `_` allowed.

Examples (valid):
```
42
-7
0
_
```

Examples (invalid):
```
"42"
+7
007
```

---

#### `float`  
IEEE‑754 binary64 (double precision) floating‑point number.
Accepts decimal literals, scientific notation, and special values (case‑insensitive).

**Format:**
- Optional leading minus sign  
- Decimal notation with optional decimal point  
- Scientific notation (`e`/`E` with optional sign)  
- Special values: `nan`, `inf`, `-inf`  

**Special values (case‑insensitive):**
- NaN: `nan`
- Positive infinity: `inf`
- Negative infinity: `-inf`

**Rules:**
- `nan`, `inf`, and `-inf` are the only canonical special values  
- Leading `+` is not permitted  
- **Values must not be quoted**  
- No thousand separators  
- Null literal `_` allowed  

Examples (valid):
```
3.14
1e6
-inf
nan
INF
_
```

Examples (invalid):
```
"3.14"
1,000
+2.5
+inf
infinity
```

---

#### `decimal`  
Exacr arbitrary‑precision literal‑preserving decimal type.
Values are stored and transmitted exactly as written — no conversion to binary floating‑point occurs.

**Format:**
- Optional leading minus sign  
- Integer part must be either `0` or a non‑zero digit followed by digits
- Optional fractional part: a decimal point followed by one or more digits
- Negative zero (`-0`) is not allowed — use `0` instead

**Rules:**
- Trailing zeros in the fractional part are allowed  
- No exponent notation  
- No special values  
- No thousand separators  
- Maximum total length: 128 characters  
- Maximum fractional digits: 64  
- **Values must not be quoted**  
- Null literal `_` allowed

Examples (valid):
```
12.345
-0.0001
42.0
_
```

Examples (invalid):
```
1e6
"12.3"
12.
```

---

### Boolean

#### `bool`  
Accepted values (case‑insensitive for textual):

- Textual: `true`, `false`
- Numeric: `1`, `0`
- Null: `_`

Only these forms are valid.  
All other textual or numeric representations (e.g. `yes`, `no`, `t`, `f`, `2`, `01`) are invalid.

**Values must not be quoted.**

---

### Text

#### `string`
UTF‑8 text.

Unquoted strings are permitted only if, after trimming leading and trailing
whitespace, the resulting value contains none of:

- `,`
- `#`
- `[` or `]`
- `(` or `)`
- `<` or `>`
- `{` or `}`
- `"`
- `'`
- `` ` `` (backtick)
- `;`
- `:`
- `=`
- `?`
- `/` or `\`
- `|`
- `@`
- control characters (tabs, newlines)

Unquoted strings may contain internal spaces. Any leading or trailing
whitespace in an unquoted string is ALWAYS trimmed. If leading or trailing
whitespace must be preserved, a quoted string must be used.

An unquoted string must not begin or end with any invisible or visually
space-like character from the canonical edge-invalid set defined in the main spec's whitespace rules. These characters are allowed inside quoted strings and are allowed when internal to an unquoted string, but they must be quoted if they appear at either edge.

If any disallowed character appears, the value must be quoted.

The null value is represented by `_` as an unquoted field.

Quoted strings use the standard double‑quote form:

- "value"
- doubled quotes represent a literal quote: ""
- may contain any UTF‑8
- empty string is ""
- "_" inside quotes is a string, not null

**Rules:**
- Strings may be quoted or unquoted
- Quoting required if disallowed characters appear
- Leading/trailing whitespace preserved only inside quotes
- Null literal `_` allowed

WARNING: Unquoted strings must never contain `(` or `)`. Writing e.g. `Alice (née Smith)` will cause `(née Smith)` to be parsed as comment syntax rather than string content, so the intended literal is not preserved. Quote it instead as `"Alice (née Smith)"`.

Examples (unquoted):
```
Bob
alpha-2
v1.0.3
Bob_the_Builder
_
```

Examples (quoted):
```
"Bob, the Builder"
" #hash "
"She said ""hi"""
"Alice (née Smith)"       # correct form when value contains parens
```

Examples (invalid):
```
Bob, the Builder
foo(bar               # ( not followed by valid comment block — parse error
foo)bar               # ) with no preceding ( — parse error
"unterminated
```

---

### Binary

#### `bytes<…>` — Scalar Byte‑Sequence Types

Scalar byte‑sequences with explicit encoding.

- These are **scalar types**: conceptually containers of sequences of bytes, but treated as single atomic values.  
- The `<encoding>` parameter defines the representation, not the structure.  
- This does **not** violate the "no nested containers" rule:  
  - `list<bytes<hex>>` and `arr<bytes<b64>>` are valid because the inner type is a **scalar**, not a container.  
- SuperCSV defines **no bare `bytes` type** to avoid ambiguity.  
- No auto‑detection between hex and Base64 is permitted.

#### `bytes<hex>`  
A hexadecimal value.

##### Rules
- Literal form is a sequence of hex digits: **`[0-9A-Fa-f]+`**  
- Case‑insensitive  
- Length **MUST** be even (each pair = one byte)  
- No prefixes (`0x`, `\x`, etc.)  
- No separators, no whitespace  
- **Values must not be quoted**  
- Null literal `_` allowed  
- Canonical output SHOULD use lowercase hex

##### Examples (valid)
```
00
deadbeef
CAFEBABE
0123456789abcdef
_
```

##### Examples (invalid)
```
0xDEADBEEF      # prefix not allowed
abc             # odd length
"deadbeef"      # quoted
ghij            # invalid hex digits
```

---

#### `bytes<b64>`  
A base64 value.

##### Rules
- Literal form MUST be valid base64 (RFC 4648)  
- Padded or unpadded forms allowed  
- URL‑safe base64 (`-` and `_`) is **not** allowed in v1; the permitted alphabet is the standard Base 64 alphabet defined in RFC 4648 §4 — not the URL and filename-safe alphabet defined in RFC 4648 §5  
- No whitespace  
- **Values must not be quoted**  
- Null literal `_` allowed  
- Canonical output SHOULD use padded base64

##### Examples (valid)
```
aGVsbG8=
YWJjZGVm
AQIDBAUGBwgJ
_
```

##### Examples (invalid)
```
hello world!    # not base64
abc===          # invalid padding
"YWJjZGVm"      # quoted
a-b_c           # URL-safe form not allowed in SuperCSV
```

---

### Time & Date

#### `date`  
- **ISO‑8601 calendar date with full calendar validation.**  
- **Accepts `YYYY-MM-DD` or `YYYY/MM/DD`.**  
- Validates:  
  - year = 4 digits  
  - month = 01–12  
  - day = correct for the given month  
  - leap‑year rules for February  
- Rejects impossible dates (e.g. `2025-02-30`, `2023-11-31`, non‑leap `2023-02-29`).  
- **Values must not be quoted.**  
- **Null literal `_` allowed.**

Examples (valid):
```
2025-01-05
1999-12-31
2024-02-29      # leap year
2025/01/05
1999/12/31
_
```

Examples (invalid):
```
2025-02-30      # February has 28 or 29 days
2023-11-31      # November has 30 days
2023-02-29      # not a leap year
"2025-01-05"    # quoted
05/01/2025      # wrong format
2025.01.05      # wrong separators
```

---

#### `time`  
ISO‑8601 local time.  
Fractional seconds allowed (0–9 digits).  
No timezone.  
**Values must not be quoted.**  
Null literal `_` allowed.

Examples (valid):
```
14:30:00
23:59:59.123
08:15:42.987654321
_
```

Examples (invalid):
```
"14:30:00"
14:30
```

---

#### `datetime`  
ISO‑8601 datetime.  
Date may use `-` or `/`.  
Separator may be space or `T`.  
Fractional seconds allowed.  
**Values must not be quoted.**  
Null literal `_` allowed.

Syntax:
YYYY-MM-DD[ T | space ]HH:MM:SS[.fraction]


Examples (valid):
```
2025-01-05 14:30:00
2025-01-05T14:30:00
2025/01/05 14:30:00.123
_
```

Examples (invalid):
```
"2025-01-05T14:30:00"
2025-01-05T14:30Z
2025-01-05T14:30:00+13:00:00
```

---

#### `datetimetz`  
ISO‑8601 datetime with required timezone.  
Date may use `-` or `/`.  
Separator may be space or `T`.  
Fractional seconds allowed.  
Timezone required (`Z` or `±HH:MM`, no seconds).  
**Values must not be quoted.**  
Null literal `_` allowed.

Syntax:
YYYY-MM-DD[ T | space ]HH:MM:SS[.fraction](Z | ±HH:MM)


Examples (valid):
```

2025-01-05T14:30:00Z
2025/01/05 14:30:00.123Z
2025/01/05 14:30:00.123-05:00
2025-01-05T14:30:00+13:00
_
```

Examples (invalid):
```
"2025-01-05T14:30:00"
2025-01-05 14:30:00
2025-01-05T14:30
2025-01-05T14:30:00+13:00:00
```

---

#### `timestamp`  
ISO‑8601 timestamp with optional timezone.  
Date may use `-` or `/`.  
Separator may be space or `T`.  
Fractional seconds allowed.  
Timezone optional (`Z` or `±HH:MM`, no seconds).  
**Values must not be quoted.**  
Null literal `_` allowed.

Syntax:
YYYY-MM-DD[ T | space ]HH:MM:SS[.fraction][(Z | ±HH:MM)]

Examples (valid):
```
2025-01-05 14:30:00
2025-01-05T14:30:00Z
2025/01/05 14:30:00.123
_
```

Examples (invalid):
```
"2025-01-05T14:30:00"
2025-01-05T14:30
2025-01-05T14:30:00+13:00:30
```

---

#### `duration`  
ISO‑8601 duration (days, hours, minutes, seconds).  
No years, months, or weeks.  
Fractional seconds allowed only on seconds (1–9 digits).  
**Values must not be quoted.**  
Null literal `_` allowed.

Examples (valid):
```
P2D
PT1H30M
PT1.5S
_
```

Examples (invalid):
```
"PT1H"
P1Y
PT1M30.5S
```

---

#### `timezone`  
Stores an IANA timezone identifier as text (e.g. `UTC`, `America/New_York`, `Pacific/Auckland`).  
Follows **string** quoting rules — it is the only type other than `string` that may be quoted or unquoted.  
Null literal `_` allowed.

**Implementations SHOULD store only valid IANA timezone identifiers.** Files that store arbitrary strings in a `timezone` column may break where strict IANA validation is enforced.

Examples (valid):
```
UTC
"Pacific/Auckland"
_
```

Examples (invalid):
```
"UTC
```

---

### UUID

#### `uuid`  
Canonical RFC‑4122 UUID.  
**Values must not be quoted.**  
Null literal `_` allowed.

Examples (valid):
```
550e8400-e29b-41d4-a716-446655440000
f47ac10b-58cc-4372-a567-0e02b2c3d479
_
```

Examples (invalid):
```
"550e8400-e29b-41d4-a716-446655440000"
550e8400e29b41d4a716446655440000
```

---

## Container Types

### `list<T>`
1D semantic collection.

**Rules:**
- Null literal `_` allowed at list level
- `T` must be a ScalarType or EnumType
- Containers may not nest
- Empty list is `[]` (only for dynamic-size lists)
- Null literal `_` allowed at list item level
- Whitespace ignored
- List values must not be quoted
- Prefix form MUST NOT be used for empty lists
- Prefix form cannot be used for fixed‑size lists

---

### Form Examples
```
list<T>         # dynamic-size
list<T>[3]      # fixed-size
```

---

### Example A
#### Header
```
list<string>
```

#### Valid Values
```
[red,green,blue]
[]
[_,green,_]
```

#### Invalid Values
```
"[red,green]"
"[]"
[red, [blue]]
```

---

### Example B
#### Header
```
list<int>[3]
```

#### Valid Values
```
[1,2,3]
[_,5,6]
```

#### Invalid Values
```
[1,2]          # wrong size
[1,2,3,4]      # wrong size
"[1,2,3]"      # quoted
```

---

### `arr<T>`
Structured 1D or 2D array.

**Rules:**
- Null literal `_` allowed at array level  
- `T` must be a ScalarType or EnumType
- Containers may not nest (element type cannot be list/arr)  
- Arrays may be **1D or 2D only**  
- 2D arrays must be **rectangular** (all rows same size)  
- 1D arrays contain elements of type `T`  
- 2D arrays contain rows, each of which is a 1D array of type `T`  
- Whitespace around commas and inside brackets is ignored (except inside unquoted strings)  
- **Array values must not be quoted**  
- Only **dynamic-size arrays** may be empty (`[]`)  
- Zero‑column 2D arrays are allowed for dynamic-size arrays (e.g. `[[]]`, `[[],[]]`)  
- Fixed-size 2D arrays (`arr<T>[R,C]`) require **R > 0** and **C > 0**  
- Fixed-size 1D arrays (`arr<T>[N]`) require **N > 0**  
- Prefix form MUST NOT be used for empty arrays  
- Prefix form cannot be used for fixed‑size arrays

---

### Form Examples
```
arr<T>
arr<T>[N]
arr<T>[R,C]
```

### Prefix form

[N][...] and [R,C][[...]] The prefix appears as a separate bracket
group before the value brackets, defining the array size for 1D and 2D arrays respectively.
R and C are positive integers to represent Row and Column size.
```
1D example: [3][1,2,3]
2D example: [2,3][[1,2,3],[4,5,6]]
```

---

### 1D Examples (valid values, independent of header)
```
[1,2,3,4]
[_,2,_]
_                 # array is null
[]                # valid only for arr<T>[]
```

### 2D Examples (valid values, independent of header)
```
[[1,2,3],[4,5,6],[7,8,9]]
[[true,false],[false,true]]
[[]]              # zero‑column 2D
[[],[]]           # zero‑column 2D with 2 rows
_                 # array is null
[]                # valid only for arr<T>[]
```

### Value‑level Prefix Examples (valid)
```
[3][1,2,3]
[2,3][[1,2,3],[4,5,6]]
```

### Examples (invalid)
```
"[[1,2],[3,4]]"     # quoted
[[1],[2,3]]         # ragged
```

---

### Example A — Fixed-size Array
#### Header
```
arr<int>
```

#### Valid Values
```
[1,2,3]                     # 1D
[]                          # empty allowed
[_,5,_]                     # 1D with nulls
[[1,2],[3,4]]               # 2D
[[]]                        # 2D zero‑column
[3][1,2,3]                   # 1D prefix
[2,3][[1,2,3],[4,5,6]]       # 2D prefix
```

#### Invalid Values
```
"[]"            # quoted
[1,2,]          # trailing comma
[[1],[2,3]]     # ragged 2D
[[[1]]]         # 3D not allowed
```

---

### Example B — size 1D
#### Header
```
arr<bool>[3]
```

#### Valid Values
```
[true,false,true]
[_,_,_]
```

#### Invalid Values
```
[]                   # empty not allowed
[true,false]         # wrong size
[[true,false]]       # 2D not allowed
```

---

### Example C — Fixed‑size 2D
#### Header
```
arr<int>[2,3]
```

#### Valid Values
```
[[1,2,3],[4,5,6]]
[ [_,_,_], [_,_,_] ]
```

#### Invalid Values
```
[]                       # empty not allowed
[[1,2,3]]                # wrong row count
[[1,2],[3,4]]            # wrong column count
[[1,2,3],[4,5]]          # ragged
```

---

## Enum Type

### `enum<...>`
**An inline EnumType whose EnumItems define the complete set of allowed EnumValues.**

**Rules**
- **EnumItem names and values** must be valid Identifiers (must not be quoted)  
- **Name uniqueness**: names MUST be unique under case‑insensitive comparison within the EnumType  
- **No cross-item value/name collision**: a value may match its own item's name (case-insensitive) but must not match any other item's name (case-insensitive)  
- **Values may duplicate**: two or more items may share the same value  
- **Matching is case‑insensitive** — values from data rows are compared case‑insensitively against declared names and values  
- **Uniform style**: all EnumItems MUST use the same form — either all name-only or all value=name; mixing is forbidden  
- **`_` in header definitions**: `_` is not a valid Identifier and cannot be used as an enum name or value — it is reserved as the null literal  
- **`_` in data rows**: `_` is always valid as the null literal for any enum column; it does not need to be declared  
- **Declared values only**: all other data values must match a declared name or value (case-insensitive)  

**Lookup order**: name is checked first; if no name matches, values are checked in declaration order and the first match is used.

**Encoding**: encoders MUST use name-form when values are not unique; encoders MAY use value-form only when all values are unique.

---

### Form Examples
```
enum<name1,name2,name3,...>
enum<value1=name1,value2=name2,value3=name3,...>
```

These two forms are mutually exclusive. `enum<name1,value2=name2>` is invalid.

---

### Valid Definitions

| Definition | What it shows | Example valid data values | Example invalid data values |
|---|---|---|---|
| `enum<low,medium,high>` | Name-only form | `low`, `HIGH`, `_` | `0`, `"low"` |
| `enum<0=low,1=medium,2=high>` | Numeric values | `low`, `0`, `_` | `3`, `"low"` |
| `enum<L=low,M=medium,H=high>` | Identifier values | `low`, `L`, `_` | `low2`, `"L"` |

### Invalid Definitions

| Definition | Reason |
|---|---|
| `enum<low,1=medium,high>` | Mixed style: name-only and value=name cannot be combined |
| `enum<low,low,high>` | Duplicate name |
| `enum<LOW,low,high>` | Duplicate name (case-insensitive) |
| `enum<ACT=ACTIVE,ACTIVE=WORK>` | Cross-pair collision: value `ACTIVE` (item 2) matches name `ACTIVE` (item 1) |

---

### Example A — Duplicate values and first-match decode order
#### Header
```
enum<0=ERROR,0=FAILURE,1=OK>
```

Valid — values may duplicate. Names (`ERROR`, `FAILURE`, `OK`) remain unique.  
No value collides with any name (`0` and `1` do not appear in the name set).  
When a data value matches multiple items by value, the **first matching item in declaration order** is used — e.g. `0` matches to `ERROR` in this example, not `FAILURE`.

---

### Example B — Self-pair: value matches own name (valid)
#### Header
```
enum<ERROR=ERROR,1=FAILURE>
```

Valid — the value `ERROR` matches only its own item's name. A value matching its own  
item's name is explicitly allowed.

---

## Type Aliases (Normative)

Type aliases provide alternate names for types.

### Scalar Aliases

| Canonical       | Small     | Tiny | Notes |
|-----------------|-----------|------|-------|
| int             | int       | i    | integer number |
| float           | flt       | f    | IEEE‑754 float |
| decimal         | dec       | d    | arbitrary‑precision decimal |
| bool            | bl        | b    | boolean true/false/1/0 |
| string          | str       | s    | UTF‑8 text with quoting rules |
| bytes<hex>      | hex       | bx   | hex‑encoded bytes |
| bytes<b64>      | b64       | b6   | base64‑encoded bytes |
| date            | dat       | da   | ISO‑8601 calendar date |
| time            | tm        | t    | ISO‑8601 local time |
| datetime        | dt        | dt   | ISO‑8601 datetime |
| datetimetz      | dtz       | dtz  | ISO‑8601 datetime tz |
| timestamp       | ts        | ts   | ISO‑8601 timestamp |
| duration        | dur       | du   | ISO‑8601 duration |
| timezone        | tz        | z    | IANA timezone identifier |
| uuid            | uu        | u    | RFC‑4122 UUID |

### Container Aliases

These aliases apply to the prefix of the type expression.

| Canonical       | Small     | Tiny | Notes |
|-----------------|-----------|------|-------|
| enum<…>         | en<…>     | e<…> | inline enumeration type |
| list<T>         | li<T>     | l<T> | 1D semantic list of T |
| arr<T>          | ar<T>     | a<T> | 1D or 2D structured array of T |

### Alias Rules

- Aliases must appear in lowercase in the header.  
- Aliases do not change type semantics.  
- Decoders MUST accept all aliases listed here when reading files.  
- Implementations MUST NOT invent additional aliases.  
- Encoders MUST emit canonical names by default.  
- Encoders MAY emit small or tiny aliases when explicitly configured to do so via an encoder option; the encoded output remains valid SuperCSV and all decoders MUST accept it.

### Examples

The following examples illustrate how canonical, small, and tiny aliases may be used in header rows. All three headers describe the same header definition.

#### Canonical
```
Id:int, Name:string, Base:arr<float>
```

#### Small
```
Id:int, Name:str, Base:ar<flt>
```

#### Tiny
```
Id:i, Name:s, Base:a<f>
```

---
