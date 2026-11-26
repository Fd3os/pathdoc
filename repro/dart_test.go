//go:build repro

package repro

import (
	"os"
	"path/filepath"
	"testing"
)

const helloDart = "void main() { print('hello-ok'); }\n"

// dart-temp: dart compile exe reads its intermediate program.dill back from
// TEMP; any non-ASCII letter there (café too) makes the read fail. An ASCII
// TEMP — what pathdoc fix sets — cures it.
func TestRepro_dart_temp(t *testing.T) {
	dart := tool(t, "dart").Path
	base := lab(t)
	proj := filepath.Join(base, "project")
	write(t, filepath.Join(proj, "hello.dart"), helloDart)
	for _, f := range folders {
		tmp := filepath.Join(base, f.name, "tmp")
		os.MkdirAll(tmp, 0o755)
		r := run(t, proj, []string{"TEMP=" + tmp, "TMP=" + tmp}, dart, "compile", "exe", "hello.dart", "-o", "hello.exe")
		if f.key == "plain" {
			pass(t, f.name+" TEMP", r, "hello.exe")
		} else {
			fail(t, f.name+" TEMP", r, "Generating AOT snapshot failed")
		}
	}
}

// dart-aot-output: writing an AOT snapshot into a folder with any non-ASCII
// letter fails; through an ASCII junction it works.
func TestRepro_dart_aot_output(t *testing.T) {
	dart := tool(t, "dart").Path
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "aot")
		write(t, filepath.Join(dir, "hello.dart"), helloDart)
		snapshot := func(root string) result {
			return run(t, root, nil, dart, "compile", "aot-snapshot", "hello.dart", "-o", "hello.aot")
		}
		if f.key == "plain" {
			pass(t, f.name, snapshot(dir), "hello.aot")
			continue
		}
		fail(t, f.name, snapshot(dir), "Generating AOT snapshot failed")
		link := junction(t, filepath.Join(base, "ja-"+f.key), dir)
		pass(t, f.name+" through a junction", snapshot(link), "hello.aot")
	}
}
