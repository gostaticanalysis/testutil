package testutil

import "golang.org/x/tools/go/analysis/analysistest"

// ErrorfFunc implements analysistest.Testing.
type ErrorfFunc func(format string, args ...interface{})

var _ analysistest.Testing = ErrorfFunc(nil)

// Errorf implements analysistest.Testing.
func (f ErrorfFunc) Errorf(format string, args ...interface{}) {
	f(format, args...)
}

// Filter calls t.Errorf when the filter returns true.
func Filter(t analysistest.Testing, filter func(format string, args ...interface{}) bool) analysistest.Testing {
	return ErrorfFunc(func(format string, args ...interface{}) {
		if filter(format, args...) {
			t.Errorf(format, args...)
		}
	})
}
