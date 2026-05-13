# SuperCSV Validator & Decoder Model

## Scope

This document defines the behavioral model for SuperCSV validators and decoders.
It describes execution modes, decoding modes, output shapes, and error semantics.
It does **not** redefine the grammar or the type table; those remain canonical in the v1.0 spec.

## Terms

- **Physical line**: a 1-based line number in the input text file.
- **Data Row**: the full CSV parsed by the CSV layer (may span multiple physical lines via trailing-comma continuation or quoted fields containing newlines). The CSV layer is responsible for reconstructing Data Rows.
- **Field**: one cell value in a Data Row.
- **HeaderDef**: the SuperCSV header definition row describing columns and types (may span multiple physical lines via trailing-comma continuation).
- **Scalar**: a non-container, non-enum value type (e.g., `int`, `datetime`).
- **Enum**: a schema-defined finite set of names (optionally with identifier values).
- **Container**: `list<T>` or `arr<T>` (1D or 2D), where `T` is a scalar or enum.

## Processing pipeline (conceptual)

Implementations follow this logical pipeline:

1. **CSV layer** reads data rows and fields.
2. **Version/router** (optional) enforces or interprets any version directives/configuration.
3. **Header parser** parses and validates the HeaderDef (including type declarations).
4. For each data row:
  - **Structural validation** (field count, CSV well-formedness)
  - **Type validation** (scalars/enums/containers)
  - **Decoding** (optional; depends on decoder mode)
5. **Error reporting** emits recoverable errors as rows, and may stop on fatal errors.

Standard mode and strict-version enforcement influence steps 1–4. Decoder mode influences step 4.

---

# 1. Validation and processing options

Validators and decoders always apply the full v1.0 canonical rules — there is effectively one validation mode. The following option can be layered on top:

**Strict-version enforcement** (opt-in): requires the version directive to be present, appear first, and match a specified version string. Without this option, version directives are parsed and recorded but not enforced.

In v1.0, only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning in this model. Other `(( ... ))` metadata blocks may be recognised so positional and structural rules can be applied correctly, but they have no defined effect on validation, decoding, or output shape.

## 1.1 Strict-version option

- By default, no version enforcement is applied.
- Pass `--strict-version <version>` (CLI) or `WithStrictVersion(v)` / `WithDecodeStrictVersion(v)` (Go SDK) to enable enforcement.
- If enforcement is enabled and the directive is absent or mismatched, processing fails immediately (fatal error).

## 1.2 Version directive behavior

When a version directive is present, the file MUST be interpreted according to the semantics of the declared version.

When no version directive is present, the file MUST be interpreted according to the latest version of the SuperCSV specification supported by the processing environment, and MUST be rejected if it uses constructs not defined in that version.

Strict-version mode requires a version directive. The directive MUST appear first and MUST match the version specified by the mode. If the directive is absent or mismatched, processing fails immediately.

**Rationale:**
The version directive is an explicit contract. When present, it defines the exact semantics to apply. When absent, the latest supported version is assumed to provide a stable, predictable interpretation model and clear rejection of unsupported constructs.

## 1.3 UTF-8 BOM handling

A UTF-8 BOM (`EF BB BF`) at file start is silently stripped before parsing. UTF-16 and UTF-32 BOMs are not valid UTF-8 and MUST cause an error.

## 1.4 Validation model

Validation is the process of checking:

- **Header/schema correctness** (column names, type declarations, and any schema constraints)
- **Row structure** (field count)
- **Field conformance** (scalar/enum/container validity for the declared type)

### Streaming validator behavior (Go implementation)

- Header parsing/validation happens before the first data row; header failure is **fatal**.
- `Next()` advances through rows even if a row is invalid.
- `RowError()` reports the validation error for the current row (the first error discovered for that row).
- `Err()` reports fatal errors (CSV parse errors, header errors, or stopping conditions like max-error limits).

### Decoder interaction

- The decoder always validates as part of decoding.
- A decode failure is treated as **fatal** for streaming decode (iteration stops and `Err()` becomes non-nil).

---

# 2. Decoder output modes

The Go reference implementation exposes a streaming decoder with a header and rows.

- Streaming output is **positional**: a decoded row is a `[]any` aligned with the header’s column order.
- The header provides the mapping from index → column name/type.

This positional shape is a deliberate performance design choice.

| Mode | Scalars | Enums | Containers | Allocations |
|------|---------|-------|------------|-------------|
| Raw | raw `[]byte` | raw `[]byte` | raw `[]byte` | allocates |
| Shallow | parsed | `EnumField` | raw `string` | mixed |
| Full | parsed | `EnumField` | parsed | pooled / zero-alloc |

## 2.1 DecodeRaw

DecodeRaw performs no semantic parsing. It returns raw field bytes (after CSV parsing) and applies strict validation.

- Row shape: `[]any`
- Scalar fields: `[]byte`
- Enum fields: `[]byte`
- Container fields: `[]byte` (container literals are not parsed)
- Nulls: `nil`
- Allocation model: fresh allocations for field bytes

## 2.2 DecodeShallow

DecodeShallow parses **scalars** (and enums) but does not parse containers.

- Row shape: `[]any`
- Scalars: decoded Go values (see scalar decoding table)
- Enums: `EnumField` — carries index, name, and value; access via `.Index()`, `.Name()`, `.Value()`
- Containers: `string` (container literal text)
- Nulls: `nil`

DecodeShallow is for “typed scalars, raw containers”.

## 2.3 DecodeFull

DecodeFull fully parses scalars, enums, and containers.

- Row shape: `[]any`
- Scalars: decoded Go values (see scalar decoding table)
- Enums: `EnumField` — carries index, name, and value; access via `.Index()`, `.Name()`, `.Value()`
- Containers:
  - `list<T>` → `[]any`
  - `arr<T>` (1D) → `[]any`
  - `arr<T>` (2D) → `[][]any`
- Nulls: `nil`

DecodeFull uses buffer pooling and is designed for zero allocations in hot paths.

`EnumField` methods:

```
ef := row[col].(EnumField)
ef.Index()  // int32 — 0-based declaration position
ef.Name()   // string — declared name (e.g. "red")
ef.Value()  // string — declared value (e.g. "1"); empty string for name-only enums
```

No header lookup is required — all three fields are carried directly on the decoded value.

## 2.4 Row ownership (streaming contract)

Implementations MAY reuse storage for row buffers across iterations.

- The row returned from a streaming decoder is only valid until the next call to `Next()`.
- If you store rows, you must copy the slice.
- In pooling modes, you may need deep copies of container elements.

---

# 3. Scalar decoding model

Scalar validation rules and canonical literal forms are defined in the v1.0 type table. This section describes the **decoded output types** and the general decode contract.

## 3.1 Decode contract

Scalar decoders:

- accept a validated byte slice
- return `(value, ok)`
- MUST be pure and deterministic
- MUST NOT mutate input

In the reference implementation, some decoders allocate output (e.g., base64/hex decode), while others do not.

## 3.2 Custom scalar types (canonical parsed forms)

The decoder returns a set of custom scalar structs for temporal and decimal types. These represent the **validated structure** of the literal without lossy conversion to standard library types. This is important for implementations in other languages that may have different temporal and numeric representations.

Current custom scalar structs:

- `Date`
- `Time`
- `Timestamp`
- `Datetime`
- `DatetimeTZ`
- `Duration`
- `Decimal`

These are returned in DecodeShallow and DecodeFull.

### Date

Represents an ISO-8601 calendar date.

- Accepted forms: `YYYY-MM-DD` or `YYYY/MM/DD`
- Month: 01–12
- Day: validated against month length and leap-year rules

Go representation:

```go
type Date struct {
    Year  int
    Month int
    Day   int
}
```

### Time

Represents an ISO-8601 local time with optional fractional seconds.

- Accepted forms: `HH:MM:SS` or `HH:MM:SS.fffffffff`
- Fractional seconds: 0–9 digits, stored as nanoseconds

```go
type Time struct {
    Hour   int
    Minute int
    Second int
    Nanos  int
}
```

### Timestamp

Represents an ISO-8601 timestamp where the timezone is optional.

- Separator: space or `T`
- Timezone: absent, `Z`, or `±HH:MM`

```go
type Timestamp struct {
    Year          int
    Month         int
    Day           int
    Hour          int
    Minute        int
    Second        int
    Nanos         int
    OffsetSeconds int // absent timezone encodes as 0
}
```

### Datetime

Represents an ISO-8601 datetime WITHOUT timezone.

- Separator: space or `T`
- Timezone: forbidden

```go
type Datetime struct {
    Year   int
    Month  int
    Day    int
    Hour   int
    Minute int
    Second int
    Nanos  int
}
```

### DatetimeTZ

Represents an ISO-8601 datetime WITH required timezone.

- Separator: space or `T`
- Timezone: required (`Z` or `±HH:MM`)

```go
type DatetimeTZ struct {
    Year          int
    Month         int
    Day           int
    Hour          int
    Minute        int
    Second        int
    Nanos         int
    OffsetSeconds int
}
```

### Duration

Represents an ISO-8601 duration limited to days/time components.

```go
type Duration struct {
    Days    int
    Hours   int
    Minutes int
    Seconds int
    Nanos   int
}
```

### Decimal

Represents an arbitrary-precision decimal while preserving the literal form.

```go
type Decimal struct {
    Value string
}
```

Exponent notation (e.g., `1e3`) is not allowed for decimal; see the type table for the full rules. The literal is preserved exactly as written.

## 3.3 Scalar decoder table (decoded types)

| Scalar type       | Decoded Go type       | Notes |
|------------------|------------------------|-------|
| `int`            | `int64`                | Canonical integer rules as per type table. |
| `float`          | `float64`              | Supports NaN/Inf variants per spec. |
| `decimal`        | `Decimal`              | Preserves literal; no exponent notation. |
| `bool`           | `bool`                 | `true` or `false` only. |
| `string`         | `string`               | After CSV unquoting. |
| `date`           | `Date`                 | Validated Y/M/D. |
| `time`           | `Time`                 | Validated H/M/S and nanos. |
| `timestamp`      | `Timestamp`            | Optional timezone offset. |
| `datetime`       | `Datetime`             | Timezone forbidden. |
| `datetimetz`     | `DatetimeTZ`           | Timezone required. |
| `duration`       | `Duration`             | ISO-8601 duration subset. |
| `uuid`           | `[16]byte`             | Canonical UUID textual form decoded to bytes. |
| `bytes<hex>`     | `[]byte`               | Allocates output. |
| `bytes<b64>`     | `[]byte`               | Allocates output. |
| `timezone`       | `string`               | A timezone identifier scalar type (schema-defined). |
| `enum<...>`      | `EnumField`            | Carries index (schema order), declared name, and declared value. Access via `.Index()`, `.Name()`, `.Value()`. |

---

# 4. Container decoding model

The formal syntax of containers is defined by the v1.0 grammar and the type table. This section describes the decode semantics and important conformance constraints.

## 4.1 Container types

- `list<T>`: variable-length
- `arr<T>`: array type (1D or 2D), with optional fixed dimensions

## 4.2 Schema nesting constraint (conformance)

In SuperCSV v1.0 schemas, **nested container types are not allowed**.

- Allowed: `list<int>`, `arr<string>[3]`, `arr<float>[2,2]`
- Forbidden: `list<list<int>>`, `arr<list<int>>`, `list<arr<int>>`, `arr<arr<int>>`

This is a schema/type-system constraint, independent of how the 2D array literal is written.

## 4.3 2D arrays are not “nested containers”

A 2D array literal uses nested brackets (e.g., `[[1,2],[3,4]]`) but it is still a single container value of type `arr<T>[R,C]` (or dynamic 2D `arr<T>`).

The inner bracket groups represent rows; they are not schema-level nested container types. The grammar uses nested brackets for 2D arrays, but this is syntactic grouping, not schema nesting.

## 4.4 Decode outputs

In DecodeFull:

- `list<T>` → `[]any`
- `arr<T>` (1D) → `[]any`
- `arr<T>` (2D) → `[][]any`

Elements are decoded using the scalar/enum decoders for `T`.

When `T` is `enum`, each element in the decoded `[]any` is an `EnumField` — the same `.Index()`, `.Name()`, `.Value()` methods apply as for scalar enum columns.

```
elements := row[col].([]any)
for _, el := range elements {
    ef := el.(EnumField)
    ef.Index()  // int32 — 0-based declaration position
    ef.Name()   // string — declared name
    ef.Value()  // string — declared value; empty for name-only enums
}
```

In DecodeShallow:

- containers are returned as `string` literals

In DecodeRaw:

- containers are returned as raw `[]byte` literals

## 4.5 Null elements

- Null elements inside containers are represented using `_`.
- In decoded output, null elements become `nil`.

## 4.6 Prefix notation

Container literals may support prefix size notation (as defined by the grammar/type table), for example:

- `[3][10,20,30]`
- `[2,2][[1,2],[3,4]]`

## 4.7 Container error conditions

Container decoding/validation fails if:

- declared fixed sizes do not match the actual number of elements
- size prefix does not match the actual literal
- an element fails validation/decoding
- the container literal is malformed

---

# 5. Error reporting model

This section defines the external error format emitted by validators/decoders.

## 5.1 External error row format

The external error format is exactly three comma-separated fields, with no header row:

1. **Line** (int): 1-based physical line number where the error occurred
2. **ErrorSection** (string): where the error occurred (field name, row/header section)
3. **ErrorMsg** (string): human-readable message derived from the error template

## 5.2 ErrorSection

- Scalar column errors: the field/column name (e.g., `Price`)
- Container element errors: field name plus indices, e.g.:
  - `Tags(4)` for the 4th element
  - `Matrix(2,3)` for row 2, column 3
- Header-level errors: `headerErr`
- Row-level errors (e.g., wrong number of columns): `rowErr`
- CSV parse errors: `csvParseError`

### Indexing convention

- All indices in external errors are **1-based**.

## 5.3 ErrorMsg

- MUST be a valid SuperCSV string literal.
- MUST follow quoting rules (quoted if it contains commas/quotes/etc.).
- Message templates SHOULD prefer single quotes for quoting embedded values.

## 5.4 Fatal vs recoverable

### Recoverable errors

Recoverable errors allow continued processing and collection of multiple errors. Recoverable errors MUST NOT stop iteration unless an implementation-defined limit is reached.

Recoverable error categories:

- invalid scalar values
- invalid enum values
- invalid container syntax
- column count mismatch (when CSV row parsing itself succeeded)

### Fatal errors

Fatal errors stop validation/decoding immediately:

- CSV parse errors (malformed CSV, unclosed quotes, etc.)
- header/schema errors
- strict-version enforcement failures
- empty file (no header)

Implementations MAY prefix fatal external messages with `FATAL:` for human visibility.

## 5.5 Exit codes (validator CLIs)

Validators SHOULD use:

- `0`: success (no errors)
- `1`: validation errors (recoverable errors present)
- `2`: fatal error (validation stopped)

## 5.6 Examples

Recoverable errors:

```text
8, Price, "invalid int value: 'abc'"
14, Tags(4), "invalid enum label: 'blueish'"
12, Matrix(2,3), "invalid int value: '/'"
9, rowErr, "expected 5 columns, got 6"
```

Fatal errors:

```text
2, headerErr, "FATAL: invalid identifier: ' Name'"
5, csvParseError, "FATAL: CSV parse error: unclosed quoted field"
1, headerErr, "FATAL: strict-version 1.0: file has no version"
```

---

# 6. Pointers to canonical docs

- Grammar: `internal/v1_0/spec/grammar.ebnf`
- Main spec: `internal/v1_0/spec/supercsv-spec-v1.0.md`
- Type table: `internal/v1_0/spec/supercsv-type-table-v1.0.md`

---

# Appendix A: Notes on correctness vs performance

- Validation and decoding are deliberately separated conceptually, but in practice decoding relies on the same validated invariants.
- Hot paths prioritize byte-level logic and avoid allocations.
- Streaming decoders may reuse buffers; batch APIs necessarily allocate.
