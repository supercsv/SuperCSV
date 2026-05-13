# Changelog

All notable changes to SuperCSV will be documented in this file.

## v1.0.0 — Initial Public Release

### Format

- 15 scalar types: `int`, `float`, `decimal`, `bool`, `string`, `bytes<hex>`, `bytes<b64>`, `date`, `time`, `datetime`, `datetimetz`, `timestamp`, `duration`, `timezone`, `uuid`
- Containers: `list<T>` and `arr<T>` (1D and 2D), fixed-size and dynamic
- Enums with name-only and value=name forms
- Comments and metadata (line-level and inline)
- Multi-line headers and multi-line data rows
- Multi-line container values
- Null literal (`_`), quoted and unquoted strings
- Version declaration (`((SuperCSV v1.0))`)
- Type aliases (default, small, tiny)

### Go SDK (`supr` package)

- Streaming validator, decoder, and encoder (constant memory)
- File-level convenience APIs (`ValidateFile`, `DecodeFile`, `EncodeFile`)
- 64 structured error codes with human-readable messages
- Configurable options: error limits, field size limits, strict version
- 3,800+ tests

### CLI

- `supercsv-validate`: standalone validation tool (SuperCSV error output)
