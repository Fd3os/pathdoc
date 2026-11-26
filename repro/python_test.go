//go:build repro

package repro

import (
	"path/filepath"
	"testing"

	"github.com/Fd3os/pathdoc/internal/rules"
)

// python-pipe-encoding: with its output piped, Python before 3.15 encodes
// stdout in the ANSI code page; pip prints its own path, and a path the code
// page cannot hold crashes it. PYTHONUTF8=1 — what pathdoc fix sets — cures
// it; Python 3.15 made it the default (PEP 686).
func TestRepro_python_pipe_encoding(t *testing.T) {
	py := tool(t, "python")
	fixed := !rules.Less(py.Version, "3.15")
	base := lab(t)
	off := []string{"PYTHONUTF8=0", "PYTHONIOENCODING"}
	for _, f := range folders {
		venv := filepath.Join(base, f.name, "venv")
		if r := run(t, base, off, py.Path, "-m", "venv", venv); r.code != 0 {
			t.Fatalf("make venv in %s: exit %d\n%s", f.name, r.code, tail(r.out))
		}
		pip := filepath.Join(venv, "Scripts", "pip.exe")
		r := run(t, base, off, pip, "--version")
		if f.key != "nonacp" || fixed {
			pass(t, f.name, r, "pip ")
			continue
		}
		fail(t, f.name, r, "UnicodeEncodeError")
		pass(t, f.name+" with PYTHONUTF8=1", run(t, base, []string{"PYTHONUTF8=1", "PYTHONIOENCODING"}, pip, "--version"), "pip ")
	}
}
