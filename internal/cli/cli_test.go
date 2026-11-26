package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fd3os/pathdoc/internal/facts/factstest"
	"github.com/Fd3os/pathdoc/internal/fix"
)

func run(sys *factstest.Fake, args ...string) (code int, stdout, stderr string) {
	var o, e bytes.Buffer
	code = Run(args, Env{Sys: sys, Stdin: strings.NewReader(""), Stdout: &o, Stderr: &e})
	return code, o.String(), e.String()
}

func dartWithJoseTemp() *factstest.Fake {
	s := factstest.New()
	s.Env["PATH"] = `C:\tools\dart-sdk\bin`
	s.Files[`C:\tools\dart-sdk\bin\dart.exe`] = ""
	s.Files[`C:\tools\dart-sdk\version`] = "3.13.1"
	s.Env["TEMP"] = `C:\Users\José\AppData\Local\Temp`
	return s
}

func TestScanJSONFindsAndExitsOne(t *testing.T) {
	code, out, _ := run(dartWithJoseTemp(), "--json")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	var got struct {
		Findings []struct {
			Rule      string   `json:"rule"`
			Offending []string `json:"offending"`
			Fix       struct {
				Env map[string]string `json:"env"`
			} `json:"fix"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(got.Findings) != 1 || got.Findings[0].Rule != "dart-temp" || got.Findings[0].Offending[0] != "José" ||
		got.Findings[0].Fix.Env["TEMP"] != `C:\dev\temp` || got.Findings[0].Fix.Env["TMP"] != `C:\dev\temp` {
		t.Fatalf("findings = %+v", got.Findings)
	}
}

func TestScanCleanExitsZero(t *testing.T) {
	code, out, _ := run(factstest.New())
	if code != 0 || !strings.Contains(out, "nothing to check") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if code, _, _ := run(factstest.New(), "scan", "--json"); code != 0 {
		t.Fatalf("scan --json on a clean machine: exit %d", code)
	}
}

func TestScanTextIsPlainWithoutColor(t *testing.T) {
	_, out, _ := run(dartWithJoseTemp())
	if strings.Contains(out, "\x1b[") || !strings.Contains(out, "pathdoc fix") {
		t.Fatalf("text:\n%s", out)
	}
}

func TestBadInputExitsTwo(t *testing.T) {
	if code, _, errOut := run(factstest.New(), "--nope"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Errorf("unknown flag: exit %d, stderr %q", code, errOut)
	}
	if code, _, errOut := run(factstest.New(), "frobnicate"); code != 2 || !strings.Contains(errOut, "usage") {
		t.Errorf("unknown command: exit %d, stderr %q", code, errOut)
	}
}

func TestVersionAndHelp(t *testing.T) {
	if code, out, _ := run(factstest.New(), "--version"); code != 0 || !strings.HasPrefix(out, "pathdoc ") {
		t.Errorf("--version: %d %q", code, out)
	}
	if code, out, _ := run(factstest.New(), "help"); code != 0 || !strings.Contains(out, "pathdoc fix") {
		t.Errorf("help: %d %q", code, out)
	}
}

func TestHereCommand(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "متجر")
	os.MkdirAll(filepath.Join(proj, "android"), 0o755)
	root := filepath.Join(base, "dev")
	link := filepath.Join(root, "p", "mtjr")
	t.Cleanup(func() { os.Remove(link) })
	s := factstest.New()
	s.Cwd = filepath.Join(proj, "android")
	s.Files[filepath.Join(proj, "pubspec.yaml")] = ""
	s.Files[filepath.Join(proj, "package.json")] = ""
	s.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	code, out, errOut := run(s, "here", "--root", root)
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	sub := filepath.Join(link, "android")
	for _, want := range []string{`Set-Location -LiteralPath '` + sub + `'`, `cd /d "` + sub + `"`, "Git Bash", "npm"} {
		if !strings.Contains(out, want) {
			t.Errorf("here output lacks %q\n%s", want, out)
		}
	}
	if _, err := os.Stat(fix.StatePath(filepath.Join(base, "local"))); err != nil {
		t.Errorf("no undo record: %v", err)
	}

	plain := factstest.New()
	plain.Cwd = filepath.Join(base, "plain")
	plain.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	if code, out, _ := run(plain, "here", "--root", root); code != 0 || !strings.Contains(out, "plain ASCII") {
		t.Errorf("ASCII project: exit %d\n%s", code, out)
	}
	if code, _, errOut := run(s, "here", "--root", `C:\my dev`); code != 2 || !strings.Contains(errOut, "--root") {
		t.Errorf("bad root: exit %d, %q", code, errOut)
	}
}

type memStore struct{ vals map[string]fix.Value }

func (m *memStore) Get(n string) (fix.Value, error) { return m.vals[n], nil }
func (m *memStore) Set(n string, v fix.Value) error { m.vals[n] = v; return nil }
func (m *memStore) Delete(n string) error           { delete(m.vals, n); return nil }
func (m *memStore) Broadcast()                      {}

func fixRun(sys *factstest.Fake, store *memStore, stdin string, interactive bool, args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(args, Env{Sys: sys, Store: store, Stdin: strings.NewReader(stdin), Interactive: interactive, Stdout: &o, Stderr: &e})
	return code, o.String(), e.String()
}

func TestFixAsksAppliesAndUndoPutsBack(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "dev")
	sys := dartWithJoseTemp()
	sys.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	old := fix.Value{Exists: true, Data: `%USERPROFILE%\AppData\Local\Temp`, Expand: true}
	store := &memStore{vals: map[string]fix.Value{"TEMP": old, "TMP": old}}

	code, out, errOut := fixRun(sys, store, "y\n", true, "fix", "--root", root)
	if code != 0 {
		t.Fatalf("fix: exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{"TEMP", filepath.Join(root, "temp"), "Apply these changes? [y/N]", "new terminal"} {
		if !strings.Contains(out, want) {
			t.Errorf("fix output lacks %q\n%s", want, out)
		}
	}
	if store.vals["TEMP"].Data != filepath.Join(root, "temp") {
		t.Fatalf("TEMP = %+v", store.vals["TEMP"])
	}

	code, out, errOut = fixRun(sys, store, "", false, "undo")
	if code != 0 || store.vals["TEMP"] != old || store.vals["TMP"] != old {
		t.Fatalf("undo: exit %d, TEMP %+v\n%s%s", code, store.vals["TEMP"], out, errOut)
	}
	if code, out, _ := fixRun(sys, store, "", false, "undo"); code != 0 || !strings.Contains(out, "nothing to undo") {
		t.Fatalf("second undo: exit %d\n%s", code, out)
	}
}

func TestFixDeclinedChangesNothing(t *testing.T) {
	sys := dartWithJoseTemp()
	sys.Env["LOCALAPPDATA"] = t.TempDir()
	store := &memStore{vals: map[string]fix.Value{}}
	if code, _, _ := fixRun(sys, store, "n\n", true, "fix", "--root", filepath.Join(t.TempDir(), "dev")); code != 1 || len(store.vals) != 0 {
		t.Fatalf("declined: exit %d, store %+v", code, store.vals)
	}
}

func TestFixRefusesAPipeWithoutYes(t *testing.T) {
	sys := dartWithJoseTemp()
	sys.Env["LOCALAPPDATA"] = t.TempDir()
	store := &memStore{vals: map[string]fix.Value{}}
	code, _, errOut := fixRun(sys, store, "y\n", false, "fix", "--root", filepath.Join(t.TempDir(), "dev"))
	if code != 2 || !strings.Contains(errOut, "--yes") || len(store.vals) != 0 {
		t.Fatalf("pipe without --yes: exit %d, %q, store %+v", code, errOut, store.vals)
	}
	if code, _, _ := fixRun(sys, store, "", false, "fix", "--yes", "--root", filepath.Join(t.TempDir(), "dev")); code != 0 || !store.vals["TEMP"].Exists {
		t.Fatalf("--yes: exit %d, store %+v", code, store.vals)
	}
}

func TestFixWithNothingToDo(t *testing.T) {
	sys := factstest.New()
	sys.Env["LOCALAPPDATA"] = t.TempDir()
	if code, out, _ := fixRun(sys, &memStore{vals: map[string]fix.Value{}}, "", false, "fix"); code != 0 || !strings.Contains(out, "nothing for pathdoc fix") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}

// A folder name with ' $ or [ ] must survive the copy-paste line.
func TestHereQuotesAnOddSubfolder(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "متجر")
	sub := filepath.Join(proj, "it's $x [1]")
	os.MkdirAll(sub, 0o755)
	root := filepath.Join(base, "dev")
	t.Cleanup(func() { os.Remove(filepath.Join(root, "p", "mtjr")) })
	s := factstest.New()
	s.Cwd = sub
	s.Files[filepath.Join(proj, "pubspec.yaml")] = ""
	s.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	code, out, errOut := run(s, "here", "--root", root)
	want := `Set-Location -LiteralPath '` + filepath.Join(root, "p", "mtjr", "it''s $x [1]") + `'`
	if code != 0 || !strings.Contains(out, want) {
		t.Fatalf("exit %d, want %q in\n%s%s", code, want, out, errOut)
	}
}

func TestFixSaysWhatItChangesBeyondTheTool(t *testing.T) {
	base := t.TempDir()
	sys := dartWithJoseTemp()
	sys.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	store := &memStore{vals: map[string]fix.Value{}}
	_, out, _ := fixRun(sys, store, "n\n", true, "fix", "--root", filepath.Join(base, "dev"))
	if !strings.Contains(out, "every program you start") {
		t.Fatalf("fix must say TEMP is user-wide:\n%s", out)
	}
}

func TestFixNamesRootWhenItCannotMakeIt(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "file")
	os.WriteFile(blocker, []byte("x"), 0o644)
	sys := dartWithJoseTemp()
	sys.Env["LOCALAPPDATA"] = filepath.Join(base, "local")
	store := &memStore{vals: map[string]fix.Value{}}
	code, _, errOut := fixRun(sys, store, "", false, "fix", "--yes", "--root", filepath.Join(blocker, "dev"))
	if code != 2 || !strings.Contains(errOut, "--root") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}
