package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/tenntenn/modver"
	tnntransform "github.com/tenntenn/text/transform"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/transform"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// WithModules creates a temp dir which is copied from srcdir and generates vendor directory with go.mod.
// go.mod can be specified by modfile.
// Example:
//	func TestAnalyzer(t *testing.T) {
//		testdata := testutil.WithModules(t, analysistest.TestData(), nil)
//		analysistest.Run(t, testdata, sample.Analyzer, "a")
//	}
func WithModules(t *testing.T, srcdir string, modfile io.Reader) (dir string) {
	abs := func(relPath string) string {
		return filepath.ToSlash(filepath.Clean(filepath.Join(srcdir, filepath.FromSlash(relPath))))
	}
	return WithModulesFS(t, os.DirFS(srcdir), modfile, abs)
}

// AbsPathFunc convert relative path to absolute path.
// Each path is slash separated.
type AbsPathFunc func(relPath string) string

// WithModules creates a temp dir which is copied from srcdir and generates vendor directory with go.mod.
// go.mod can be specified by modfile.
func WithModulesFS(t TestingT, srcFsys fs.FS, modfile io.Reader, abs AbsPathFunc) (dir string) {
	t.Helper()
	dir = t.TempDir()

	var modRoots []string
	err := fs.WalkDir(srcFsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		ds, err := fs.ReadDir(srcFsys, path)
		if err != nil {
			return err
		}

		var eg errgroup.Group
		dstDir := filepath.Join(dir, filepath.FromSlash(path))
		for _, d := range ds {
			d := d
			eg.Go(func() error {
				switch {
				case d.Name() == "go.mod":
					modRoots = append(modRoots, dstDir)
					if err := copyModFile(srcFsys, dstDir, path, modfile, abs); err != nil {
						return err
					}
				default:
					srcPath := filepath.Join(path, d.Name())
					src, err := srcFsys.Open(srcPath)
					if err != nil {
						return fmt.Errorf("cannot open %s: %w", srcPath, err)
					}
					defer src.Close()

					dstName := filepath.Join(dstDir, d.Name())
					if err := copyFile(dstName, src); err != nil {
						return err
					}
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			t.Fatal("unexpected error:", err)
		}

		return nil
	})
	if err != nil {
		t.Fatal("go mod vendor:", err)
	}

	for _, dir := range modRoots {
		execCmd(t, dir, "go", "mod", "tidy")
		execCmd(t, dir, "go", "mod", "vendor")
	}

	return dir
}

func copyModFile(srcFsys fs.FS, dstDir, srcDir string, modfileReader io.Reader, abs AbsPathFunc) error {
	srcName := filepath.Join(srcDir, "go.mod")

	if modfileReader == nil {
		f, err := srcFsys.Open(srcName)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", srcName, err)
		}
		defer f.Close()
		modfileReader = f
	}

	moddata, err := io.ReadAll(modfileReader)
	if err != nil {
		return fmt.Errorf("cannot read go.mod in %s: %w", srcDir, err)
	}

	file, err := modfile.Parse(srcName, moddata, nil)
	if err != nil {
		return fmt.Errorf("cannot parse go.mod in %s: %w", srcDir, err)
	}

	// fix relative pathes of replace directive to absolute pathes
	for _, r := range file.Replace {
		fpath := filepath.FromSlash(r.New.Path)
		if filepath.IsAbs(r.New.Path) {
			continue
		}

		err := file.AddReplace(r.Old.Path, r.Old.Version, abs(fpath), r.New.Version)
		if err != nil {
			return fmt.Errorf("cannot add replace directive: %w", err)
		}
	}

	newMod, err := file.Format()
	if err != nil {
		return fmt.Errorf("cannot format go.mod: %w", err)
	}

	dstName := filepath.Join(dstDir, "go.mod")
	if err := copyFile(dstName, bytes.NewReader(newMod)); err != nil {
		return err
	}

	return nil
}

func copyFile(dstName string, src io.Reader) error {
	dst, err := os.Create(dstName)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", dstName, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy file to %s: %w", dstName, err)
	}

	if err := dst.Close(); err != nil {
		return fmt.Errorf("cannot close %s: %w", dstName, err)
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

	data, err := os.ReadFile(path)
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

func execCmd(t TestingT, dir, cmd string, args ...string) io.Reader {
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
