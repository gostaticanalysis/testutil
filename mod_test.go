package testutil_test

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
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
	}{
		"normal":           {false},
		"vendoring":        {false},
		"replacemodfile":   {true},
		"replacedirective": {false},
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

			if flagUpdate {
				golden.Update(t, testdata, name, got)
				return
			}

			if diff := golden.Diff(t, testdata, name, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
