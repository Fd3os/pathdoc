//go:build repro

package repro

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/win"
)

type folder struct{ key, name string }

var folders = []folder{{"plain", "plain"}, {"latin", "café"}, {"nonacp", "ابوي"}}

// lab makes an empty ASCII base with no spaces for one test.
func lab(t *testing.T) string {
	t.Helper()
	if cp := win.ActiveCodePage(); win.Represents("ابوي", cp) {
		if os.Getenv("PATHDOC_REPRO_STRICT") == "1" {
			t.Fatalf("code page %d holds Arabic; the lab needs one that does not (1252 does not)", cp)
		}
		t.Skipf("code page %d holds Arabic; the lab needs one that does not (1252 does not)", cp)
	}
	base := filepath.Join(os.Getenv("SystemDrive")+`\`, "pd-repro", strings.ReplaceAll(t.Name(), "/", "_"))
	removeTree(base)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeTree(base) })
	return base
}

// removeTree deletes a lab folder, taking junctions out first so nothing is
// ever deleted through one.
func removeTree(dir string) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if _, err := win.JunctionTarget(p); err == nil {
			os.Remove(p)
			continue
		}
		if e.IsDir() {
			removeTree(p)
			continue
		}
		os.Remove(p)
	}
	os.Remove(dir)
}

// junction links link to target for the rest of the test.
func junction(t *testing.T, link, target string) string {
	t.Helper()
	if err := win.CreateJunction(link, target); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(link) })
	return link
}

// tool finds a tool the way pathdoc does (skipping the Store's placeholder
// python.exe), or skips — or fails, when strict.
func tool(t *testing.T, key string) facts.Tool {
	t.Helper()
	tl, ok := facts.Gather(facts.Real()).Tools[key]
	if !ok {
		if os.Getenv("PATHDOC_REPRO_STRICT") == "1" {
			t.Fatalf("%s is not installed", key)
		}
		t.Skipf("%s is not installed", key)
	}
	t.Logf("%s %s at %s", tl.Name, tl.Version, tl.Path)
	return tl
}

type result struct {
	code int
	out  string
}

// run runs a command with its output piped (the way an IDE or CI does),
// env entries overriding the inherited ones; an entry "NAME" alone removes it.
func run(t *testing.T, dir string, env []string, name string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = environ(env)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return result{exit.ExitCode(), string(out)}
	case err != nil:
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return result{0, string(out)}
}

func environ(over []string) []string {
	drop := map[string]bool{}
	var add []string
	for _, kv := range over {
		k, _, isSet := strings.Cut(kv, "=")
		drop[strings.ToUpper(k)] = true
		if isSet {
			add = append(add, kv)
		}
	}
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[strings.ToUpper(k)] {
			env = append(env, kv)
		}
	}
	return append(env, add...)
}

func pass(t *testing.T, what string, r result, want string) {
	t.Helper()
	if r.code != 0 || !strings.Contains(r.out, want) {
		t.Errorf("%s: want success with %q, got exit %d\n%s", what, want, r.code, tail(r.out))
	}
}

func fail(t *testing.T, what string, r result, symptom string) {
	t.Helper()
	if r.code == 0 || !strings.Contains(r.out, symptom) {
		t.Errorf("%s: want failure with %q, got exit %d\n%s", what, symptom, r.code, tail(r.out))
	}
}

func tail(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return strings.Join(lines, "\n")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		to := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(to)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
