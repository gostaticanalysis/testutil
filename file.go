package testutil

import (
	"io/fs"

	"golang.org/x/tools/go/analysis/analysistest"
)

// WriteFiles wrapper of analysistest.WriteFiles.
//
// WriteFiles is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using filemap (which
// maps file names to contents).
//
// On success it returns the name of the directory.
// The directory will be deleted by t.Cleanup.
func WriteFiles(t TestingT, filemap map[string]string) string {
	t.Helper()
	dir, clean, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(clean)
	return dir
}

// WriteFiles wrapper of analysistest.WriteFiles.
//
// WriteFiles is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using fs.FS.
//
// On success it returns the name of the directory.
// The directory will be deleted by t.Cleanup.
func WriteFilesFS(t TestingT, fsys fs.FS) string {
	t.Helper()
	filemap := make(map[string]string)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		filemap[path] = string(data)

		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return WriteFiles(t, filemap)
}
