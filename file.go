package testutil

import (
	"io/fs"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/txtar"
)

// WriteFiles wrapper of analysistest.WriteFiles.
//
// WriteFiles is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using filemap (which
// maps file names to contents).
//
// On success it returns the name of the directory.
// The directory will be deleted by t.Cleanup.
func WriteFiles(t testing.TB, filemap map[string]string) string {
	t.Helper()
	dir, clean, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(clean)
	return dir
}

// WriteFilesFS wrapper of analysistest.WriteFiles.
//
// WriteFilesFS is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using fs.FS.
//
// On success it returns the name of the directory.
// The directory will be deleted by t.Cleanup.
func WriteFilesFS(t testing.TB, fsys fs.FS) string {
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

		filemap[filepath.FromSlash(path)] = string(data)

		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return WriteFiles(t, filemap)
}

// WriteFilesTxtar wrapper of analysistest.WriteFiles.
//
// WriteFilesTxtar is a helper function that creates a temporary directory
// and populates it with a GOPATH-style project using [*txtar.Archive].
//
// On success it returns the name of the directory.
// The directory will be deleted by t.Cleanup.
func WriteFilesTxtar(t testing.TB, a *txtar.Archive) string {
	t.Helper()

	filemap := make(map[string]string)
	for _, file := range a.Files {
		filemap[filepath.FromSlash(file.Name)] = string(file.Data)
	}

	return WriteFiles(t, filemap)
}
