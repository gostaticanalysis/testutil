package testutil_test

type MockTestingT struct {
	mockTestingT
}

func (m *MockTestingT) IsError() bool {
	return len(m.ErrorCalls()) > 0 ||
		len(m.ErrorfCalls()) > 0
}

func (m *MockTestingT) IsFatal() bool {
	return len(m.FatalCalls()) > 0 ||
		len(m.FatalfCalls()) > 0
}
