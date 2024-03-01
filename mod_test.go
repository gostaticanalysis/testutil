package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestWithModules(t *testing.T) {
	t.Parallel()

	t.Run("The line directive is appended to the go source codes", func(t *testing.T) {
		testdata := WithModules(t, analysistest.TestData(), nil)
		tests := []struct {
			path string
			want string
		}{
			{filepath.Join(testdata, "src", "a", "a.go"), "//line src/a/a.go:1"},
			{filepath.Join(testdata, "src", "a", "b", "b.go"), "//line src/a/b/b.go:1"},
		}
		for _, tt := range tests {
			b, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(b[:len(tt.want)]); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		}
	})
}
