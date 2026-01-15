// Package util provides utility functions for common operations
// across the finmgmt application.
package util

import (
	"fmt"
	"sort"
	"strings"
)

// EnumMapToString converts a map representing a set of enum values
// to a comma-separated string representation.
//
// This function is particularly useful for:
//   - Converting enum validation sets to human-readable error messages
//   - Generating documentation or help text showing valid enum values
//   - Debugging and logging enum constraints
//
// The function accepts a map[T]struct{} where the keys represent the
// valid enum values. Using struct{} as the value type is a Go idiom
// for creating memory-efficient sets.
//
// The returned string contains all enum values in alphabetical order,
// separated by commas and spaces.
//
// Type parameter T must be comparable (supports == and !=), which includes
// all basic types (string, int, etc.) and most custom types used for enums.
//
// Example:
//
//	type Status string
//	const (
//	    StatusDraft     Status = "DRAFT"
//	    StatusFinalized Status = "FINALIZED"
//	)
//
//	validStatuses := map[Status]struct{}{
//	    StatusDraft:     {},
//	    StatusFinalized: {},
//	}
//
//	str := EnumMapToString(validStatuses)
//	// Returns: "DRAFT, FINALIZED"
//
// The function uses fmt.Sprint to convert each enum value to a string,
// so any type with a String() method will use that representation.
func EnumMapToString[T comparable](m map[T]struct{}) string {
	values := make([]string, 0, len(m))
	for k, _ := range m {
		values = append(values, fmt.Sprint(k))
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}
