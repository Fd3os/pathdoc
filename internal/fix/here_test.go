package fix

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fd3os/pathdoc/internal/win"
)

// project makes base\<parts...> with an android subfolder and removes any
// junctions under root before the temp folder goes.
func project(t *testing.T, parts ...string) (base, proj, root string) {
	t.Helper()
	base = t.TempDir()
	proj = filepath.Join(append([]string{base}, parts...)...)
	if err := os.MkdirAll(filepath.Join(proj, "android"), 0o755); err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(base, "dev")
	t.Cleanup(func() {
		links, _ := filepath.Glob(filepath.Join(root, "p", "*"))
		for _, l := range links {
			os.Remove(l)
		}
	})
	return base, proj, root
}

func kinds(st *State) map[string]int {
	n := map[string]int{}
	for _, c := range st.Changes {
		n[c.Kind]++
	}
	return n
}

func TestHereMakesJunctionAndKeepsSubpath(t *testing.T) {
	_, proj, root := project(t, "مشاريع", "متجر")
	st := &State{Root: root}
	link, cd, err := Here(proj, filepath.Join(proj, "android"), root, "", st)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "p", "mtjr"); link != want {
		t.Fatalf("link = %q, want %q", link, want)
	}
	if cd != filepath.Join(link, "android") {
		t.Fatalf("cd = %q", cd)
	}
	if got, err := win.JunctionTarget(link); err != nil || got != proj {
		t.Fatalf("junction -> %q, %v", got, err)
	}
	if k := kinds(st); k["junction"] != 1 || k["dir"] != 2 {
		t.Fatalf("state changes = %v, want 1 junction and the 2 folders made (root, p)", k)
	}
}

func TestHereReusesItsOwnJunction(t *testing.T) {
	_, proj, root := project(t, "متجر")
	st := &State{Root: root}
	first, _, err := Here(proj, proj, root, "", st)
	if err != nil {
		t.Fatal(err)
	}
	n := len(st.Changes)
	again, _, err := Here(proj, proj, root, "", st)
	if err != nil || again != first || len(st.Changes) != n {
		t.Fatalf("second run: %q %v, changes %d -> %d", again, err, n, len(st.Changes))
	}
}

func TestHereStepsAroundATakenName(t *testing.T) {
	base, proj, root := project(t, "متجر")
	other := filepath.Join(base, "other")
	os.Mkdir(other, 0o755)
	os.MkdirAll(filepath.Join(root, "p"), 0o755)
	if err := win.CreateJunction(filepath.Join(root, "p", "mtjr"), other); err != nil {
		t.Fatal(err)
	}
	link, _, err := Here(proj, proj, root, "", &State{Root: root})
	if err != nil || link != filepath.Join(root, "p", "mtjr-2") {
		t.Fatalf("link = %q, %v; want mtjr-2", link, err)
	}
}

func TestHereNameHasNoSpaces(t *testing.T) {
	_, proj, root := project(t, "My Apps", "My متجر")
	link, _, err := Here(proj, proj, root, "", &State{Root: root})
	if err != nil || filepath.Base(link) != "my-mtjr" {
		t.Fatalf("link = %q, %v", link, err)
	}
}

func TestHereOnAnASCIIProject(t *testing.T) {
	_, proj, root := project(t, "plain")
	if _, _, err := Here(proj, proj, root, "", &State{Root: root}); !errors.Is(err, ErrAlreadyASCII) {
		t.Fatalf("err = %v, want ErrAlreadyASCII", err)
	}
}

func TestHereRejectsABadName(t *testing.T) {
	_, proj, root := project(t, "متجر")
	if _, _, err := Here(proj, proj, root, "my app", &State{Root: root}); err == nil {
		t.Fatal("a name with a space: want an error")
	}
}

func TestValidateRoot(t *testing.T) {
	if err := ValidateRoot(`C:\dev`); err != nil {
		t.Errorf("C:\\dev: %v", err)
	}
	for _, bad := range []string{`C:\مجلد`, `C:\my dev`, `dev`, `\\server\share`} {
		if err := ValidateRoot(bad); err == nil {
			t.Errorf("ValidateRoot(%q): want an error", bad)
		}
	}
}

func TestStateSavesAndLoads(t *testing.T) {
	path := StatePath(filepath.Join(t.TempDir(), "محمد", "AppData", "Local"))
	st, err := LoadState(path)
	if err != nil || len(st.Changes) != 0 {
		t.Fatalf("missing state: %+v %v", st, err)
	}
	st.Changes = append(st.Changes, Change{Kind: "junction", Path: `C:\dev\p\x`, Target: `C:\ابوي`})
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	back, err := LoadState(path)
	if err != nil || len(back.Changes) != 1 || back.Changes[0].Target != `C:\ابوي` {
		t.Fatalf("reloaded %+v %v", back, err)
	}
	back.Changes = nil
	if err := back.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(back.Path); !os.IsNotExist(err) {
		t.Fatalf("empty state should delete its file: %v", err)
	}
}
