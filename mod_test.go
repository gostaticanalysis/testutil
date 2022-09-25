package testutil_test

import (
	"flag"
	"io"
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
		modfile io.Reader
	}{
		"normal":    {nil},
		"vendoring": {nil},
	}

	testdata := filepath.Join("testdata", t.Name())

	for name, tt := range cases {
		name, tt := name, tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var mt MockTestingT
			mt.TempDirFunc = t.TempDir
			src := golden.Txtar(t, filepath.Join(testdata, name))
			srcFsys := txtarfs.As(txtar.Parse([]byte(src)))
			gotDir := testutil.WithModulesFS(&mt, srcFsys, tt.modfile)
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
