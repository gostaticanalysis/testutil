package testutil_test

import "github.com/gostaticanalysis/testutil"

type MockT struct {
	IsErr   bool
	IsFatal bool
}

func (t *MockT) Cleanup(_ func())                  {}
func (t *MockT) Errorf(_ string, _ ...interface{}) { t.IsErr = true }
func (t *MockT) Fatalf(_ string, _ ...interface{}) { t.IsFatal = true }
func (t *MockT) Helper()                           {}

var _ testutil.TestingT = (*MockT)(nil)
