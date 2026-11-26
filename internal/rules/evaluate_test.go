package rules

import (
	"reflect"
	"testing"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/win"
)

func at(cp uint32) func(string) bool { return func(s string) bool { return win.Represents(s, cp) } }

func TestOffending(t *testing.T) {
	cp1252 := at(1252)
	if got := Offending(`C:\Users\محمد\.gradle`, NotInCodePage, cp1252); !reflect.DeepEqual(got, []string{"محمد"}) {
		t.Errorf("Arabic home = %v", got)
	}
	if got := Offending(`C:\café\x`, NotInCodePage, cp1252); len(got) != 0 {
		t.Errorf("café, not-in-code-page = %v, want none", got)
	}
	if got := Offending(`C:\café\x`, AnyNonASCII, cp1252); !reflect.DeepEqual(got, []string{"café"}) {
		t.Errorf("café, any-non-ASCII = %v", got)
	}
	if got := Offending(`C:\plain\dir`, AnyNonASCII, cp1252); len(got) != 0 {
		t.Errorf("plain = %v", got)
	}
	if got := Offending(`C:\ابوي\x\ابوي`, AnyNonASCII, cp1252); !reflect.DeepEqual(got, []string{"ابوي"}) {
		t.Errorf("repeated segment = %v, want it once", got)
	}
}

func TestLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"3.47.1", "3.50.0", true}, {"3.9", "3.15", true}, {"3.50.0", "3.50.0", false},
		{"3.50", "3.50.0", false}, {"3.50.1", "3.50.0", false}, {"", "1", false}, {"3.12", "3.15", true},
	}
	for _, c := range cases {
		if got := Less(c.a, c.b); got != c.want {
			t.Errorf("Less(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// machine builds facts like Gather would: code page 1252, the given tools,
// locations and project markers.
func machine(tools map[string]string, locs map[string]facts.Location, markers ...string) *facts.Facts {
	f := &facts.Facts{CodePage: 1252, Represents: at(1252), Tools: map[string]facts.Tool{}, Locations: map[string]facts.Location{}, Markers: map[string]bool{}}
	for k, v := range tools {
		f.Tools[k] = facts.Tool{Name: k, Version: v}
	}
	for k, l := range locs {
		l.Key = k
		f.Locations[k] = l
		if k == "project" {
			f.Project = l.Path
		}
	}
	for _, m := range markers {
		f.Markers[m] = true
	}
	return f
}

func fired(f *facts.Facts) []string {
	var ids []string
	for _, fd := range Evaluate(f, Catalog) {
		ids = append(ids, fd.Rule.ID)
	}
	return ids
}

func has(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func TestGradleProjectPath(t *testing.T) {
	proj := map[string]facts.Location{"project": {Path: `C:\Users\x\ابوي\app`}}
	if !has(fired(machine(map[string]string{"java": "17"}, proj, "android/gradlew.bat")), "gradle-project-path") {
		t.Error("Arabic project with android/gradlew.bat and Java: want gradle-project-path")
	}
	latin := map[string]facts.Location{"project": {Path: `C:\Users\x\café\app`}}
	if has(fired(machine(map[string]string{"java": "17"}, latin, "gradlew.bat")), "gradle-project-path") {
		t.Error("café project: the lab passed, want silence")
	}
	if has(fired(machine(nil, proj, "gradlew.bat")), "gradle-project-path") {
		t.Error("no Java: want silence")
	}
	if has(fired(machine(map[string]string{"java": "17"}, proj, "pubspec.yaml")), "gradle-project-path") {
		t.Error("no Gradle wrapper: want silence")
	}
	// A Flutter project ignores gradlew.bat in git; Flutter writes it on the
	// first Android build — the build that then fails.
	for _, settings := range []string{"android/settings.gradle.kts", "android/settings.gradle"} {
		if !has(fired(machine(map[string]string{"java": "17"}, proj, "pubspec.yaml", settings)), "gradle-project-path") {
			t.Errorf("fresh Flutter clone with %s: want gradle-project-path", settings)
		}
	}
}

func TestGradleUserHome(t *testing.T) {
	home := map[string]facts.Location{"gradle-home": {Path: `C:\Users\محمد\.gradle`, Default: true}}
	if !has(fired(machine(map[string]string{"java": "17", "flutter": "3.47.1"}, home)), "gradle-user-home") {
		t.Error("Java + Flutter + Arabic home: want gradle-user-home")
	}
	if has(fired(machine(map[string]string{"java": "17"}, home)), "gradle-user-home") {
		t.Error("Java only, default home that does not exist: want silence")
	}
	used := map[string]facts.Location{"gradle-home": {Path: `C:\Users\محمد\.gradle`, Default: true, Exists: true}}
	if !has(fired(machine(map[string]string{"java": "17"}, used)), "gradle-user-home") {
		t.Error("Java + existing Arabic home: want gradle-user-home")
	}
	set := map[string]facts.Location{"gradle-home": {Path: `D:\ابوي`}}
	if !has(fired(machine(map[string]string{"java": "17"}, set)), "gradle-user-home") {
		t.Error("Java + GRADLE_USER_HOME set to Arabic: want gradle-user-home")
	}
}

func TestDartTemp(t *testing.T) {
	temp := map[string]facts.Location{"temp": {Path: `C:\Users\José\AppData\Local\Temp`}}
	if !has(fired(machine(map[string]string{"dart": "3.13.1"}, temp)), "dart-temp") {
		t.Error("Dart + José temp: want dart-temp (any non-ASCII breaks it)")
	}
	if !has(fired(machine(map[string]string{"flutter": "3.47.1"}, temp)), "dart-temp") {
		t.Error("Flutter only + José temp: want dart-temp")
	}
	if has(fired(machine(map[string]string{"java": "17"}, temp)), "dart-temp") {
		t.Error("no Dart: want silence")
	}
}

func TestDartAotOutput(t *testing.T) {
	proj := map[string]facts.Location{"project": {Path: `C:\work\café\app`}}
	if !has(fired(machine(map[string]string{"dart": "3.13.1"}, proj, "pubspec.yaml")), "dart-aot-output") {
		t.Error("Dart project in café: want dart-aot-output")
	}
	if has(fired(machine(map[string]string{"dart": "3.13.1"}, proj, "package.json")), "dart-aot-output") {
		t.Error("no pubspec.yaml: want silence")
	}
}

func TestFlutterAnalyze(t *testing.T) {
	proj := map[string]facts.Location{"project": {Path: `C:\work\café\app`}}
	if !has(fired(machine(map[string]string{"flutter": "3.47.1"}, proj, "pubspec.yaml")), "flutter-analyze") {
		t.Error("Flutter 3.47.1 in café: want flutter-analyze")
	}
	if has(fired(machine(map[string]string{"flutter": "3.50.0"}, proj, "pubspec.yaml")), "flutter-analyze") {
		t.Error("Flutter 3.50.0 has the fix: want silence")
	}
	if has(fired(machine(map[string]string{"flutter": ""}, proj, "pubspec.yaml")), "flutter-analyze") {
		t.Error("unknown Flutter version: want silence, not a guess")
	}
}

func TestPythonPipeEncoding(t *testing.T) {
	py := map[string]facts.Location{"python": {Path: `C:\Users\محمد\AppData\Local\Programs\Python\Python312`}}
	if !has(fired(machine(map[string]string{"python": "3.12"}, py)), "python-pipe-encoding") {
		t.Error("Python 3.12 under an Arabic home: want python-pipe-encoding")
	}
	if has(fired(machine(map[string]string{"python": "3.15"}, py)), "python-pipe-encoding") {
		t.Error("Python 3.15 is UTF-8 by default: want silence")
	}
	proj := map[string]facts.Location{"python": {Path: `C:\Python312`}, "project": {Path: `C:\work\ابوي`}}
	if !has(fired(machine(map[string]string{"python": "3.12"}, proj, "requirements.txt")), "python-pipe-encoding") {
		t.Error("ASCII Python, Arabic Python project: want python-pipe-encoding")
	}
	if has(fired(machine(map[string]string{"python": "3.12"}, proj, "pubspec.yaml")), "python-pipe-encoding") {
		t.Error("Arabic project that is not a Python project: want silence")
	}
}

func TestCodePage65001SilencesCodePageRules(t *testing.T) {
	f := machine(map[string]string{"java": "17", "flutter": "3.47.1", "python": "3.12"}, map[string]facts.Location{
		"gradle-home": {Path: `C:\Users\محمد\.gradle`, Exists: true},
		"project":     {Path: `C:\Users\محمد\app`},
		"python":      {Path: `C:\Users\محمد\py`},
	}, "android/gradlew.bat", "pubspec.yaml")
	f.CodePage, f.Represents = 65001, at(65001)
	ids := fired(f)
	for _, id := range []string{"gradle-project-path", "gradle-user-home", "python-pipe-encoding"} {
		if has(ids, id) {
			t.Errorf("code page 65001: %s fired", id)
		}
	}
	if !has(ids, "flutter-analyze") {
		t.Error("code page 65001: any-non-ASCII rules still fire; want flutter-analyze")
	}
}

func TestFindingCarriesWhereAndOffending(t *testing.T) {
	f := machine(map[string]string{"dart": "3.13.1"}, map[string]facts.Location{"temp": {Label: "Temp folder", Path: `C:\Users\José\Temp`, EnvVar: "TEMP"}})
	fs := Evaluate(f, Catalog)
	if len(fs) != 1 || fs[0].Where.EnvVar != "TEMP" || !reflect.DeepEqual(fs[0].Offending, []string{"José"}) {
		t.Fatalf("findings = %+v", fs)
	}
}
