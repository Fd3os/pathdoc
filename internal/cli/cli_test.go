package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Fd3os/pathdoc/internal/facts/factstest"
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
