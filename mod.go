package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/otiai10/copy"
	tnntransform "github.com/tenntenn/text/transform"
	"golang.org/x/mod/modfile"
	"golang.org/x/text/transform"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// WithModules creates a temp dir which is copied from srcdir and generates vendor directory with go.mod.
// go.mod can be specified by modfileReader.
// Example:
//	func TestAnalyzer(t *testing.T) {
//		testdata := testutil.WithModules(t, analysistest.TestData(), nil)
//		analysistest.Run(t, testdata, sample.Analyzer, "a")
//	}
func WithModules(t *testing.T, srcdir string, modfile io.Reader) (dir string) {
	t.Helper()
	dir = t.TempDir()
	if err := copy.Copy(srcdir, dir); err != nil {
		t.Fatal("cannot copy a directory:", err)
	}

	var ok bool
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		files, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.Name() == "go.mod" {
				if modfile != nil {
					fn := filepath.Join(path, "go.mod")
					f, err := os.Create(fn)
					if err != nil {
						t.Fatal("cannot create go.mod:", err)
					}

					if _, err := io.Copy(f, modfile); err != nil {
						t.Fatal("cannot create go.mod:", err)
					}

					if err := f.Close(); err != nil {
						t.Fatal("cannot close go.mod", err)
					}
				}
				execCmd(t, path, "go", "mod", "vendor")
				ok = true
				return nil
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal("go mod vendor:", err)
	}

	if !ok {
		t.Fatal("does not find go.mod")
	}

	return dir
}

// ModFile opens a mod file with the path and fixes versions by the version fixer.
// If the path is direcotry, ModFile opens go.mod which is under the path.
func ModFile(t *testing.T, path string, fix modfile.VersionFixer) io.Reader {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal("cannot get stat of path:", err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "go.mod")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read go.mod:", err)
	}

	f, err := modfile.Parse(path, data, fix)
	if err != nil {
		t.Fatal("cannot parse go.mod:", err)
	}

	out, err := f.Format()
	if err != nil {
		t.Fatal("cannot format go.mod:", err)
	}

	return bytes.NewReader(out)
}

// ModuleVersion has module path and its version.
type ModuleVersion struct {
	Module  string
	Version string
}

// String implements fmt.Stringer.
func (modver ModuleVersion) String() string {
	return fmt.Sprintf("%s@%s", modver.Module, modver.Version)
}

// AllVersion get available all versions of the module.
func AllVersion(t *testing.T, module string) []ModuleVersion {
	t.Helper()

	dir := t.TempDir()
	execCmd(t, dir, "go", "mod", "init", "tmp")
	r := execCmd(t, dir, "go", "list", "-m", "-versions", "-json", module)
	var v struct{ Versions []string }
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatal("cannot decode JSON", err)
	}

	vers := make([]ModuleVersion, len(v.Versions))
	for i := range v.Versions {
		vers[i] = ModuleVersion{
			Module:  module,
			Version: v.Versions[i],
		}
	}

	return vers
}

// FilterVersion returns versions of the module which satisfy the constraints such as ">= v2.0.0"
// The constraints rule uses github.com/hashicorp/go-version.
//
// Example:
//	func TestAnalyzer(t *testing.T) {
//		vers := FilterVersion(t, "github.com/tenntenn/greeting/v2", ">= v2.0.0")
//		RunWithVersions(t, analysistest.TestData(), mod.Analyzer, vers, "a")
//	}
func FilterVersion(t *testing.T, module, constraints string) []ModuleVersion {
	t.Helper()

	c, err := version.NewConstraint(constraints)
	if err != nil {
		t.Fatal("cannot parse constraints", err)
	}

	var vers []ModuleVersion
	for _, ver := range AllVersion(t, module) {
		v, err := version.NewVersion(ver.Version)
		if err != nil {
			t.Fatal("cannot parse version", err)
		}
		if c.Check(v) {
			vers = append(vers, ver)
		}
	}

	return vers
}

// RunWithVersions runs analysistest.Run with modules which version is specified the vers.
//
// Example:
//	func TestAnalyzer(t *testing.T) {
//		vers := AllVersion(t, "github.com/tenntenn/greeting/v2")
//		RunWithVersions(t, analysistest.TestData(), mod.Analyzer, vers, "a")
//	}
//
// The test run in temporary directory which is isolated the dir.
// analysistest.Run uses packages.Load and it prints errors into os.Stderr.
// Becase the error messages include the temporary directory path, so RunWithVersions replaces os.Stderr.
// Replacing os.Stderr is not thread safe.
// If you want to turn off replacing os.Stderr, you can use ReplaceStderr(false).
func RunWithVersions(t *testing.T, dir string, a *analysis.Analyzer, vers []ModuleVersion, pkg string) map[ModuleVersion][]*analysistest.Result {
	path := filepath.Join(dir, "src", pkg)

	results := make(map[ModuleVersion][]*analysistest.Result, len(vers))
	for _, modver := range vers {
		modver := modver
		t.Run(modver.String(), func(t *testing.T) {
			modfile := ModFile(t, path, func(module, ver string) (string, error) {
				if modver.Module == module {
					return modver.Version, nil
				}
				return ver, nil
			})
			tmpdir := WithModules(t, dir, modfile)
			replaceStderr(t, tmpdir, dir)
			results[modver] = analysistest.Run(t, tmpdir, a, pkg)
		})
	}

	return results
}

func execCmd(t *testing.T, dir, cmd string, args ...string) io.Reader {
	var stdout, stderr bytes.Buffer
	_cmd := exec.Command(cmd, args...)
	_cmd.Stdout = &stdout
	_cmd.Stderr = &stderr
	_cmd.Dir = dir
	if err := _cmd.Run(); err != nil {
		t.Fatal(err, "\n", &stderr)
	}
	return &stdout
}

var (
	stderrMutex            sync.RWMutex
	doNotUseFilteredStderr bool
)

func ReplaceStderr(onoff bool) {
	stderrMutex.Lock()
	doNotUseFilteredStderr = !onoff
	stderrMutex.Unlock()
}

func replaceStderr(t *testing.T, old, new string) {
	stderrMutex.RLock()
	ok := !doNotUseFilteredStderr
	stderrMutex.RUnlock()
	if !ok {
		return
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal("cannot create pipe", err)
	}

	origStderr := os.Stderr
	stderrMutex.Lock()
	os.Stderr = w
	stderrMutex.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		stderrMutex.Lock()
		os.Stderr = origStderr
		stderrMutex.Unlock()
	})

	go func() {
		t := tnntransform.ReplaceString(old, new)
		w := transform.NewWriter(origStderr, t)
		for {
			select {
			case <-ctx.Done():
			default:
				io.CopyN(w, r, 1024)
			}
		}
	}()
}
