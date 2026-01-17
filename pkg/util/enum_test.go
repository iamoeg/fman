package util

import (
	"testing"
)

func TestEnumMapToString(t *testing.T) {
	t.Parallel()

	// Define some enum types for testing
	type Status string
	type Priority int
	type Gender string

	const (
		StatusDraft     Status = "DRAFT"
		StatusFinalized Status = "FINALIZED"
		StatusArchived  Status = "ARCHIVED"
	)

	const (
		PriorityLow    Priority = 1
		PriorityMedium Priority = 2
		PriorityHigh   Priority = 3
	)

	const (
		GenderMale   Gender = "MALE"
		GenderFemale Gender = "FEMALE"
	)

	tests := []struct {
		name     string
		input    interface{} // Using interface{} to test different map types
		expected string
	}{
		{
			name: "string enum with multiple values",
			input: map[Status]struct{}{
				StatusDraft:     {},
				StatusFinalized: {},
				StatusArchived:  {},
			},
			expected: "ARCHIVED, DRAFT, FINALIZED", // Alphabetically sorted
		},
		{
			name: "string enum with two values",
			input: map[Gender]struct{}{
				GenderMale:   {},
				GenderFemale: {},
			},
			expected: "FEMALE, MALE", // Alphabetically sorted
		},
		{
			name: "integer enum",
			input: map[Priority]struct{}{
				PriorityLow:    {},
				PriorityMedium: {},
				PriorityHigh:   {},
			},
			expected: "1, 2, 3", // Numerically appears sorted as strings
		},
		{
			name:     "empty map",
			input:    map[Status]struct{}{},
			expected: "",
		},
		{
			name: "single value",
			input: map[Status]struct{}{
				StatusDraft: {},
			},
			expected: "DRAFT",
		},
		{
			name: "raw string map",
			input: map[string]struct{}{
				"alpha":   {},
				"charlie": {},
				"bravo":   {},
			},
			expected: "alpha, bravo, charlie",
		},
		{
			name: "integer map",
			input: map[int]struct{}{
				100: {},
				20:  {},
				3:   {},
			},
			expected: "100, 20, 3", // String sorting, not numeric
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result string

			// Type switch to call the generic function with appropriate type
			switch v := tt.input.(type) {
			case map[Status]struct{}:
				result = EnumMapToString(v)
			case map[Gender]struct{}:
				result = EnumMapToString(v)
			case map[Priority]struct{}:
				result = EnumMapToString(v)
			case map[string]struct{}:
				result = EnumMapToString(v)
			case map[int]struct{}:
				result = EnumMapToString(v)
			default:
				t.Fatalf("unexpected input type: %T", v)
			}

			if result != tt.expected {
				t.Errorf("EnumMapToString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestEnumMapToString_Consistency verifies that multiple calls
// with the same input produce the same output (deterministic behavior)
func TestEnumMapToString_Consistency(t *testing.T) {
	t.Parallel()

	type Status string
	const (
		StatusDraft     Status = "DRAFT"
		StatusFinalized Status = "FINALIZED"
	)

	input := map[Status]struct{}{
		StatusDraft:     {},
		StatusFinalized: {},
	}

	// Call multiple times
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		results[i] = EnumMapToString(input)
	}

	// All results should be identical
	expected := results[0]
	for i, result := range results {
		if result != expected {
			t.Errorf("Call %d produced %q, expected consistent result %q", i, result, expected)
		}
	}
}

// BenchmarkEnumMapToString measures performance with different map sizes
func BenchmarkEnumMapToString(b *testing.B) {
	type Status string

	benchmarks := []struct {
		name string
		size int
	}{
		{"small_5", 5},
		{"medium_20", 20},
		{"large_100", 100},
	}

	for _, bm := range benchmarks {
		// Create map of specified size
		input := make(map[Status]struct{}, bm.size)
		for i := 0; i < bm.size; i++ {
			input[Status("STATUS_"+string(rune('A'+i%26)))] = struct{}{}
		}

		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = EnumMapToString(input)
			}
		})
	}
}
