package testutil

//go:generate moq -out testing_moq_test.go -pkg testutil_test -stub . TestingT:mockTestingT
type TestingT interface {
	Cleanup(func())
	Errorf(format string, args ...any)
	Error(args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Helper()
	TempDir() string
}
