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
	for _, want := range []string{`Set-Location "` + sub + `"`, `cd /d "` + sub + `"`, "Git Bash", "npm"} {
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
