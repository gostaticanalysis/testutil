package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/otiai10/copy"
	"github.com/tenntenn/modver"
	tnntransform "github.com/tenntenn/text/transform"
	"golang.org/x/mod/modfile"
	"golang.org/x/text/transform"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// WithModules creates a temp dir which is copied from srcdir and generates vendor directory with go.mod.
// go.mod can be specified by modfileReader.
// Example:
//
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
			// Prepend line directive to .go files
			if filepath.Ext(file.Name()) == ".go" {
				fn := filepath.Join(path, file.Name())
				rel, err := filepath.Rel(dir, fn)
				if err != nil {
					t.Fatal("cannot get relative path:", err)
				}
				if err := prependToFile(fn, fmt.Sprintf("//line %s:1\n", rel)); err != nil {
					t.Fatal("cannot prepend line directive:", err)
				}
			}
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

func appendFileContent(tmp io.Writer, filename string) error {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(tmp, f); err != nil {
		return err
	}
	return nil
}

func prependToFile(filename string, content string) error {
	// Create temp file
	tmp, err := os.CreateTemp("", "prepend")
	if err != nil {
		return err
	}
	tmpFilePath := tmp.Name()
	// Prepend line directive
	if _, err := tmp.Write([]byte(content)); err != nil {
		tmp.Close()
		os.Remove(tmpFilePath)
		return err
	}
	// Write the original file content
	if err := appendFileContent(tmp, filename); err != nil {
		tmp.Close()
		os.Remove(tmpFilePath)
		return err
	}
	// Close tmp file
	if err = tmp.Close(); err != nil {
		os.Remove(tmpFilePath)
		return err
	}
	// Rename the temp file
	if err = os.Rename(tmpFilePath, filename); err != nil {
		os.Remove(tmpFilePath)
		return err
	}
	return nil
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
type ModuleVersion = modver.ModuleVersion

// AllVersion get available all versions of the module.
func AllVersion(t *testing.T, module string) []ModuleVersion {
	t.Helper()
	vers, err := modver.AllVersion(module)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	return vers
}

// FilterVersion returns versions of the module which satisfy the constraints such as ">= v2.0.0"
// The constraints rule uses github.com/hashicorp/go-version.
//
// Example:
//
//	func TestAnalyzer(t *testing.T) {
//		vers := FilterVersion(t, "github.com/tenntenn/greeting/v2", ">= v2.0.0")
//		RunWithVersions(t, analysistest.TestData(), sample.Analyzer, vers, "a")
//	}
func FilterVersion(t *testing.T, module, constraints string) []ModuleVersion {
	t.Helper()
	vers, err := modver.FilterVersion(module, constraints)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	return vers
}

// LatestVersion returns most latest versions (<= max) of each minner version.
//
// Example:
//
//	func TestAnalyzer(t *testing.T) {
//		vers := LatestVersion(t, "github.com/tenntenn/greeting/v2", 3)
//		RunWithVersions(t, analysistest.TestData(), sample.Analyzer, vers, "a")
//	}
func LatestVersion(t *testing.T, module string, max int) []ModuleVersion {
	t.Helper()
	vers, err := modver.LatestVersion(module, max)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	return vers
}

// RunWithVersions runs analysistest.Run with modules which version is specified the vers.
//
// Example:
//
//	func TestAnalyzer(t *testing.T) {
//		vers := AllVersion(t, "github.com/tenntenn/greeting/v2")
//		RunWithVersions(t, analysistest.TestData(), sample.Analyzer, vers, "a")
//	}
//
// The test run in temporary directory which is isolated the dir.
// analysistest.Run uses packages.Load and it prints errors into os.Stderr.
// Becase the error messages include the temporary directory path, so RunWithVersions replaces os.Stderr.
// Replacing os.Stderr is not thread safe.
// If you want to turn off replacing os.Stderr, you can use ReplaceStderr(false).
func RunWithVersions(t *testing.T, dir string, a *analysis.Analyzer, vers []ModuleVersion, pkg string) map[ModuleVersion][]*analysistest.Result {
	t.Helper()

	path := filepath.Join(dir, "src", pkg)

	results := make(map[ModuleVersion][]*analysistest.Result, len(vers))
	for _, modver := range vers {
		modver := modver
		t.Run(modver.String(), func(t *testing.T) {
			t.Parallel()
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
	t.Helper()
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

// ReplaceStderr sets whether RunWithVersions replace os.Stderr or not.
// The default value is true which means that RunWithVersions replaces os.Stderr.
func ReplaceStderr(onoff bool) {
	stderrMutex.Lock()
	doNotUseFilteredStderr = !onoff
	stderrMutex.Unlock()
}

func replaceStderr(t *testing.T, old, new string) {
	t.Helper()

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
