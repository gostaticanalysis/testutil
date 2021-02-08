package testutil

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// WriteFiles wrapper of analysistest.WriteFiles.
//
// WriteFiles is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using filemap (which
// maps file names to contents).
//
// On success it returns the name of the directory and a cleanup function to delete it.
func WriteFiles(t *testing.T, filemap map[string]string) string {
	t.Helper()
	dir, clean, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(clean)
	return dir
}
