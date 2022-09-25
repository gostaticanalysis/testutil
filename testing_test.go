package testutil_test

import (
	"bytes"
	"fmt"
)

type MockTestingT struct {
	mockTestingT
}

func (m *MockTestingT) IsError() bool {
	return len(m.ErrorCalls()) > 0 ||
		len(m.ErrorfCalls()) > 0
}

func (m *MockTestingT) ErrorMsg() string {
	if !m.IsError() {
		return ""
	}

	var buf bytes.Buffer

	for _, call := range m.ErrorCalls() {
		fmt.Fprintln(&buf, call.Args...)
	}

	for _, call := range m.ErrorfCalls() {
		fmt.Fprintf(&buf, call.Format, call.Args...)
	}

	return buf.String()
}

func (m *MockTestingT) IsFatal() bool {
	return len(m.FatalCalls()) > 0 ||
		len(m.FatalfCalls()) > 0
}

func (m *MockTestingT) FatalMsg() string {
	if !m.IsFatal() {
		return ""
	}

	var buf bytes.Buffer

	for _, call := range m.FatalCalls() {
		fmt.Fprintln(&buf, call.Args...)
	}

	for _, call := range m.FatalfCalls() {
		fmt.Fprintf(&buf, call.Format, call.Args...)
	}

	return buf.String()
}
