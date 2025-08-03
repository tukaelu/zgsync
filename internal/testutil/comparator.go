package testutil

import (
	"reflect"
	"testing"
)

// StructComparator provides utilities for comparing complex structs
type StructComparator struct {
	t *testing.T
}

// NewStructComparator creates a new StructComparator instance
func NewStructComparator(t *testing.T) *StructComparator {
	return &StructComparator{t: t}
}

// CompareStructs performs a deep comparison of two structs
func (sc *StructComparator) CompareStructs(expected, actual interface{}, context string) {
	sc.t.Helper()
	
	if !reflect.DeepEqual(expected, actual) {
		sc.t.Errorf("%s: structs are not equal\nExpected: %+v\nActual: %+v", context, expected, actual)
	}
}

// CompareStructFields compares specific fields of structs using reflection
func (sc *StructComparator) CompareStructFields(expected, actual interface{}, fields []string, context string) {
	sc.t.Helper()
	
	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	
	// Handle pointers
	if expectedVal.Kind() == reflect.Ptr {
		expectedVal = expectedVal.Elem()
	}
	if actualVal.Kind() == reflect.Ptr {
		actualVal = actualVal.Elem()
	}
	
	for _, fieldName := range fields {
		expectedField := expectedVal.FieldByName(fieldName)
		actualField := actualVal.FieldByName(fieldName)
		
		if !expectedField.IsValid() {
			sc.t.Errorf("%s: field %s not found in expected struct", context, fieldName)
			continue
		}
		if !actualField.IsValid() {
			sc.t.Errorf("%s: field %s not found in actual struct", context, fieldName)
			continue
		}
		
		if !reflect.DeepEqual(expectedField.Interface(), actualField.Interface()) {
			sc.t.Errorf("%s.%s: expected %v, got %v", context, fieldName, expectedField.Interface(), actualField.Interface())
		}
	}
}

// CompareSlices compares two slices with detailed error reporting
func (sc *StructComparator) CompareSlices(expected, actual interface{}, context string) {
	sc.t.Helper()
	
	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	
	if expectedVal.Kind() != reflect.Slice || actualVal.Kind() != reflect.Slice {
		sc.t.Errorf("%s: both values must be slices", context)
		return
	}
	
	if expectedVal.Len() != actualVal.Len() {
		sc.t.Errorf("%s: slice length mismatch, expected %d, got %d", context, expectedVal.Len(), actualVal.Len())
		return
	}
	
	for i := 0; i < expectedVal.Len(); i++ {
		expectedItem := expectedVal.Index(i).Interface()
		actualItem := actualVal.Index(i).Interface()
		
		if !reflect.DeepEqual(expectedItem, actualItem) {
			sc.t.Errorf("%s[%d]: expected %v, got %v", context, i, expectedItem, actualItem)
		}
	}
}

// ComparePointers compares pointer values, handling nil cases properly
func (sc *StructComparator) ComparePointers(expected, actual interface{}, context string) {
	sc.t.Helper()
	
	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	
	// Both nil
	if !expectedVal.IsValid() && !actualVal.IsValid() {
		return
	}
	
	// One nil, one not
	if (!expectedVal.IsValid() && actualVal.IsValid()) || (expectedVal.IsValid() && !actualVal.IsValid()) {
		sc.t.Errorf("%s: pointer nil mismatch, expected %v, got %v", context, expected, actual)
		return
	}
	
	// Both pointers
	if expectedVal.Kind() == reflect.Ptr && actualVal.Kind() == reflect.Ptr {
		if expectedVal.IsNil() && actualVal.IsNil() {
			return
		}
		if expectedVal.IsNil() || actualVal.IsNil() {
			sc.t.Errorf("%s: pointer nil mismatch, expected %v, got %v", context, expected, actual)
			return
		}
		
		// Compare dereferenced values
		if !reflect.DeepEqual(expectedVal.Elem().Interface(), actualVal.Elem().Interface()) {
			sc.t.Errorf("%s: pointer values differ, expected %v, got %v", context, expectedVal.Elem().Interface(), actualVal.Elem().Interface())
		}
		return
	}
	
	// Direct comparison for non-pointers
	if !reflect.DeepEqual(expected, actual) {
		sc.t.Errorf("%s: values differ, expected %v, got %v", context, expected, actual)
	}
}

// MapComparator provides utilities for comparing maps
type MapComparator struct {
	t *testing.T
}

// NewMapComparator creates a new MapComparator instance
func NewMapComparator(t *testing.T) *MapComparator {
	return &MapComparator{t: t}
}

// CompareMaps compares two maps with detailed error reporting
func (mc *MapComparator) CompareMaps(expected, actual interface{}, context string) {
	mc.t.Helper()
	
	expectedVal := reflect.ValueOf(expected)
	actualVal := reflect.ValueOf(actual)
	
	if expectedVal.Kind() != reflect.Map || actualVal.Kind() != reflect.Map {
		mc.t.Errorf("%s: both values must be maps", context)
		return
	}
	
	expectedKeys := expectedVal.MapKeys()
	actualKeys := actualVal.MapKeys()
	
	if len(expectedKeys) != len(actualKeys) {
		mc.t.Errorf("%s: map size mismatch, expected %d keys, got %d keys", context, len(expectedKeys), len(actualKeys))
	}
	
	// Check all expected keys exist and have correct values
	for _, key := range expectedKeys {
		expectedValue := expectedVal.MapIndex(key)
		actualValue := actualVal.MapIndex(key)
		
		if !actualValue.IsValid() {
			mc.t.Errorf("%s: missing key %v", context, key.Interface())
			continue
		}
		
		if !reflect.DeepEqual(expectedValue.Interface(), actualValue.Interface()) {
			mc.t.Errorf("%s[%v]: expected %v, got %v", context, key.Interface(), expectedValue.Interface(), actualValue.Interface())
		}
	}
	
	// Check for unexpected keys
	for _, key := range actualKeys {
		expectedValue := expectedVal.MapIndex(key)
		if !expectedValue.IsValid() {
			mc.t.Errorf("%s: unexpected key %v", context, key.Interface())
		}
	}
}

// ErrorComparator provides utilities for comparing errors
type ErrorComparator struct {
	t *testing.T
}

// NewErrorComparator creates a new ErrorComparator instance
func NewErrorComparator(t *testing.T) *ErrorComparator {
	return &ErrorComparator{t: t}
}

// CompareErrors compares two errors, handling nil cases
func (ec *ErrorComparator) CompareErrors(expected, actual error, context string) {
	ec.t.Helper()
	
	if expected == nil && actual == nil {
		return
	}
	
	if expected == nil && actual != nil {
		ec.t.Errorf("%s: expected no error, got %v", context, actual)
		return
	}
	
	if expected != nil && actual == nil {
		ec.t.Errorf("%s: expected error %v, got none", context, expected)
		return
	}
	
	if expected.Error() != actual.Error() {
		ec.t.Errorf("%s: error message mismatch, expected '%s', got '%s'", context, expected.Error(), actual.Error())
	}
}