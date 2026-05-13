// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package dfa

// DFA package provides validation functions for scalar types.
// Most scalar types are validated using pure functions (stateless).
// Enum types require EnumValidator objects (stateful - contain tries of allowed values).
