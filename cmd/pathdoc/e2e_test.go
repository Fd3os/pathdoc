package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// The real binary, run from a folder named ابوي with its output piped the way
// an IDE or CI captures it: the bytes must be UTF-8 and the JSON must parse.
func TestBinaryPipedFromArabicFolder(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "pathdoc.exe")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	proj := filepath.Join(tmp, "ابوي")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pubspec.yaml"), []byte("name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "--json")
	cmd.Dir = proj
	out, err := cmd.Output()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code != 0 && code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !utf8.Valid(out) {
		t.Fatal("output is not UTF-8")
	}
	var got struct {
		Findings []struct {
			Rule string `json:"rule"`
			Path string `json:"path"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if (code == 1) != (len(got.Findings) > 0) {
		t.Fatalf("exit %d with %d findings", code, len(got.Findings))
	}
	t.Logf("exit %d, findings %+v", code, got.Findings)
}
