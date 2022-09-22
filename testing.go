package testutil

type TestingT interface {
	Cleanup(func())
	Errorf(format string, args ...any)
	Error(args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Helper()
	TempDir() string
}
