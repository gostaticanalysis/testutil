package testutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/gostaticanalysis/testutil"
)

func TestWithModules_LineComment(t *testing.T) {
	t.Parallel()

	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	tests := []struct {
		path string
		want string
	}{
		{filepath.Join(testdata, "src", "a", "a.go"), "//line a.go:1"},
		{filepath.Join(testdata, "src", "a", "b", "b.go"), "//line b/b.go:1"},
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
}
