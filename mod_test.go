package testutil

import (
	"os"
	"path/filepath"
	"strings"
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
			{filepath.Join(testdata, "src", "a", "a.go"), "//line a/a.go:1"},
			{filepath.Join(testdata, "src", "a", "b", "b.go"), "//line a/b/b.go:1"},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				t.Parallel()

				src, err := os.ReadFile(tt.path)
				if err != nil {
					t.Fatal(err)
				}

				got, _, _ := strings.Cut(string(src), "\n")
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			})
		}
	})
}
