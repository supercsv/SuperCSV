# SuperCSV - A new **standard** for data

- A typed, self‑describing, everyday data format.
- Read by humans, parsed by machines.  
- Streamable, round‑trip safe, and definition‑first.

Every `.supr` file contains its own structure, types, and rules.
No external schemas, no guessing, no surprises.

Learn more at [supercsv.com](https://www.supercsv.com/)

## Why SuperCSV Exists

Most real‑world data formats rely on hidden context.

- CSV requires external knowledge of types.
- JSON requires external knowledge of structure.
- Pipelines rely on ad‑hoc mapping code to make sense of values.

SuperCSV avoids all of this.
- The structure and types are explicit.
- The rules are obvious.

A SuperCSV file stands on its own.

For the full motivation and design philosophy, see: [supercsv.com/about](https://www.supercsv.com/about.html)

## Minimal Example

Files use the `.supr` extension.

```text
((SuperCSV v1.0))

Name:string, Age:int, Active:bool,  MathType:string,   Value:float

Alice      ,      30,       true,          "pi (π)",    3.141593
Bob        ,      25,      false,  square root of 2,    1.414214
Charlie    ,      41,       true,           sin 60°,    0.866025
Diana      ,      19,       true,                 φ,    1.618034
Evan       ,      33,      false,                 e,    2.718282
```

More examples at [supercsv.com](https://www.supercsv.com/)

## SuperCSV v1.0 — Format at a Glance

- SuperCSV is a self-contained format
- Structured, typed header definitions
- 15 well-defined scalar types
- Enums, lists and arrays
- Human‑readable, machine‑parseable syntax

## Reference Implementation

Go was chosen as the implementation language for its neutrality:

- Coded for clarity, performance, and portability
- Zero‑allocation validation hot paths
- Deterministic DFA‑based scalar parsing
- Spec‑aligned validation, decoding, and encoding pipeline
- Guaranteed, typed round‑trip encoding

## Quick Start

### Install the CLI validator

```
go install github.com/supercsv/supercsv/cmd/supercsv-validate@latest
```

### Validate a file

```
supercsv-validate people.supr
```

### Use the Go SDK

```go
import "github.com/supercsv/supercsv/supr"
```

```go
// Validate
header, rowErrs, fatalErr := supr.ValidateFile(f)

// Decode
d := supr.NewDecoder(f)
for d.Next() {
    row := d.Row() // []any with typed values
}

// Encode
err := supr.EncodeFile(f, &supr.File{Header: header, Rows: rows})
```

For the full API walkthrough — validate, decode, encode, options, type mapping, and a complete round-trip example — see the [SDK Guide](docs/supercsv-encode-decode-validate-guide.md).

## Recommended Reading Order

The repository docs are the source of truth for the format and the Go SDK.

1. [ABOUT.md](ABOUT.md) introduction and overview.
2. [Quick Reference](internal/v1_0/spec/supercsv-quick-reference-v1.0.md)  the format rules in compact form.
3. [End-to-End Demo](docs/end-to-end-demo.md) shows generate data in Go, write a `.supr` file, validate it, and decode it again.
4. [Encode, Decode & Validate Guide](docs/supercsv-encode-decode-validate-guide.md) for the full Go SDK surface, decode modes, options, and type mapping details.
5. [Specification](internal/v1_0/spec/supercsv-spec-v1.0.md), [Type Table](internal/v1_0/spec/supercsv-type-table-v1.0.md), and [Data Model](internal/v1_0/spec/supercsv-model-v1.0.md) when you need the full format details.

## Specification

The canonical SuperCSV v1.0 specification lives in this repository:

- [supercsv-spec-v1.0.md](internal/v1_0/spec/supercsv-spec-v1.0.md) — format rules, syntax, validation
- [supercsv-type-table-v1.0.md](internal/v1_0/spec/supercsv-type-table-v1.0.md) — all 15 scalar types
- [supercsv-model-v1.0.md](internal/v1_0/spec/supercsv-model-v1.0.md) — data model and processing architecture
- [grammar.ebnf](internal/v1_0/spec/grammar.ebnf) — formal grammar

The spec is also available on the website: [supercsv.com/spec](https://www.supercsv.com/spec.html)

The format is versioned and designed for backwards compatibility.

## Documentation

- [End-to-End Demo](docs/end-to-end-demo.md) — create a `.supr` file in Go, validate it with the CLI, and decode it in multiple ways
- [Encode, Decode & Validate Guide](docs/supercsv-encode-decode-validate-guide.md) — full Go SDK walkthrough, including Go type usage for encoding and decoding each type category
- [Error Codes](docs/validator/error-codes.md) — all 64 validation error codes
- [Error Examples](docs/validator/error-examples.md) — example errors with context
- [About the Reference Implementation](ABOUT.md) — design decisions, architecture, and guidance for other implementations

## Testing

```bash
# Run all tests
go test ./...

# Run public API tests only
go test ./supr/...
```

## Learn More

- [Website](https://www.supercsv.com)
- [About SuperCSV](https://www.supercsv.com/about.html)
- [Specification](https://www.supercsv.com/spec.html)
- [Type Table](https://www.supercsv.com/types.html)
- [Encode](https://www.supercsv.com/encode.html) · [Decode](https://www.supercsv.com/decode.html) · [Validate](https://www.supercsv.com/validate.html)
- [SuperCSV vs The World](https://www.supercsv.com/vs-the-world.html)

## Licence

[Apache License 2.0](LICENSE)

---

SuperCSV was crafted in Aotearoa New Zealand with aroha and whakapau kaha.
