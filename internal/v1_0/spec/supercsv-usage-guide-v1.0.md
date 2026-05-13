# SuperCSV Usage Guide (v1.0)

This is the **how-to** guide for using the SuperCSV tooling and Go SDK.

- For the *canonical language specification*, see `opensource/internal/v1_0/spec/`.
- For the *conceptual mental model* (data row, field, modes, outputs), see `supercsv-model-v1.0.md`.

---

## 1. CLI: `supercsv-validate`

### Install

From source (repo checkout):

```bash
go build ./cmd/supercsv-validate
```

Or install (module path):

```bash
go install github.com/xras/supercsv/cmd/supercsv-validate@latest
```

### Run

```bash
supercsv-validate [flags] file.supr
```

### Exit codes

- `0` — valid (no errors)
- `1` — validation errors found
- `2` — fatal validation failure, invalid args, or an I/O/parse error

### Flags (current implementation)

- `--version` — print tool version and exit
- `-q`, `--quiet` — suppress per-error output (still prints a one-line summary)
- `--max-field-size N` — maximum field size in bytes (`0` = unlimited)
- `--max-errors N` — stop after N errors (`0` = unlimited; if the limit is reached, a fatal “validation stopped” error is emitted and the tool exits `2`)
- `--strict-version v1.0` — require a specific SuperCSV version header (must match the file’s `Version:` directive value)
- `--json` — emit newline-delimited JSON errors instead of SUPR

Notes:

- Only `-q` has a short alias in the current implementation.
- `--max-errors` values `< 0` are treated as `0` (unlimited).

### Output formats

#### Default (SUPR)

By default, `supercsv-validate` emits a SUPER stream with a header row and exactly three columns:

- `Line:int`
- `ErrorSection:string`
- `ErrorMsg:string`

Example:

```text
Line:int,ErrorSection:string,ErrorMsg:string
8,Price,"invalid int value: 'abc'"
14,Tags(4),"invalid enum name: 'blueish'"
```

Notes:

- Indices in `ErrorSection` for containers are **1-based** (e.g., `Tags(4)`, `Matrix(2,3)`).
- A final summary line is printed to **stderr**.

#### JSON (`--json`)

`--json` emits NDJSON (one JSON object per line):

```json
{"line":8,"errorSection":"Price","errorMsg":"invalid int value: 'abc'"}
```

#### Quiet mode (`--quiet`)

Suppresses per-error output; the tool still returns an appropriate exit code and prints a summary line to stderr.

---

## 2. Go SDK: Validation

### Streaming validation (constant memory)

Use `NewValidator` to scan row-by-row:

```go
v := supr.NewValidator(r,
    supr.WithMaxErrors(100),
    supr.WithMaxFieldSize(1024*1024),
    supr.WithStrictVersion("v1.0"),
)

for v.Next() {
    if err := v.RowError(); err != nil {
        // Per-row validation error
        // Typically *supr.ValidationError
        log.Println(err)
    }
}

if err := v.Err(); err != nil {
    // Fatal errors / stopping conditions (CSV parse, invalid header, max errors reached, etc.)
    log.Fatal(err)
}

if v.Valid() {
    log.Println("All rows valid")
}
```

Behavior notes:

- Header parsing/validation happens during `NewValidator`.
- `Next()` reads *and validates* one row.
- `RowError()` reports the error for the current row (if any).

### Batch validation (convenience)

If you just want `header + []errors`:

```go
header, errs, err := supr.ValidateFile(r)
```

This is a convenience wrapper around the streaming validator.

---

## 3. Go SDK: Decoding

### Streaming decode

```go
d := supr.NewDecoder(r,
    supr.WithDecodeMode(supr.DecodeFull),
    supr.WithDecodeMaxFieldSize(1024*1024),
    supr.WithDecodeStrictVersion("v1.0"),
)

header := d.Header()
_ = header

for d.Next() {
    row := d.Row() // []any
    // Process immediately; storage may be reused on the next Next()
    _ = row
}

if err := d.Err(); err != nil {
    log.Fatal(err)
}
```

### Decode modes

- `DecodeRaw`
  - scalars/enums/containers returned as raw `[]byte`
- `DecodeShallow`
  - scalars decoded to Go values
  - enums decoded to `EnumField`
  - containers returned as `string` literals
- `DecodeFull` (default)
  - scalars decoded to Go values
  - enums decoded to `EnumField`
  - containers decoded to `[]any` / `[][]any` with pooling

### Decoded enum fields

In `DecodeShallow` and `DecodeFull` modes, enum fields decode to `EnumField` — a value type that carries the index, name, and optional coded value directly. No header lookup is required:

```go
ef := row[col].(supr.EnumField)
ef.Index()  // int32  — 0-based declaration position (e.g. 0)
ef.Name()   // string — declared name (e.g. "red")
ef.Value()  // string — declared value (e.g. "1"); empty string for name-only enums
```

In `DecodeRaw` mode, enum fields are returned as raw `[]byte` (no decoding applied).

### Enum elements in containers

When a container's element type is `enum` (e.g. `list<enum<pending,active,done>>`), each element in the decoded `[]any` is an `EnumField` — the same methods apply:

```go
elements := row[col].([]any)
for _, el := range elements {
    ef := el.(supr.EnumField)
    ef.Name()   // "pending", "active", etc.
    ef.Index()  // 0, 1, 2, ...
    ef.Value()  // "" for name-only enums
}
```

In `DecodeRaw` / `DecodeShallow` modes, container elements are not individually decoded (the container is returned as raw `[]byte` or `string`).

### Row ownership (important)

The slice returned by `Row()` is only valid until the next `Next()`.

- For `DecodeRaw` / `DecodeShallow`: a shallow copy of the row slice is typically sufficient if you need to store it.
- For `DecodeFull`: containers may reference pooled buffers; deep copy is required if you persist rows/containers.

### Batch decode

```go
file, err := supr.DecodeFile(path)
```

Or with a `Decoder` instance:

```go
file, err := d.DecodeFile()
```

---

## 4. Go SDK: Encoding

Use `FileEncoder` to write canonical SuperCSV output:

```go
enc := supr.NewFileEncoder(header, w) // or WithSmallHeaderTypes / WithTinyHeaderTypes
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

Rules:

- Call `WriteHeader()` exactly once before any rows.
- Each `row` must have exactly `len(header.Columns)` elements.
- Use `nil` for null values.
- Container values are provided as Go slices (e.g., `[]any`, `[][]any`) matching the declared type.

### Building a row (`[]any`)

`WriteRow` always takes a single `[]any` whose length matches the header’s column count.

Given a header like:

```text
ID:int,Active:bool,Name:string,Tags:list<int>,Matrix:arr<float>[2,2]
```

You would construct a row like:

```go
row := []any{
    int64(123),
    true, // encoded as 1/0 in canonical output
    "Alice",
    []any{int64(10), int64(20), nil}, // list<int> (nil encodes as _)
    [][]any{
        {float64(1.25), float64(2.5)},
        {float64(3.75), float64(4.0)},
    },
}
```

Notes:

- `int` must be `int64`, and `float` must be `float64`.
- Enums must be encoded as `string` (the item name, e.g. `"red"`).
- Fixed-size arrays (e.g., `[2,2]`) must match the declared dimensions exactly.

#### 2D arrays (`arr<T>[R,C]`)

For a fixed-size 2D array column like:

```text
Matrix:arr<int>[2,3]
```

Provide the value as either `[][]any` or as `[]any` where each element is a `[]any` row:

```go
matrix := [][]any{
    {int64(1), int64(2), int64(3)},
    {int64(4), int64(5), int64(6)},
}

row := []any{matrix}
```

Common pitfalls (these return an error from `WriteRow`):

- Dimension mismatch (wrong number of rows or columns)
- Non-rectangular arrays (row lengths differ)

---

## 5. Error handling patterns (Go)

Common sentinel errors are in `supr/errors.go` (e.g., `ErrMaxErrors`, `ErrInvalidHeader`, `ErrValidation`).

Per-row validation errors returned by the streaming validator are typically `*supr.ValidationError`.

Fatal streaming errors surface from `Err()` (validator/decoder). Use `errors.Is` / `errors.As` as appropriate.
