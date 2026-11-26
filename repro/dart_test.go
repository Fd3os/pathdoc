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

// dart-aot-output, the way most Flutter apps meet it: build_runner (used by
// json_serializable, freezed, riverpod_generator…) compiles its build script
// to an AOT snapshot under the project's .dart_tool, so any non-ASCII letter
// in the project path fails the build; through an ASCII junction it works.
func TestRepro_dart_build_runner(t *testing.T) {
	dart := tool(t, "dart").Path
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "br")
		write(t, filepath.Join(dir, "pubspec.yaml"), "name: br\nenvironment:\n  sdk: ^3.0.0\ndev_dependencies:\n  build_runner: any\n")
		write(t, filepath.Join(dir, "lib", "br.dart"), "int answer = 42;\n")
		build := func(root string) result {
			if r := run(t, root, nil, dart, "pub", "get"); r.code != 0 {
				t.Fatalf("%s: dart pub get failed\n%s", f.name, tail(r.out))
			}
			return run(t, root, nil, dart, "run", "build_runner", "build")
		}
		if f.key == "plain" {
			pass(t, f.name, build(dir), "Built with build_runner")
			continue
		}
		fail(t, f.name, build(dir), "Unable to write file")
		link := junction(t, filepath.Join(base, "jb-"+f.key), dir)
		pass(t, f.name+" through a junction", build(link), "Built with build_runner")
	}
}
