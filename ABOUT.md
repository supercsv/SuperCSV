# About SuperCSV

## [supercsv.com](https://www.supercsv.com)

SuperCSV is a format specification: a typed, self‑describing everyday data format defined by a stable, versioned spec.  
The full specification is included in this repository and is licensed under the Apache 2.0 open‑source license.

This repository also contains the canonical reference implementation of SuperCSV v1.0.

If you are new to the format, the following documents provide the formal definition:

- [**Quick Reference**](internal/v1_0/spec/supercsv-quick-reference-v1.0.md) — concise overview of structure, types, and rules  
- [**Specification**](internal/v1_0/spec/supercsv-spec-v1.0.md) — complete format definition  
- [**Type Table**](internal/v1_0/spec/supercsv-type-table-v1.0.md) — all scalar types and their constraints  
- [**Data Model**](internal/v1_0/spec/supercsv-model-v1.0.md) — processing model and structural semantics  

This document explains the engineering rationale behind the reference implementation.  
It is intended for anyone building a SuperCSV implementation in another language.

If you need to use the Go SDK for SuperCSV, start with the [End-to-End Demo](docs/end-to-end-demo.md) and the [Encode, Decode & Validate Guide](docs/supercsv-encode-decode-validate-guide.md), then come back to this document for more details.

For the motivation and design philosophy behind the SuperCSV format, see [supercsv.com/about](https://www.supercsv.com/about.html).

# About the Reference Implementation

This repository contains the **reference implementation** — a Go library and CLI tool that implements the SuperCSV v1.0 spec exactly.

This document explains the design decisions behind the reference implementation. It is written for anyone building their own SuperCSV implementation in another language.

## Why Go

Go was chosen as the implementation language for its neutrality:

- Simple, readable syntax that translates naturally to other languages
- No advanced type system features that would obscure the logic
- Explicit control flow — no hidden allocations, no hidden behaviour
- First-class support for byte-level processing and streaming I/O
- Language stability — Go evolves slowly and avoids breaking changes

Go's constraints are a feature: they force clarity and prevent cleverness.

The goal is that a developer reading this Go code can understand the spec's intent and translate it to any other language without needing Go-specific knowledge.

## Intentionally Minimal

The reference implementation is deliberately written using minimal Go-specific libraries and language features. This is not a limitation — it is a design constraint that ensures portability and faithfulness to the spec.

Every language specific feature avoided is a feature that other implementations do not need to replicate. The simpler the reference, the more accurately it can be ported.

The goal is not to showcase Go. It is to provide a clear, unambiguous reference for the SuperCSV format.

## Architecture Mirrors the Spec

The implementation follows a strict processing hierarchy that maps directly to the specification:

1. **File Validator** — entry point, file-level structural checks
2. **CSV Stream Parser** — byte-level row/field splitting, zero allocation
3. **Row Validator** — field count, null markers, structural rules
4. **Container Validator** — list and array syntax, size constraints
5. **Scalar Validator DFAs** — one deterministic finite automaton per type, byte-by-byte

Decoding and encoding follow the same layered structure. Validation, decoding, and encoding are strictly separated — they never share logic or state.

This hierarchy is not an implementation detail. It reflects how the spec defines processing, and other implementations should follow the same separation.

## Zero-Allocation Hot Paths

All validation operates on raw byte slices with zero heap allocations in the critical path. Scalar validators are implemented as DFAs that process one byte at a time. This matters because:

- It forces the implementation to be deterministic and portable
- It proves the spec can be implemented without runtime overhead
- It sets a performance baseline other implementations can target

## Round-Trip Safety

The implementation guarantees:

```
encode(decode(input)) == canonical(input)
```

No lossy transformations. No normalisation unless the spec defines it. Data that passes through the reference implementation comes out unchanged.

## What This Means for Other Implementations

If you are building a SuperCSV implementation in another language:

- **Follow the spec, not the Go code.** The spec is the source of truth. This implementation is a faithful reference, but the spec governs.
- **Maintain the processing hierarchy.** Keep validation, decoding, and encoding as separate concerns. Do not merge them.
- **Keep it simple.** Avoid clever abstractions. The format is designed to be implemented with basic language features — loops, byte arrays, and switch statements.
- **Test against the spec examples.** The canonical examples in the spec define correct behaviour. Your implementation should produce identical results.
- **Do not infer or normalise.** SuperCSV is explicit by design. An implementation must not guess types, infer structure, or silently transform values.

## Further Reading

- [SuperCSV v1.0 Specification](internal/v1_0/spec/supercsv-spec-v1.0.md)
- [Type Table](internal/v1_0/spec/supercsv-type-table-v1.0.md)
- [Data Model](internal/v1_0/spec/supercsv-model-v1.0.md)
- [Formal Grammar](internal/v1_0/spec/grammar.ebnf)
