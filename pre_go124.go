//go:build !go1.24

package testutil

import "testing"

// This function does nothing.
// Because go.mod does not have a tool directive until Go1.24.
func removeToolDirective(tb testing.TB) {
	t.Helper()
}
