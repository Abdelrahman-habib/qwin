package testutils

import (
	"fmt"
	"testing"
)

func TestFieldsToMap(t *testing.T) {
	tests := []struct {
		name     string
		fields   []any
		expected map[string]any
	}{
		{
			name:     "empty fields",
			fields:   []any{},
			expected: map[string]any{},
		},
		{
			name:     "single key-value pair",
			fields:   []any{"key", "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "multiple key-value pairs",
			fields:   []any{"name", "John", "age", 30, "active", true},
			expected: map[string]any{"name": "John", "age": 30, "active": true},
		},
		{
			name:     "mixed types",
			fields:   []any{"string", "text", "int", 42, "float", 3.14, "bool", false},
			expected: map[string]any{"string": "text", "int": 42, "float": 3.14, "bool": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FieldsToMap(t, tt.fields)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %q not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("Key %q: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestFieldsToMap_MalformedInput(t *testing.T) {
	// Capture test failures to verify error handling
	var errorMessages []string
	mockT := &mockTestingT{
		errorFunc: func(args ...any) {
			errorMessages = append(errorMessages, args[0].(string))
		},
	}

	t.Run("odd number of fields", func(t *testing.T) {
		errorMessages = nil
		fields := []any{"key1", "value1", "key2"} // missing value for key2

		result := FieldsToMap(mockT, fields)

		if len(result) != 1 {
			t.Errorf("Expected 1 valid pair, got %d", len(result))
		}

		if result["key1"] != "value1" {
			t.Errorf("Expected key1=value1, got key1=%v", result["key1"])
		}

		if len(errorMessages) != 1 {
			t.Errorf("Expected 1 error message, got %d", len(errorMessages))
		}
	})

	t.Run("non-string key", func(t *testing.T) {
		errorMessages = nil
		fields := []any{123, "value", "valid_key", "valid_value"} // 123 is not a string key

		result := FieldsToMap(mockT, fields)

		if len(result) != 1 {
			t.Errorf("Expected 1 valid pair, got %d", len(result))
		}

		if result["valid_key"] != "valid_value" {
			t.Errorf("Expected valid_key=valid_value, got valid_key=%v", result["valid_key"])
		}

		if len(errorMessages) != 1 {
			t.Errorf("Expected 1 error message, got %d", len(errorMessages))
		}
	})
}

// mockTestingT implements the TestingT interface for testing error handling
type mockTestingT struct {
	errorFunc func(args ...any)
}

func (m *mockTestingT) Errorf(format string, args ...any) {
	if m.errorFunc != nil {
		m.errorFunc(fmt.Sprintf(format, args...))
	}
}
