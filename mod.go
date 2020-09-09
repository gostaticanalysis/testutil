package testutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/rogpeppe/go-internal/modfile"
)

// WithModules creates a temp dir which is copied from baseDir and generates vendor directory with go.mod.
// go.mod can be specified by modfileReader.
func WithModules(t *testing.T, baseDir string, modfileReader io.Reader) (dir string) {
	t.Helper()
	dir = t.TempDir()
	if err := copy.Copy(baseDir, dir); err != nil {
		t.Fatal("cannot copy a directory:", err)
	}

	if modfileReader != nil {
		fn := filepath.Join(dir, "go.mod")
		f, err := os.Create(fn)
		if err != nil {
			t.Fatal("cannot create go.mod:", err)
		}

		if _, err := io.Copy(f, modfileReader); err != nil {
			t.Fatal("cannot create go.mod:", err)
		}

		if err := f.Close(); err != nil {
			t.Fatal("cannot close go.mod", err)
		}
	}

	cmd := exec.Command("go", "mod", "vendor")
	cmd.Stdout = ioutil.Discard
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal("go mod vendor:", err, "\n", errBuf.String())
	}

	return dir
}

// ModFile opens a mod file and fixes versions by the version fixer.
func ModFile(t *testing.T, modfilePath string, fix modfile.VersionFixer) io.Reader {
	t.Helper()
	data, err := ioutil.ReadFile(modfilePath)
	if err != nil {
		t.Fatal("cannot read go.mod:", err)
	}

	f, err := modfile.Parse(modfilePath, data, fix)
	if err != nil {
		t.Fatal("cannot parse go.mod:", err)
	}

	out, err := f.Format()
	if err != nil {
		t.Fatal("cannot format go.mod:", err)
	}

	return bytes.NewReader(out)
}
