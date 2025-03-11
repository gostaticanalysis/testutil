package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
func WithModules(t *testing.T, testdata string, gomodfile io.Reader) (dir string) {
	t.Helper()
	dir = t.TempDir()
	if err := copy.Copy(testdata, dir); err != nil {
		t.Fatal("cannot copy a directory:", err)
	}

	src := filepath.Join(dir, "src")

	var data []byte
	if gomodfile != nil {
		_data, err := io.ReadAll(gomodfile)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		data = _data
	}

	replaceGoMod(t, src, data)
	addLineComment(t, src)

	return dir
}

func replaceGoMod(t *testing.T, src string, gomodfile []byte) {
	t.Helper()

	var ok bool
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != "go.mod" {
			return nil
		}

		if gomodfile != nil {
			if err := os.WriteFile(path, gomodfile, 0o644); err != nil {
				t.Fatal("cannot write go.mod:", err)
			}
		}

		dir := filepath.Dir(path)
		execCmd(t, dir, "go", "mod", "tidy")
		execCmd(t, dir, "go", "mod", "vendor")
		ok = true

		return nil
	})
	if err != nil {
		t.Fatal("go mod vendor:", err)
	}

	if ok {
		return
	}

	if gomodfile == nil {
		t.Fatal("does not find go.mod")
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgdir := filepath.Join(src, entry.Name())
		fn := filepath.Join(pkgdir, "go.mod")
		gomod, err := modfile.Parse(fn, gomodfile, nil)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		gomod.AddModuleStmt(entry.Name())
		gomod.Cleanup()

		out, err := gomod.Format()
		if err != nil {
			t.Fatal("cannot format go.mod:", err)
		}

		if err := os.WriteFile(fn, out, 0o644); err != nil {
			t.Fatal("cannot write go.mod:", err)
		}

		execCmd(t, pkgdir, "go", "mod", "tidy")
		execCmd(t, pkgdir, "go", "mod", "vendor")
	}
}

func addLineComment(t *testing.T, src string) {
	t.Helper()

	moddirs := make(map[string]string)
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Prepend line directive to .go files
		if filepath.Ext(info.Name()) == ".go" {
			dir := filepath.Dir(path)
			moddir, ok := moddirs[dir]
			if !ok {
				r := execCmd(t, dir, "go", "list", "-m", "-json")
				var mod struct {
					Dir string
				}
				if err := json.NewDecoder(r).Decode(&mod); err != nil {
					t.Fatal("unexpected error:", err)
				}
				realModDir, err := filepath.EvalSymlinks(mod.Dir)
				if err != nil {
					t.Fatal("failed to eval symlinks for module dir:", err)
				}
				moddir = realModDir
				moddirs[dir] = moddir
			}

			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				t.Fatal("failed to eval symlinks for module path:", err)
			}

			rel, err := filepath.Rel(moddir, realPath)
			if err != nil {
				t.Fatal("cannot get relative path:", err)
			}
			if err := prependToFile(path, fmt.Sprintf("//line %s:1\n", rel)); err != nil {
				t.Fatal("cannot prepend line directive:", err)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal("go mod vendor:", err)
	}
}

func prependToFile(filename string, ld string) error {
	f, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if _, err := f.WriteString(ld + "\n"); err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	return nil
}

// ModFile opens a mod file with the path and fixes versions by the version fixer.
// If the path is direcotry, ModFile opens go.mod which is under the path.
func ModFile(t *testing.T, path string, fix modfile.VersionFixer) io.Reader {
	t.Helper()

	gomod := path

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal("cannot get stat of path:", err)
	}
	if info.IsDir() {
		gomod = modfilePath(t, path)
	}

	data, err := os.ReadFile(gomod)
	if err != nil {
		t.Fatal("cannot read go.mod:", err)
	}

	f, err := modfile.Parse(gomod, data, fix)
	if err != nil {
		t.Fatal("cannot parse go.mod:", err)
	}

	out, err := f.Format()
	if err != nil {
		t.Fatal("cannot format go.mod:", err)
	}

	return bytes.NewReader(out)
}

func modfilePath(t *testing.T, dir string) string {
	t.Helper()

	var stdout bytes.Buffer
	cmd := exec.Command("go", "list", "-m", "-f", "{{.GoMod}}")
	cmd.Dir = dir
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("cannot get the parent module with %s: %v", dir, err)
	}

	gomod := strings.TrimSpace(stdout.String())
	if gomod == "" {
		t.Fatalf("cannot find go.mod, %s may not managed with Go Modules", dir)
	}

	return gomod
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
	_cmd.Env = append(os.Environ(), "GOWORK=off")
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
