package testutil_test

import "github.com/gostaticanalysis/testutil"

type MockT struct {
	testutil.TestingT
	IsErr   bool
	IsFatal bool
}

func (t *MockT) Errorf(_ string, _ ...any) { t.IsErr = true }
func (t *MockT) Error(_ ...any)            { t.IsErr = true }
func (t *MockT) Fatalf(_ string, _ ...any) { t.IsFatal = true }
func (t *MockT) Fatal(_ ...any)            { t.IsFatal = true }
func (t *MockT) Cleanup(func())            {}
func (t *MockT) Helper()                   {}
