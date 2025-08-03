// Package testutil provides common testing utilities and helpers
// to reduce code duplication across test files.
package testutil

import (
	"strings"
	"testing"
)

// ErrorChecker provides utilities for testing error conditions
type ErrorChecker struct {
	t *testing.T
}

// NewErrorChecker creates a new ErrorChecker instance
func NewErrorChecker(t *testing.T) *ErrorChecker {
	return &ErrorChecker{t: t}
}

// ExpectError asserts that an error should occur
func (ec *ErrorChecker) ExpectError(err error, context string) {
	ec.t.Helper()
	if err == nil {
		ec.t.Errorf("%s: expected error but got none", context)
	}
}

// ExpectNoError asserts that no error should occur
func (ec *ErrorChecker) ExpectNoError(err error, context string) {
	ec.t.Helper()
	if err != nil {
		ec.t.Errorf("%s: expected no error but got: %v", context, err)
	}
}

// ExpectErrorContaining asserts that an error should occur and contain specific text
func (ec *ErrorChecker) ExpectErrorContaining(err error, expectedText, context string) {
	ec.t.Helper()
	if err == nil {
		ec.t.Errorf("%s: expected error containing '%s' but got none", context, expectedText)
	} else if !strings.Contains(err.Error(), expectedText) {
		ec.t.Errorf("%s: expected error containing '%s', got: %v", context, expectedText, err)
	}
}

// AssertionHelper provides utilities for common value assertions
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper creates a new AssertionHelper instance
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// Equal asserts that two values are equal
func (ah *AssertionHelper) Equal(expected, actual interface{}, context string) {
	ah.t.Helper()
	if expected != actual {
		ah.t.Errorf("%s: expected %v, got %v", context, expected, actual)
	}
}

// NotEqual asserts that two values are not equal
func (ah *AssertionHelper) NotEqual(expected, actual interface{}, context string) {
	ah.t.Helper()
	if expected == actual {
		ah.t.Errorf("%s: expected values to be different, but both were %v", context, expected)
	}
}

// Contains asserts that a string contains a substring
func (ah *AssertionHelper) Contains(haystack, needle, context string) {
	ah.t.Helper()
	if !strings.Contains(haystack, needle) {
		ah.t.Errorf("%s: expected string to contain '%s', got: %s", context, needle, haystack)
	}
}

// NotEmpty asserts that a string is not empty
func (ah *AssertionHelper) NotEmpty(value, context string) {
	ah.t.Helper()
	if value == "" {
		ah.t.Errorf("%s: expected non-empty string", context)
	}
}

// True asserts that a condition is true
func (ah *AssertionHelper) True(condition bool, context string) {
	ah.t.Helper()
	if !condition {
		ah.t.Errorf("%s: expected condition to be true", context)
	}
}

// False asserts that a condition is false
func (ah *AssertionHelper) False(condition bool, context string) {
	ah.t.Helper()
	if condition {
		ah.t.Errorf("%s: expected condition to be false", context)
	}
}

// TestResult represents the result of an operation that should be tested
type TestResult struct {
	Value interface{}
	Error error
}

// NewTestResult creates a TestResult from an operation
func NewTestResult(value interface{}, err error) TestResult {
	return TestResult{Value: value, Error: err}
}

// AssertSuccess asserts that the operation succeeded
func (tr TestResult) AssertSuccess(t *testing.T, context string) interface{} {
	t.Helper()
	if tr.Error != nil {
		t.Fatalf("%s: expected success but got error: %v", context, tr.Error)
	}
	return tr.Value
}

// AssertError asserts that the operation failed
func (tr TestResult) AssertError(t *testing.T, context string) error {
	t.Helper()
	if tr.Error == nil {
		t.Fatalf("%s: expected error but operation succeeded", context)
	}
	return tr.Error
}

// AssertErrorContaining asserts that the operation failed with specific error text
func (tr TestResult) AssertErrorContaining(t *testing.T, expectedText, context string) error {
	t.Helper()
	if tr.Error == nil {
		t.Fatalf("%s: expected error containing '%s' but operation succeeded", context, expectedText)
	}
	if !strings.Contains(tr.Error.Error(), expectedText) {
		t.Fatalf("%s: expected error containing '%s', got: %v", context, expectedText, tr.Error)
	}
	return tr.Error
}

// FieldComparer provides utilities for comparing struct fields
type FieldComparer struct {
	t       *testing.T
	context string
}

// NewFieldComparer creates a new FieldComparer instance
func NewFieldComparer(t *testing.T, context string) *FieldComparer {
	return &FieldComparer{t: t, context: context}
}

// CompareString compares string fields
func (fc *FieldComparer) CompareString(fieldName, expected, actual string) {
	fc.t.Helper()
	if expected != actual {
		fc.t.Errorf("%s.%s failed: got %v, want %v", fc.context, fieldName, actual, expected)
	}
}

// CompareInt compares integer fields
func (fc *FieldComparer) CompareInt(fieldName string, expected, actual int) {
	fc.t.Helper()
	if expected != actual {
		fc.t.Errorf("%s.%s failed: got %v, want %v", fc.context, fieldName, actual, expected)
	}
}

// CompareBool compares boolean fields
func (fc *FieldComparer) CompareBool(fieldName string, expected, actual bool) {
	fc.t.Helper()
	if expected != actual {
		fc.t.Errorf("%s.%s failed: got %v, want %v", fc.context, fieldName, actual, expected)
	}
}

// CompareIntPtr compares *int fields (handles nil cases)
func (fc *FieldComparer) CompareIntPtr(fieldName string, expected, actual *int) {
	fc.t.Helper()
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		fc.t.Errorf("%s.%s failed: got %v, want %v", fc.context, fieldName, actual, expected)
		return
	}
	if *expected != *actual {
		fc.t.Errorf("%s.%s failed: got %v, want %v", fc.context, fieldName, *actual, *expected)
	}
}

// CompareStringSlice compares []string fields
func (fc *FieldComparer) CompareStringSlice(fieldName string, expected, actual []string) {
	fc.t.Helper()
	if len(expected) != len(actual) {
		fc.t.Errorf("%s.%s failed: got length %d, want length %d", fc.context, fieldName, len(actual), len(expected))
		return
	}
	for i, exp := range expected {
		if i >= len(actual) || exp != actual[i] {
			fc.t.Errorf("%s.%s[%d] failed: got %v, want %v", fc.context, fieldName, i, actual, expected)
			return
		}
	}
}

// CompareIntSlice compares []int fields
func (fc *FieldComparer) CompareIntSlice(fieldName string, expected, actual []int) {
	fc.t.Helper()
	if len(expected) != len(actual) {
		fc.t.Errorf("%s.%s failed: got length %d, want length %d", fc.context, fieldName, len(actual), len(expected))
		return
	}
	for i, exp := range expected {
		if i >= len(actual) || exp != actual[i] {
			fc.t.Errorf("%s.%s[%d] failed: got %v, want %v", fc.context, fieldName, i, actual, expected)
			return
		}
	}
}

// TableTestRunner provides utilities for running table-driven tests
type TableTestRunner[T any] struct {
	t *testing.T
}

// NewTableTestRunner creates a new TableTestRunner instance
func NewTableTestRunner[T any](t *testing.T) *TableTestRunner[T] {
	return &TableTestRunner[T]{t: t}
}

// Run runs a table test with the provided test cases
func (ttr *TableTestRunner[T]) Run(testCases []T, nameFunc func(T) string, testFunc func(*testing.T, T)) {
	ttr.t.Helper()
	for _, tc := range testCases {
		name := nameFunc(tc)
		ttr.t.Run(name, func(t *testing.T) {
			testFunc(t, tc)
		})
	}
}