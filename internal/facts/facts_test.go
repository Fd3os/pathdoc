package facts_test

import (
	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/facts/factstest"
	"testing"
	"time"
)

func TestGatherDefaultsGradleHome(t *testing.T) {
	s := factstest.New()
	s.Env["USERPROFILE"] = `C:\Users\محمد`
	got := facts.Gather(s).Locations["gradle-home"]
	want := facts.Location{Key: "gradle-home", Label: "Gradle home", Path: `C:\Users\محمد\.gradle`, EnvVar: "GRADLE_USER_HOME", Default: true}
	if got != want {
		t.Fatalf("gradle-home = %+v, want %+v", got, want)
	}
	s.Env["GRADLE_USER_HOME"] = `D:\g`
	if got := facts.Gather(s).Locations["gradle-home"]; got.Path != `D:\g` || got.Default || got.Exists {
		t.Fatalf("set GRADLE_USER_HOME: got %+v", got)
	}
	s.Dirs[`D:\g`] = true
	if got := facts.Gather(s).Locations["gradle-home"]; !got.Exists {
		t.Fatalf("existing Gradle home: got %+v", got)
	}
}

func TestGatherTemp(t *testing.T) {
	s := factstest.New()
	s.Env["TMP"] = `C:\t2`
	if got := facts.Gather(s).Locations["temp"]; got.Path != `C:\t2` || got.EnvVar != "TMP" {
		t.Fatalf("TMP only: got %+v", got)
	}
	s.Env["TEMP"] = `C:\Users\محمد\AppData\Local\Temp`
	got := facts.Gather(s).Locations["temp"]
	if got.Path != `C:\Users\محمد\AppData\Local\Temp` || got.EnvVar != "TEMP" || got.Label != "Temp folder" {
		t.Fatalf("TEMP: got %+v", got)
	}
	if _, ok := facts.Gather(factstest.New()).Locations["temp"]; ok {
		t.Fatal("no TEMP or TMP: want no temp location")
	}
}

func TestGatherSkipsPythonStub(t *testing.T) {
	s := factstest.New()
	stub := `C:\Users\محمد\AppData\Local\Microsoft\WindowsApps`
	s.Env["PATH"] = stub + `;"C:\Python312"`
	s.Files[stub+`\python.exe`] = ""
	s.Files[`C:\Python312\python.exe`] = ""
	s.Vers[`C:\Python312\python.exe`] = "3.12.3150.1013"
	f := facts.Gather(s)
	if py := f.Tools["python"]; py.Path != `C:\Python312\python.exe` || py.Version != "3.12" {
		t.Fatalf("python = %+v", py)
	}
	if loc := f.Locations["python"]; loc.Path != `C:\Python312` {
		t.Fatalf("python location = %+v", loc)
	}

	only := factstest.New()
	only.Env["PATH"] = stub
	only.Files[stub+`\python.exe`] = ""
	if py, ok := facts.Gather(only).Tools["python"]; ok {
		t.Fatalf("only the Store stub: want no python, got %+v", py)
	}
}

func TestGatherJava(t *testing.T) {
	s := factstest.New()
	javapath := `C:\Program Files (x86)\Common Files\Oracle\Java\java8path`
	s.Env["PATH"] = javapath
	s.Files[javapath+`\java.exe`] = ""
	s.Vers[javapath+`\java.exe`] = "8.0.5010.8"
	if j := facts.Gather(s).Tools["java"]; j.Path != javapath+`\java.exe` || j.Version != "8.0.5010" {
		t.Fatalf("java from PATH = %+v", j)
	}
	home := `C:\jdk-17`
	s.Env["JAVA_HOME"] = home
	s.Files[home+`\bin\java.exe`] = ""
	s.Files[home+`\release`] = "IMPLEMENTOR=\"x\"\nJAVA_VERSION=\"17.0.20\"\n"
	if j := facts.Gather(s).Tools["java"]; j.Path != home+`\bin\java.exe` || j.Version != "17.0.20" {
		t.Fatalf("java from JAVA_HOME = %+v", j)
	}
}

func TestGatherFlutterAndDart(t *testing.T) {
	s := factstest.New()
	s.Env["PATH"] = `C:\flutter\bin`
	s.Env["PATHEXT"] = ".COM;.EXE;.BAT;.CMD"
	s.Files[`C:\flutter\bin\flutter.bat`] = ""
	s.Files[`C:\flutter\bin\dart.bat`] = ""
	s.Files[`C:\flutter\bin\cache\flutter.version.json`] = `{"frameworkVersion": "3.47.1", "channel": "stable"}`
	s.Files[`C:\flutter\bin\cache\dart-sdk\version`] = "3.13.1\n"
	f := facts.Gather(s)
	if fl := f.Tools["flutter"]; fl.Path != `C:\flutter\bin\flutter.bat` || fl.Version != "3.47.1" {
		t.Fatalf("flutter = %+v", fl)
	}
	if d := f.Tools["dart"]; d.Version != "3.13.1" {
		t.Fatalf("dart in flutter = %+v", d)
	}

	old := factstest.New()
	old.Env["PATH"] = `C:\flutter\bin`
	old.Files[`C:\flutter\bin\flutter.bat`] = ""
	old.Files[`C:\flutter\version`] = "3.22.0\n"
	if fl := facts.Gather(old).Tools["flutter"]; fl.Version != "3.22.0" {
		t.Fatalf("flutter from version file = %+v", fl)
	}

	alone := factstest.New()
	alone.Env["PATH"] = `C:\tools\dart-sdk\bin`
	alone.Files[`C:\tools\dart-sdk\bin\dart.exe`] = ""
	alone.Files[`C:\tools\dart-sdk\version`] = "3.9.0"
	if d := facts.Gather(alone).Tools["dart"]; d.Path != `C:\tools\dart-sdk\bin\dart.exe` || d.Version != "3.9.0" {
		t.Fatalf("standalone dart = %+v", d)
	}
}

func TestGatherProjectWalksUp(t *testing.T) {
	s := factstest.New()
	s.Cwd = `C:\Users\محمد\app\lib`
	s.Files[`C:\Users\محمد\app\pubspec.yaml`] = ""
	s.Files[`C:\Users\محمد\app\android\gradlew.bat`] = ""
	f := facts.Gather(s)
	if f.Project != `C:\Users\محمد\app` {
		t.Fatalf("Project = %q", f.Project)
	}
	if !f.Markers["pubspec.yaml"] || !f.Markers["android/gradlew.bat"] || f.Markers["gradlew.bat"] {
		t.Fatalf("Markers = %v", f.Markers)
	}
	if loc := f.Locations["project"]; loc.Path != `C:\Users\محمد\app` || loc.Label != "Project" {
		t.Fatalf("project location = %+v", loc)
	}

	clone := factstest.New()
	clone.Cwd = `C:\src\app`
	clone.Files[`C:\src\app\android\settings.gradle.kts`] = ""
	if cf := facts.Gather(clone); cf.Project != `C:\src\app` || !cf.Markers["android/settings.gradle.kts"] {
		t.Fatalf("Android settings marker: Project %q, Markers %v", cf.Project, cf.Markers)
	}

	none := factstest.New()
	none.Cwd = `C:\Users\محمد\Desktop`
	nf := facts.Gather(none)
	if nf.Project != "" {
		t.Fatalf("no markers: Project = %q", nf.Project)
	}
	if _, ok := nf.Locations["project"]; ok {
		t.Fatal("no markers: want no project location")
	}
}

func TestGatherBindsCodePage(t *testing.T) {
	s := factstest.New()
	f := facts.Gather(s)
	if f.CodePage != 1252 || f.Represents("ابوي") || !f.Represents("café") {
		t.Fatalf("CodePage=%d Represents(ابوي)=%v", f.CodePage, f.Represents("ابوي"))
	}
}

func TestRealGatherUnder2s(t *testing.T) {
	if testing.Short() {
		t.Skip("real machine")
	}
	start := time.Now()
	f := facts.Gather(facts.Real())
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Gather took %v", d)
	}
	t.Logf("tools %v, project %q, took %v", f.Tools, f.Project, time.Since(start))
}

// An Android project is known by its manifest, in Flutter's layout or Android's own.
func TestGatherSeesAndroidManifests(t *testing.T) {
	for manifest, key := range map[string]string{
		`android\app\src\main\AndroidManifest.xml`: "android/app/src/main/AndroidManifest.xml",
		`app\src\main\AndroidManifest.xml`:         "app/src/main/AndroidManifest.xml",
	} {
		s := factstest.New()
		s.Cwd = `C:\src\café`
		s.Files[`C:\src\café\`+manifest] = ""
		if f := facts.Gather(s); f.Project != `C:\src\café` || !f.Markers[key] {
			t.Errorf("%s: Project %q, Markers %v", manifest, f.Project, f.Markers)
		}
	}
}

// Web developers: Node is a tool pathdoc names, so a clean scan of a web
// project says what it checked instead of "nothing to check".
func TestGatherNode(t *testing.T) {
	s := factstest.New()
	s.Env["PATH"] = `C:\Program Files\nodejs`
	s.Files[`C:\Program Files\nodejs\node.exe`] = ""
	s.Vers[`C:\Program Files\nodejs\node.exe`] = "24.19.0.0"
	if n := facts.Gather(s).Tools["node"]; n.Name != "Node" || n.Version != "24.19.0" {
		t.Fatalf("node = %+v", n)
	}
}
