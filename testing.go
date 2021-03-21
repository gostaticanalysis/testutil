package testutil

type TestingT interface {
	Cleanup(func())
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
}
