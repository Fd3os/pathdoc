package fix

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/rules"
	"github.com/Fd3os/pathdoc/internal/win"
)

type fakeStore struct {
	vals       map[string]Value
	broadcasts int
}

func newStore() *fakeStore                       { return &fakeStore{vals: map[string]Value{}} }
func (s *fakeStore) Get(n string) (Value, error) { return s.vals[n], nil }
func (s *fakeStore) Set(n string, v Value) error { s.vals[n] = v; return nil }
func (s *fakeStore) Delete(n string) error       { delete(s.vals, n); return nil }
func (s *fakeStore) Broadcast()                  { s.broadcasts++ }

func finding(id string) rules.Finding {
	for _, r := range rules.Catalog {
		if r.ID == id {
			return rules.Finding{Rule: r, Where: facts.Location{Key: r.Locations[0]}}
		}
	}
	panic(id)
}

func all() []rules.Finding {
	return []rules.Finding{finding("gradle-user-home"), finding("dart-temp"), finding("flutter-analyze"), finding("python-pipe-encoding")}
}

func userTemp() Value {
	return Value{Exists: true, Data: `%USERPROFILE%\AppData\Local\Temp`, Expand: true}
}

func TestPlanListsEachVariableOnce(t *testing.T) {
	root := `C:\dev`
	steps, err := Plan(all(), root, newStore())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range steps {
		got = append(got, s.Var+"="+s.New.Data+"|"+s.Dir)
	}
	want := []string{`GRADLE_USER_HOME=C:\dev\gradle|C:\dev\gradle`, `TEMP=C:\dev\temp|C:\dev\temp`, `TMP=C:\dev\temp|C:\dev\temp`, `PYTHONUTF8=1|`}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("steps:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	store := newStore()
	store.vals["GRADLE_USER_HOME"] = Value{Exists: true, Data: `C:\dev\gradle`}
	steps, _ = Plan(all(), root, store)
	for _, s := range steps {
		if s.Var == "GRADLE_USER_HOME" {
			t.Fatal("already set to the target: want no step")
		}
	}
	if _, err := Plan(all(), `C:\my dev`, newStore()); err == nil {
		t.Fatal("bad root: want an error")
	}
}

func TestApplyThenUndoRestoresExactly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dev")
	store := newStore()
	store.vals["TEMP"], store.vals["TMP"] = userTemp(), userTemp()
	st := &State{}
	steps, _ := Plan(all(), root, store)
	if err := Apply(steps, st, store); err != nil {
		t.Fatal(err)
	}
	if store.vals["TEMP"].Data != filepath.Join(root, "temp") || store.vals["PYTHONUTF8"].Data != "1" {
		t.Fatalf("after apply: %+v", store.vals)
	}
	if _, err := os.Stat(filepath.Join(root, "gradle")); err != nil {
		t.Fatalf("gradle folder not made: %v", err)
	}
	var out bytes.Buffer
	skipped, err := Undo(st, store, false, &out)
	if err != nil || skipped != 0 {
		t.Fatalf("undo: skipped %d, %v\n%s", skipped, err, out.String())
	}
	if store.vals["TEMP"] != userTemp() || store.vals["TMP"] != userTemp() {
		t.Fatalf("TEMP not restored exactly: %+v", store.vals["TEMP"])
	}
	if _, ok := store.vals["GRADLE_USER_HOME"]; ok {
		t.Fatal("GRADLE_USER_HOME did not exist before: undo must delete it")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("empty folders pathdoc made should be gone: %v", err)
	}
	if len(st.Changes) != 0 || store.broadcasts < 2 {
		t.Fatalf("changes left %d, broadcasts %d", len(st.Changes), store.broadcasts)
	}
}

func TestFixTwiceKeepsTheFirstOld(t *testing.T) {
	base := t.TempDir()
	store := newStore()
	store.vals["TEMP"] = userTemp()
	st := &State{}
	temp := []rules.Finding{finding("dart-temp")}
	steps, _ := Plan(temp, filepath.Join(base, "a"), store)
	Apply(steps, st, store)
	steps, _ = Plan(temp, filepath.Join(base, "b"), store)
	Apply(steps, st, store)
	for _, c := range st.Changes {
		if c.Kind == "env" && c.Name == "TEMP" && (c.Old != userTemp() || c.New != filepath.Join(base, "b", "temp")) {
			t.Fatalf("TEMP change = %+v", c)
		}
	}
	Undo(st, store, false, &bytes.Buffer{})
	if store.vals["TEMP"] != userTemp() {
		t.Fatalf("after two fixes and an undo TEMP = %+v", store.vals["TEMP"])
	}
}

func TestUndoLeavesAValueTheUserChanged(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dev")
	store := newStore()
	store.vals["TEMP"] = userTemp()
	st := &State{}
	steps, _ := Plan([]rules.Finding{finding("dart-temp")}, root, store)
	Apply(steps, st, store)
	store.vals["TEMP"] = Value{Exists: true, Data: `D:\mine`}
	var out bytes.Buffer
	skipped, err := Undo(st, store, false, &out)
	if err != nil || skipped != 1 || store.vals["TEMP"].Data != `D:\mine` {
		t.Fatalf("skipped %d, err %v, TEMP %+v\n%s", skipped, err, store.vals["TEMP"], out.String())
	}
	if !strings.Contains(out.String(), `D:\mine`) || len(st.Changes) != 1 {
		t.Fatalf("want the skip explained and the entry kept; changes %d\n%s", len(st.Changes), out.String())
	}
	if _, err := Undo(st, store, true, &bytes.Buffer{}); err != nil || store.vals["TEMP"] != userTemp() || len(st.Changes) != 0 {
		t.Fatalf("force: TEMP %+v, changes %d, %v", store.vals["TEMP"], len(st.Changes), err)
	}
}

func TestUndoKeepsAFolderThatIsNotEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dev")
	store := newStore()
	st := &State{}
	steps, _ := Plan([]rules.Finding{finding("gradle-user-home")}, root, store)
	Apply(steps, st, store)
	cache := filepath.Join(root, "gradle", "caches.bin")
	os.WriteFile(cache, []byte("x"), 0o644)
	var out bytes.Buffer
	if _, err := Undo(st, store, false, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("undo deleted a file in a folder it made: %v", err)
	}
	if !strings.Contains(out.String(), "kept") || len(st.Changes) != 0 {
		t.Fatalf("changes %d\n%s", len(st.Changes), out.String())
	}
}

func TestUndoRemovesADanglingJunction(t *testing.T) {
	_, proj, root := project(t, "متجر")
	st := &State{}
	link, _, err := Here(proj, proj, root, "", st)
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(proj)
	var out bytes.Buffer
	if _, err := Undo(st, newStore(), false, &out); err != nil {
		t.Fatalf("undo: %v\n%s", err, out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("dangling junction still there: %v", err)
	}
	if len(st.Changes) != 0 {
		t.Fatalf("changes left: %+v", st.Changes)
	}
}

func TestFoldersFixMakesArePrivate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dev")
	store := newStore()
	steps, _ := Plan([]rules.Finding{finding("gradle-user-home"), finding("dart-temp")}, root, store)
	if err := Apply(steps, &State{}, store); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{root, filepath.Join(root, "gradle"), filepath.Join(root, "temp")} {
		if ok, err := win.IsPrivate(d); err != nil || !ok {
			t.Errorf("%s: private = %v, %v; other accounts could write into it", d, ok, err)
		}
	}
}
