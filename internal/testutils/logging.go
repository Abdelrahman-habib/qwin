package testutils

// TestingT is a minimal interface that matches the methods we need from testing.T
type TestingT interface {
	Errorf(format string, args ...any)
}

// FieldsToMap safely converts a slice of alternating key-value pairs to a map.
// It performs safe type assertions and handles malformed entries gracefully.
// This is commonly used in logging tests to validate structured log fields.
func FieldsToMap(t TestingT, fields []any) map[string]any {
	fieldsMap := make(map[string]any)

	for i := 0; i < len(fields); i += 2 {
		// Ensure we have both key and value
		if i+1 >= len(fields) {
			t.Errorf("Malformed fields slice: missing value for key at index %d", i)
			continue
		}

		// Safe type assertion for the key
		key, ok := fields[i].(string)
		if !ok {
			t.Errorf("Malformed fields slice: key at index %d is not a string, got %T", i, fields[i])
			continue
		}

		// Store the key-value pair
		fieldsMap[key] = fields[i+1]
	}

	return fieldsMap
}
