# Test Utils Package

This package contains shared testing utilities used across the qwin project's test suite.

## Purpose

The `testutils` package helps eliminate code duplication in test files by providing common testing utilities and helper functions that are used across multiple test packages.

## Functions

### `FieldsToMap(t TestingT, fields []any) map[string]any`

Safely converts a slice of alternating key-value pairs to a map. This is commonly used in logging tests to validate structured log fields.

**Parameters:**

- `t`: A testing interface that provides `Errorf` method (typically `*testing.T`)
- `fields`: A slice of alternating keys (strings) and values (any type)

**Returns:**

- A map with string keys and any values

**Features:**

- Performs safe type assertions for keys (must be strings)
- Handles malformed entries gracefully by logging errors via the testing interface
- Skips invalid entries and continues processing valid ones

**Example Usage:**

```go
import "qwin/internal/testutils"

func TestSomething(t *testing.T) {
    fields := []any{"name", "John", "age", 30, "active", true}
    fieldsMap := testutils.FieldsToMap(t, fields)

    // fieldsMap now contains: {"name": "John", "age": 30, "active": true}

    if fieldsMap["name"] != "John" {
        t.Errorf("Expected name to be John, got %v", fieldsMap["name"])
    }
}
```

## Adding New Utilities

When adding new test utilities to this package:

1. Ensure the utility is used in at least 2 different test packages
2. Make functions generic and reusable
3. Use interfaces instead of concrete types when possible (like `TestingT` instead of `*testing.T`)
4. Add comprehensive tests for the utility functions
5. Document the function's purpose, parameters, and usage examples
