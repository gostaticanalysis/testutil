package testutil_test

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/josharian/txtarfs"
	"github.com/tenntenn/golden"
	"golang.org/x/tools/txtar"
)

var (
	flagUpdate bool
)

func init() {
	flag.BoolVar(&flagUpdate, "update", false, "update golden files")
}

func TestWithModulesFS(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		replaceModfile bool
		afterCommand   string
	}{
		"normal":           {false, ""},
		"vendoring":        {false, ""},
		"replacemodfile":   {true, ""},
		"replacedirective": {false, ""},
		"linecomment":      {false, "go test"},
	}

	testdata, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal("cannot get absolute path of testdata", err)
	}
	testdata = filepath.Join(testdata, t.Name())

	for name, tt := range cases {
		name, tt := name, tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var mt MockTestingT
			mt.TempDirFunc = t.TempDir
			srcdir := filepath.Join(testdata, name)
			src := golden.Txtar(t, srcdir)
			srcFsys := txtarfs.As(txtar.Parse([]byte(src)))

			var modfile io.Reader
			if tt.replaceModfile {
				mf, err := os.ReadFile(filepath.Join(testdata, name+"_go.mod"))
				if err != nil {
					t.Fatal("unexpected error:", err)
				}
				modfile = bytes.NewReader(mf)
			}

			abs := func(relPath string) string {
				return filepath.ToSlash(filepath.Clean(filepath.Join(srcdir, filepath.FromSlash(relPath))))
			}

			gotDir := testutil.WithModulesFS(&mt, srcFsys, modfile, abs)
			if msg := mt.FatalMsg(); msg != "" {
				t.Fatal("unexpected fatal:", msg)
			}
			got := golden.Txtar(t, gotDir)
			// Remove local path and convert to constant path
			got = strings.ReplaceAll(got, filepath.ToSlash(testdata), "/path/to/testdata")

			gotCmd := execCmd(t, gotDir, tt.afterCommand)

			if flagUpdate {
				golden.Update(t, testdata, name, got)
				golden.Update(t, testdata, name+"_after", gotCmd)
				return
			}

			if diff := golden.Diff(t, testdata, name, got); diff != "" {
				t.Error(diff)
			}

			if diff := golden.Diff(t, testdata, name+"_after", gotCmd); diff != "" {
				t.Error(diff)
			}

		})
	}
}

func execCmd(t *testing.T, dir, cmd string) string {
	t.Helper()
	if cmd == "" {
		return ""
	}

	args := strings.Split(cmd, " ")
	var buf bytes.Buffer
	_cmd := exec.Command(args[0], args[1:]...)
	_cmd.Stdout = &buf
	_cmd.Stderr = &buf
	_cmd.Dir = dir
	var eerr *exec.Error
	err := _cmd.Run()
	if errors.As(err, &eerr) {
		t.Fatal(eerr)
	}
	return buf.String()
}
