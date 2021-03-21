package testutil_test

import (
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gostaticanalysis/testutil"
	"github.com/josharian/txtarfs"
	"golang.org/x/tools/txtar"
)

func TestWriteFilesFS(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		files string
		fatal bool
	}{
		"single": {"-- a.txt --\nhello", false},
		"multi": {"-- a.txt --\nhello\n-- b.txt --\ngophers", false},
		"nested": {"-- a.txt --\nhello\n-- b/b.txt --\ngophers", false},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			var mt MockT
			a := txtar.Parse([]byte(tt.files))
			fsys := txtarfs.As(a)

			dir := testutil.WriteFilesFS(&mt, fsys)

			switch {
			case tt.fatal && !mt.IsFatal:
				t.Fatal("expected fatal does not occur")
			case !tt.fatal && mt.IsFatal:
				t.Fatal("unexpected fatal")
			}

			gotA, err := txtarfs.From(os.DirFS(dir))
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			got := string(txtar.Format(gotA))

			wantA := &txtar.Archive{
				Files: make([]txtar.File, len(a.Files)),
			}
			for i := range wantA.Files {
				wantA.Files[i] = txtar.File{
					Name: path.Join("src", a.Files[i].Name),
					Data: a.Files[i].Data,
				}
			}
			want := string(txtar.Format(wantA))

			if diff := cmp.Diff(want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
