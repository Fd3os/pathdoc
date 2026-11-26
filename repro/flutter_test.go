//go:build repro

package repro

import (
	"path/filepath"
	"testing"

	"github.com/Fd3os/pathdoc/internal/rules"
)

// flutter-analyze: before Flutter 3.50.0 the tool counted the LSP
// Content-Length in characters, not bytes, so any non-ASCII letter in the
// project path cut the analysis server's first message short. A junction
// cures it; so does 3.50.0 (flutter/flutter#191377), which this test checks
// too when it runs on a newer Flutter.
func TestRepro_flutter_analyze(t *testing.T) {
	fl := tool(t, "flutter")
	fixed := !rules.Less(fl.Version, "3.50.0")
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "fa")
		write(t, filepath.Join(dir, "pubspec.yaml"), "name: demo\nenvironment:\n  sdk: ^3.0.0\n")
		write(t, filepath.Join(dir, "lib", "main.dart"), "int add(int a, int b) => a + b;\n")
		analyze := func(root string) result { return run(t, root, nil, fl.Path, "analyze") }
		if f.key == "plain" || fixed {
			pass(t, f.name, analyze(dir), "No issues found")
			continue
		}
		fail(t, f.name, analyze(dir), "FormatException")
		link := junction(t, filepath.Join(base, "jf-"+f.key), dir)
		pass(t, f.name+" through a junction", analyze(link), "No issues found")
	}
}
