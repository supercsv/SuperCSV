# supercsv-validate

Validates `.supr` (SuperCSV) files against the SuperCSV v1.0 specification.

## Installation

```bash
go install github.com/supercsv/supercsv/cmd/supercsv-validate@latest
```

Or build from source:

```bash
cd cmd/supercsv-validate
go build
```

## Usage

```bash
supercsv-validate [flags] file.supr
```

### Exit Codes

- `0` — File is valid SuperCSV v1.0
- `1` — Validation errors found
- `2` — I/O error or invalid arguments

## Flags

| Flag | Description |
|------|-------------|
| `-q, --quiet` | Suppress per-error output; keep the final summary and exit code |
| `--max-errors N` | Stop after N errors (default: unlimited) |
| `--json` | Emit newline-delimited JSON errors instead of SUPR format |
| `--strict-version VERSION` | Require file to declare specific version (e.g., `v1.0`) |
| `--version` | Print version and exit |

## Output Formats

### SUPR (default)

Machine-readable SuperCSV format, suitable for processing with SuperCSV tools:

```bash
$ supercsv-validate data.supr
Line:int,ErrorSection:string,ErrorMsg:string
5,Age,"invalid int: strconv.ParseInt: parsing ""twenty"": invalid syntax"
12,Score,"invalid float: strconv.ParseFloat: parsing ""N/A"": invalid syntax"
18,"","expected 4 columns, got 3"
```

### JSON (`--json`)

Newline-delimited JSON (NDJSON) with structured error objects:

```bash
$ supercsv-validate --json data.supr
{"line":5,"errorSection":"Age","errorMsg":"invalid int: strconv.ParseInt: parsing \"twenty\": invalid syntax"}
{"line":12,"errorSection":"Score","errorMsg":"invalid float: strconv.ParseFloat: parsing \"N/A\": invalid syntax"}
{"line":18,"errorSection":"","errorMsg":"expected 4 columns, got 3"}
```

## Examples

### Validate a file

```bash
$ supercsv-validate data.supr
```

Success (exit code 0):
```
Line:int,ErrorSection:string,ErrorMsg:string
data.supr: OK
```

In a terminal these lines appear together, but they are written to different streams:

- `Line:int,ErrorSection:string,ErrorMsg:string` is written to stdout
- `data.supr: OK` is written to stderr

Validation errors (exit code 1):
```
Line:int,ErrorSection:string,ErrorMsg:string
5,Age,"invalid int: strconv.ParseInt: parsing ""abc"": invalid syntax"
```

### Quiet mode (summary plus exit code)

```bash
$ supercsv-validate --quiet data.supr
data.supr: OK
$ echo $?
0
```

`--quiet` suppresses the per-error diagnostic rows, but the final summary line is still written to stderr.

### Stop after 10 errors

```bash
$ supercsv-validate --max-errors 10 large-file.supr
```

### Require version declaration

```bash
$ supercsv-validate --strict-version v1.0 data.supr
```

Requires file to start with:
```
((SuperCSV v1.0))
```

### JSON output for CI/tooling

```bash
$ supercsv-validate --json data.supr | jq '.errorMsg' | sort | uniq -c
```

## Validation Rules

The validator enforces all SuperCSV v1.0 specification requirements. For a concise summary of types, containers, and file structure rules, see the [Quick Reference](../../internal/v1_0/spec/supercsv-quick-reference-v1.0.md).

## Error Codes

For error code details and examples, see:
- **[Error Codes Catalog](../../docs/validator/error-codes.md)** — Complete list organized by category
- **[Error Examples](../../docs/validator/error-examples.md)** — Teaching examples for each error category

## Examples

### Valid File

```csv
((SuperCSV v1.0))
# Example: People data
Name:string,Age:int,Score:float,Active:bool
Alice,30,95.5,true
Bob,25,87.3,1
Charlie,35,92.1,false
```

```bash
$ supercsv-validate people.supr
Line:int,ErrorSection:string,ErrorMsg:string
people.supr: OK
$ echo $?
0
```

### Invalid File

```csv
Name:string,Age:int,Score:float
Alice,thirty,95.5
Bob,25,N/A
```

```bash
$ supercsv-validate bad-data.supr
Line:int,ErrorSection:string,ErrorMsg:string
2,Age,"invalid int: strconv.ParseInt: parsing ""thirty"": invalid syntax"
3,Score,"invalid float: strconv.ParseFloat: parsing ""N/A"": invalid syntax"
$ echo $?
1
```

## See Also

- **[Error Codes Catalog](../../docs/validator/error-codes.md)** — Complete list of error codes by category
- **[Error Examples](../../docs/validator/error-examples.md)** — Teaching examples for each error category
