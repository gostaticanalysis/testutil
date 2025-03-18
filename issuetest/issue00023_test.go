//go:build go1.24

package issuetest

import (
	"flag"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/tenntenn/golden"

	"github.com/gostaticanalysis/testutil"
)

var flagGoldenUpdate bool

func init() {
	flag.BoolVar(&flagGoldenUpdate, "update-golden", false, "update golden files")
}

func TestIssue00023(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	target := filepath.Join(testdata, "target")
	tmpdir := testutil.WithModules(t, target, nil)
	got := golden.Txtar(t, tmpdir)
	if diff := golden.Check(t, flagGoldenUpdate, testdata, t.Name(), got); diff != "" {
		t.Errorf("golden file mismatch: (-want, +got) = %s", diff)
	}
}
