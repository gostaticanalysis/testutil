//go:build go1.24

package testutil

import (
	"encoding/json"
	"testing"
)

// Remove tool directive from go.mod.
func removeToolDirective(tb testing.TB, dir string) {
	tb.Helper()

	r := execCmd(tb, dir, "go", "mod", "edit", "-json")
	var modfile struct {
		Tool []struct {
			Path string
		}
	}
	if err := json.NewDecoder(r).Decode(&modfile); err != nil {
		tb.Fatal("unexpected error:", err)
	}

	for _, tool := range modfile.Tool {
		execCmd(tb, dir, "go", "mod", "edit", "-droptool", tool.Path)
	}
}
