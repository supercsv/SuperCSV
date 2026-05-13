# SuperCSV: Encode, Decode & Validate Guide

This guide is the main Go SDK reference for **validating**, **decoding**, and **encoding** `.supr` files.

If you want a quick walkthrough first, read [End-to-End Demo](end-to-end-demo.md).

---

## Install

```bash
go get github.com/supercsv/supercsv
```

```go
import "github.com/supercsv/supercsv/supr"
```

---

## 1. Validate

Validation checks that a `.supr` file conforms to the SuperCSV v1.0 specification — correct header syntax, valid types, and well-formed data in every row.

### Quick: validate an entire file

```go
f, _ := os.Open("data.supr")
defer f.Close()

header, rowErrs, fatalErr := supr.ValidateFile(f)
if fatalErr != nil {
    log.Fatal(fatalErr) // header parse failure, I/O error, etc.
}
for _, err := range rowErrs {
    fmt.Println(err) // per-row validation errors
}
if len(rowErrs) == 0 {
    fmt.Println("Valid!", header.Columns)
}
```

### Streaming: validate row-by-row (constant memory)

For large files, use the streaming validator to avoid loading all errors into memory:

```go
f, _ := os.Open("data.supr")
defer f.Close()

v := supr.NewValidator(f)

for v.Next() {
    header := v.Header() // available after the first successful Next()
    _ = header

    if err := v.RowError(); err != nil {
        fmt.Println(err)
    }
}
if err := v.Err(); err != nil {
    log.Fatal(err)
}
if v.Valid() {
    fmt.Println("All rows valid")
}
```

If you need the parsed schema while streaming, call `v.Header()` after the first successful `Next()` call. The validator parses the header before yielding row data, so `Header()` is available during normal row iteration.

### Validator options

```go
v := supr.NewValidator(f,
    supr.WithMaxErrors(100),          // stop after 100 errors
    supr.WithMaxFieldSize(1024*1024), // 1 MB max field size
    supr.WithStrictVersion("v1.0"),   // require an explicit version directive matching v1.0
)
```

---

## 2. Decode

Decoding reads a `.supr` file and converts each row into a Go slice of typed values.

### Whole-file processing: decode everything first

```go
file, err := supr.DecodeFile("data.supr")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Columns:", file.Header.Columns)
for _, row := range file.Rows {
    fmt.Println(row) // []any with typed values
}
```

Use `supr.DecodeFile` when you want all rows loaded into memory for later iteration, random access, or multi-pass processing.

### Process rows as they stream

```go
f, _ := os.Open("data.supr")
defer f.Close()

d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeFull))
header := d.Header() // available before iteration if you need schema information
fmt.Println("columns:", len(header.Columns))

for d.Next() {
    row := d.Row() // []any — valid only until next Next() call
    id := row[0].(int64)
    name := row[1].(string)
    status := row[2].(supr.EnumField)
    fmt.Printf("id=%d name=%s status=%s\n", id, name, status.Name())
}
if err := d.Err(); err != nil {
    log.Fatal(err)
}
```

> **Important: Row ownership.** The slice returned by `Row()` is reused on the next call to `Next()`. If you need to store rows, copy the slice before calling `Next()` again.

Use streaming decode when you want constant-memory row-by-row processing.

### Streaming vs whole-file decode

Use streaming decode when you want constant-memory row-by-row processing. Use `DecodeFile` when you want all rows loaded into memory for later iteration, random access, or multi-pass processing.

### Decoder options

```go
d := supr.NewDecoder(f,
    supr.WithDecodeMode(supr.DecodeFull), // default — full decode
    supr.WithDecodeMaxFieldSize(1024*1024),
    supr.WithDecodeStrictVersion("v1.0"), // require an explicit version directive matching v1.0
)
```

### Decode modes

`WithDecodeMode` controls how much type decoding is applied to each row:

| Mode | Scalar result type | Container result type | Enum result type | When to use it |
|------|--------------------|-----------------------|------------------|----------------|
| `supr.DecodeFull` | decoded Go values such as `int64`, `float64`, `string` | decoded `[]any` or `[][]any` | `supr.EnumField` | normal typed processing |
| `supr.DecodeShallow` | decoded Go values such as `int64`, `float64`, `string` | raw field text as `string` | `supr.EnumField` | typed scalar processing when container parsing can be deferred |
| `supr.DecodeRaw` | raw `[]byte` | raw `[]byte` | raw `[]byte` | passthrough, auditing, or custom downstream parsing |

Use `DecodeShallow` when you need scalar values but want to defer container parsing. Use `DecodeRaw` when you only need to inspect or forward field text without allocating container slices.

```go
// Shallow: scalars decoded, containers left as raw strings
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeShallow))

// Raw: all fields returned as []byte — no type decoding at all
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeRaw))
```

> **Note:** In `DecodeShallow` and `DecodeRaw` modes the row slice is still reused on the next `Next()` call — the row-ownership rule applies regardless of mode.

### Decode mode examples

Given this schema:

```go
// id:int, status:enum<pending,active,done>, tags:list<string>
```

And a row like:

```go
// 1,active,[math,sci]
```

The row values differ by decode mode:

**DecodeFull** (default)

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeFull))

row := d.Row()
id := row[0].(int64)
status := row[1].(supr.EnumField)
tags := row[2].([]any)

fmt.Println(id)            // 1
fmt.Println(status.Name()) // active
fmt.Println(status.Index()) // 1
fmt.Println(tags)          // []any{"math", "sci"}
```

**DecodeShallow**

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeShallow))

row := d.Row()
id := row[0].(int64)
status := row[1].(supr.EnumField)
tags := row[2].(string)

fmt.Println(id)            // 1
fmt.Println(status.Name()) // active
fmt.Println(tags)          // [math,sci]
```

**DecodeRaw**

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeRaw))

row := d.Row()
id := row[0].([]byte)
status := row[1].([]byte)
tags := row[2].([]byte)

fmt.Println(string(id))     // 1
fmt.Println(string(status)) // active
fmt.Println(string(tags))   // [math,sci]
```

### Working with decoded enums

Enum fields decode to `supr.EnumField`, which carries the index, label name, and optional coded value — no header lookup required:

```go
ef := row[col].(supr.EnumField)
fmt.Println(ef.Index()) // 0-based position in enum declaration, e.g. 0
fmt.Println(ef.Name())  // label name, e.g. "red"
fmt.Println(ef.Value()) // coded value if declared, e.g. "7"; empty string for name-only enums
```

> **Note:** In `DecodeRaw` mode enum fields are returned as `[]byte`, not `supr.EnumField`.

### Working with decoded containers

List and array fields decode to `[]any`:

```go
tags := row[col].([]any)
for _, tag := range tags {
    fmt.Println(tag.(string))
}
```

2D arrays decode to `[][]any`:

```go
matrix := row[col].([][]any)
for _, rowVals := range matrix {
    for _, v := range rowVals {
        fmt.Print(v.(int64), " ")
    }
    fmt.Println()
}
```

---

## 3. Encode

Encoding writes a `.supr` file from a Go header and rows.

Encoders MUST always include a version directive (`((SuperCSV v1.0))`) as the first line of output. The reference implementation does this automatically.

### Quick: encode an entire file

```go
header := supr.NewHeader([]supr.Column{
    {Name: "id", Type: supr.Int},
    {Name: "name", Type: supr.String},
    {Name: "score", Type: supr.Float},
    {Name: "tags", Type: supr.List(supr.String)},
})

rows := [][]any{
    {int64(1), "Alice", 95.5, []any{"math", "sci"}},
    {int64(2), "Bob", 87.3, []any{"art"}},
}

f, _ := os.Create("output.supr")
defer f.Close()

err := supr.EncodeFile(f, &supr.File{Header: header, Rows: rows})
if err != nil {
    log.Fatal(err)
}
```

### Streaming: encode row-by-row

```go
f, _ := os.Create("output.supr")
defer f.Close()

enc := supr.NewFileEncoder(header, f)
if err := enc.WriteHeader(); err != nil {
    log.Fatal(err)
}
for _, row := range rows {
    if err := enc.WriteRow(row); err != nil {
        log.Fatal(err)
    }
}
if err := enc.Close(); err != nil {
    log.Fatal(err)
}
```

### Header type aliases

By default the encoder writes canonical type names in the header. If you want shorter aliases on the wire, use one of the header-format options:

```go
enc := supr.NewFileEncoder(header, f, supr.WithSmallHeaderTypes())
// Example: int -> i, string -> str

enc := supr.NewFileEncoder(header, f, supr.WithTinyHeaderTypes())
// Example: int -> i, string -> s
```

These options only affect the serialized header text. They do not change the in-memory schema or the decoded Go value types.

### Building rows

Each row is a `[]any` with exactly one element per column. Use the correct Go types:

```go
row := []any{
    int64(42),                           // int
    3.14,                                // float (float64)
    "hello",                             // string
    true,                                // bool
    nil,                                 // null (any type)
    []any{int64(1), int64(2), int64(3)}, // list<int>
    [][]any{                             // arr<float>[2,3]
        {1.0, 2.0, 3.0},
        {4.0, 5.0, 6.0},
    },
}
```

### Encoding enums

Enums accept the label string:

```go
// For enum(red,green,blue):
row := []any{"red"} // by label
```

### Encoding nulls

Use `nil` for any nullable field, including containers. A null container is a missing value — it encodes to `_` on the wire and decodes back to `nil`:

```go
row := []any{nil, nil, nil} // all fields null — encodes to: _,_,_
```

### Null vs empty containers

`nil` and an empty container are distinct values:

| Go value | SuperCSV form | Decoded back as | Meaning |
|---|---|---|---|
| `nil` | `_` | `nil` | absent / null field |
| `[]any{}` | `[]` | `[]any{}` | present but empty list/array |

```go
// Null container — the field is absent
row := []any{nil}                     // encodes to: _

// Empty container — the field is present but has no elements
row := []any{[]any{}}                 // encodes to: []
```

Empty containers are only valid for dynamic-size (unfixed) lists and arrays. For fixed-size containers such as `list<int>[3]` or `arr<int>[3]`, `[]` is a validation error because the element count must match the declared size.

### Encoding temporal and special types

Temporal types (`date`, `time`, `timestamp`, `datetime`, `datetimetz`, `duration`) and `decimal` accept string values in their canonical format:

```go
row := []any{
    "2024-01-15",                  // date
    "14:30:00",                    // time
    "2024-01-15T14:30:00Z",        // timestamp
    "P1DT2H30M",                   // duration
    "123.456",                     // decimal
}
```

---

## Type Reference

This section is the Go-facing reference for SuperCSV types. Use it to answer: what Go value do I pass when encoding, and what Go value do I get back when decoding?

For wire-level rules, exact lexical formats, and full format constraints, use the spec documents as the authority:

- [Type Table](../internal/v1_0/spec/supercsv-type-table-v1.0.md)
- [Quick Reference](../internal/v1_0/spec/supercsv-quick-reference-v1.0.md)

### Go usage patterns by type category

Most Go usage falls into four groups:

#### 1. Native scalar values

These encode and decode as ordinary Go values:

```go
row := []any{
    int64(42),     // int
    3.14,          // float
    "hello",      // string
    true,          // bool
    nil,           // null for any nullable field
}

id := row[0].(int64)
score := row[1].(float64)
name := row[2].(string)
active := row[3].(bool)
```

#### 2. Canonical string-form scalar values

These are encoded from strings in canonical SuperCSV form. On decode they come back as the SDK's typed wrapper values or strings, as shown in the table below.

```go
row := []any{
    "123.456",                    // decimal
    "2024-01-15",                 // date
    "14:30:00",                   // time
    "2024-01-15T14:30:00Z",       // timestamp
    "2024-01-15T14:30:00",        // datetime
    "2024-01-15T14:30:00+05:00",  // datetimetz
    "P1DT2H30M",                  // duration
    "America/New_York",           // timezone
    "550e8400-e29b-41d4-a716-446655440000", // uuid
    "4a6f686e",                   // byteshex
    "Sm9obg==",                   // bytesb64
}
```

#### 3. Enums

Encode enums using the label string. Decode them as `supr.EnumField` unless you are in `DecodeRaw` mode.

```go
priorityType := supr.Enum([]supr.EnumValue{
    {Name: "low"},
    {Name: "medium"},
    {Name: "high"},
})

row := []any{"medium"} // encode by label

priority := decodedRow[0].(supr.EnumField)
fmt.Println(priority.Index()) // 1
fmt.Println(priority.Name())  // medium
fmt.Println(priority.Value()) // empty for name-only enums
```

#### 4. Containers

Encode 1D containers as `[]any` and 2D arrays as `[][]any`. In `DecodeFull`, they come back in the same structural shape.

```go
row := []any{
    []any{"math", "sci"},                  // list<string>
    []any{int64(1), int64(2), int64(3)},      // arr<int>[3]
    [][]any{{1.0, 2.0}, {3.0, 4.0}},          // arr<float>[2,2]
}

tags := decodedRow[0].([]any)
scores := decodedRow[1].([]any)
matrix := decodedRow[2].([][]any)
```

### Type-coverage guidance

When you want a type-coverage demo, model it as one header with many columns, not one row per type. A practical sample often uses 5-10 rows and one column for each scalar, enum, and container variant you want to demonstrate.

Treat row count and type count as separate choices. Add rows when you want realistic sample data. Add columns when you want broader schema coverage.

For broad format coverage, include at least:

- one column for each scalar type you want to demonstrate
- one enum column, preferably with both labels and coded values if that distinction matters for the example
- one dynamic container and one fixed-size container
- a few null values so the example shows absence distinctly from empty containers

| SuperCSV type | Go decode type | Go encode type | Example |
|---|---|---|---|
| `int` | `int64` | `int64` | `int64(42)` |
| `float` | `float64` | `float64` | `3.14` |
| `string` | `string` | `string` | `"hello"` |
| `bool` | `bool` | `bool` | `true` |
| `decimal` | `supr.DecimalValue` | `string` | `"99.99"` |
| `date` | `supr.DateValue` | `string` | `"2024-01-15"` |
| `time` | `supr.TimeValue` | `string` | `"14:30:00"` |
| `timestamp` | `supr.TimestampValue` | `string` | `"2024-01-15T14:30:00Z"` |
| `datetime` | `supr.DatetimeValue` | `string` | `"2024-01-15T14:30:00"` |
| `datetimetz` | `supr.DatetimeTZValue` | `string` | `"2024-01-15T14:30:00+05:00"` |
| `duration` | `supr.DurationValue` | `string` | `"P1DT2H30M"` |
| `timezone` | `string` | `string` | `"America/New_York"` |
| `uuid` | `string` | `string` | `"550e8400-..."` |
| `byteshex` | `string` | `string` | `"4a6f686e"` |
| `bytesb64` | `string` | `string` | `"Sm9obg=="` |
| `enum(...)` | `supr.EnumField` | `string` | `.Index()==0`, `.Name()=="red"` |
| `list<T>` | `[]any` | `[]any` | `[]any{int64(1), int64(2)}` |
| `list<T>` (empty) | `[]any` | `[]any{}` | `[]any{}` (non-nil, len 0) |
| `arr<T>[N]` | `[]any` | `[]any` | `[]any{1.0, 2.0, 3.0}` |
| `arr<T>[R,C]` | `[][]any` | `[][]any` | `[][]any{{1.0, 2.0}, {3.0, 4.0}}` |
| null (any type) | `nil` | `nil` | `nil` |

---

## Compact Round-Trip Example

If you want a full project walkthrough with setup, CLI validation, streaming decode, whole-file decode, and decode-mode comparison in one place, use [End-to-End Demo](end-to-end-demo.md). The example below is the shorter reference version.

```go
package main

import (
    "bytes"
    "fmt"
    "log"
    "strings"

    "github.com/supercsv/supercsv/supr"
)

func main() {
    // 1. Build a header
    header := supr.NewHeader([]supr.Column{
        {Name: "id", Type: supr.Int},
        {Name: "name", Type: supr.String},
        {Name: "score", Type: supr.Float},
        {Name: "tags", Type: supr.List(supr.String)},
    })

    // 2. Encode rows to a buffer
    rows := [][]any{
        {int64(1), "Alice", 95.5, []any{"math", "sci"}},
        {int64(2), "Bob", 87.3, []any{"art"}},
        {int64(3), "Charlie", 92.1, []any{"math", "art", "music"}},
    }

    var buf bytes.Buffer
    if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: rows}); err != nil {
        log.Fatal(err)
    }
    encoded := buf.String()
    fmt.Println("Encoded:")
    fmt.Println(encoded)

    // 3. Validate
    _, rowErrs, fatalErr := supr.ValidateFile(strings.NewReader(encoded))
    if fatalErr != nil {
        log.Fatal(fatalErr)
    }
    if len(rowErrs) > 0 {
        log.Fatal("Validation errors:", rowErrs)
    }
    fmt.Println("Validation: PASS")

    // 4. Decode
    d := supr.NewDecoder(strings.NewReader(encoded))
    file, err := d.DecodeFile()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\nDecoded %d rows:\n", len(file.Rows))
    for i, row := range file.Rows {
        fmt.Printf("  Row %d: id=%v name=%v score=%v tags=%v\n",
            i, row[0], row[1], row[2], row[3])
    }
}
```

Output:

```
Encoded:
((SuperCSV v1.0))
id:int,name:string,score:float,tags:list<string>
1,Alice,95.5,[math,sci]
2,Bob,87.3,[art]
3,Charlie,92.1,[math,art,music]

Validation: PASS

Decoded 3 rows:
  Row 0: id=1 name=Alice score=95.5 tags=[math sci]
  Row 1: id=2 name=Bob score=87.3 tags=[art]
  Row 2: id=3 name=Charlie score=92.1 tags=[math art music]
```
