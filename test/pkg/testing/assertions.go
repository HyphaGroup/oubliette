package testing

import (
	"fmt"
	"strings"
)

// Assertions provides helper methods for test assertions
type Assertions struct {
	ctx   *TestContext
	Count int
}

// NewAssertions creates a new Assertions helper
func NewAssertions(ctx *TestContext) *Assertions {
	return &Assertions{
		ctx:   ctx,
		Count: 0,
	}
}

// AssertTrue asserts that the condition is true
func (a *Assertions) AssertTrue(condition bool, message string) {
	a.Count++
	if !condition {
		a.ctx.Log("❌ Assertion failed: %s", message)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertFalse asserts that the condition is false
func (a *Assertions) AssertFalse(condition bool, message string) {
	a.Count++
	if condition {
		a.ctx.Log("❌ Assertion failed: %s", message)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertEqual asserts that two values are equal
func (a *Assertions) AssertEqual(expected, actual interface{}, message string) {
	a.Count++
	if expected != actual {
		a.ctx.Log("❌ Assertion failed: %s (expected: %v, actual: %v)", message, expected, actual)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertNotEqual asserts that two values are not equal
func (a *Assertions) AssertNotEqual(expected, actual interface{}, message string) {
	a.Count++
	if expected == actual {
		a.ctx.Log("❌ Assertion failed: %s (both values: %v)", message, expected)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertContains asserts that a string contains a substring
func (a *Assertions) AssertContains(haystack, needle, message string) {
	a.Count++
	if !strings.Contains(haystack, needle) {
		a.ctx.Log("❌ Assertion failed: %s (haystack does not contain '%s')", message, needle)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertNotContains asserts that a string does not contain a substring
func (a *Assertions) AssertNotContains(haystack, needle, message string) {
	a.Count++
	if strings.Contains(haystack, needle) {
		a.ctx.Log("❌ Assertion failed: %s (haystack contains '%s')", message, needle)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertNotEmpty asserts that a string is not empty
func (a *Assertions) AssertNotEmpty(value, message string) {
	a.Count++
	if value == "" {
		a.ctx.Log("❌ Assertion failed: %s (expected non-empty string)", message)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertNoError asserts that an error is nil
func (a *Assertions) AssertNoError(err error, message string) {
	a.Count++
	if err != nil {
		a.ctx.Log("❌ Assertion failed: %s (error: %v)", message, err)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertError asserts that an error is not nil
func (a *Assertions) AssertError(err error, message string) {
	a.Count++
	if err == nil {
		a.ctx.Log("❌ Assertion failed: %s (expected error but got nil)", message)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s (error: %v)", message, err)
	}
}

// AssertErrorContains asserts that an error contains a specific message
func (a *Assertions) AssertErrorContains(err error, expectedMsg, message string) {
	a.Count++
	if err == nil {
		a.ctx.Log("❌ Assertion failed: %s (expected error but got nil)", message)
		a.ctx.MarkFailed()
		return
	}

	if !strings.Contains(err.Error(), expectedMsg) {
		a.ctx.Log("❌ Assertion failed: %s (error '%v' does not contain '%s')", message, err, expectedMsg)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertGreaterThan asserts that actual > expected (for integers)
func (a *Assertions) AssertGreaterThan(actual, expected int, message string) {
	a.Count++
	if actual <= expected {
		a.ctx.Log("❌ Assertion failed: %s (actual: %d, expected > %d)", message, actual, expected)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertGreaterOrEqual asserts that actual >= expected (for integers)
func (a *Assertions) AssertGreaterOrEqual(actual, expected int, message string) {
	a.Count++
	if actual < expected {
		a.ctx.Log("❌ Assertion failed: %s (actual: %d, expected >= %d)", message, actual, expected)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertLessThan asserts that actual < expected (for integers)
func (a *Assertions) AssertLessThan(actual, expected int, message string) {
	a.Count++
	if actual >= expected {
		a.ctx.Log("❌ Assertion failed: %s (actual: %d, expected < %d)", message, actual, expected)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// AssertLengthEqual asserts that a slice/array/string has the expected length
func (a *Assertions) AssertLengthEqual(actual interface{}, expected int, message string) {
	a.Count++
	var length int

	switch v := actual.(type) {
	case string:
		length = len(v)
	case []string:
		length = len(v)
	case []interface{}:
		length = len(v)
	default:
		a.ctx.Log("❌ Assertion failed: %s (unsupported type for length check)", message)
		a.ctx.MarkFailed()
		return
	}

	if length != expected {
		a.ctx.Log("❌ Assertion failed: %s (actual length: %d, expected: %d)", message, length, expected)
		a.ctx.MarkFailed()
	} else {
		a.ctx.Log("✓ Assertion passed: %s", message)
	}
}

// Fail explicitly fails the test with a message
func (a *Assertions) Fail(message string) {
	a.ctx.Log("❌ Test failed: %s", message)
	a.ctx.MarkFailed()
}

// FailWithError explicitly fails the test with an error
func (a *Assertions) FailWithError(err error, message string) {
	a.ctx.Log("❌ Test failed: %s (error: %v)", message, err)
	a.ctx.MarkFailed()
}

// LogInfo logs an informational message (not an assertion)
func (a *Assertions) LogInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	a.ctx.Log("ℹ️  %s", msg)
}
