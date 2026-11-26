//go:build repro

package repro

import (
	"os"
	"path/filepath"
	"testing"
)

// android-project-path: the Android Gradle Plugin stops a build whose project
// path has any non-ASCII letter, even one the code page can write (café):
// "Your project path contains non-ASCII characters". In ابوي the Gradle
// wrapper fails before the plugin runs (gradle-project-path). From an ASCII
// junction both build.
func TestRepro_android_project_path(t *testing.T) {
	flutter := tool(t, "flutter").Path
	base := lab(t)
	for _, f := range folders {
		parent := filepath.Join(base, f.name)
		os.MkdirAll(parent, 0o755)
		if r := run(t, parent, nil, flutter, "create", "--project-name", "app", "--platforms", "android", "app"); r.code != 0 {
			t.Fatalf("%s: flutter create failed\n%s", f.name, tail(r.out))
		}
		app := filepath.Join(parent, "app")
		build := func(dir string) result { return run(t, dir, nil, flutter, "build", "apk", "--debug") }
		switch f.key {
		case "plain":
			pass(t, f.name, build(app), "Built build")
			continue
		case "latin":
			fail(t, f.name, build(app), "Your project path contains non-ASCII characters")
		default:
			fail(t, f.name, build(app), "GradleWrapperMain")
		}
		link := junction(t, filepath.Join(base, "jx-"+f.key), app)
		pass(t, f.name+" through a junction", build(link), "Built build")
	}
}
