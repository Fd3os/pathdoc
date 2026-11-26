package facts

import (
	"testing"
	"time"
)

func TestGatherDefaultsGradleHome(t *testing.T) {
	s := newFake()
	s.env["USERPROFILE"] = `C:\Users\محمد`
	got := Gather(s).Locations["gradle-home"]
	want := Location{Key: "gradle-home", Label: "Gradle home", Path: `C:\Users\محمد\.gradle`, EnvVar: "GRADLE_USER_HOME", Default: true}
	if got != want {
		t.Fatalf("gradle-home = %+v, want %+v", got, want)
	}
	s.env["GRADLE_USER_HOME"] = `D:\g`
	if got := Gather(s).Locations["gradle-home"]; got.Path != `D:\g` || got.Default {
		t.Fatalf("set GRADLE_USER_HOME: got %+v", got)
	}
}

func TestGatherTemp(t *testing.T) {
	s := newFake()
	s.env["TMP"] = `C:\t2`
	if got := Gather(s).Locations["temp"]; got.Path != `C:\t2` || got.EnvVar != "TMP" {
		t.Fatalf("TMP only: got %+v", got)
	}
	s.env["TEMP"] = `C:\Users\محمد\AppData\Local\Temp`
	got := Gather(s).Locations["temp"]
	if got.Path != `C:\Users\محمد\AppData\Local\Temp` || got.EnvVar != "TEMP" || got.Label != "Temp folder" {
		t.Fatalf("TEMP: got %+v", got)
	}
	if _, ok := Gather(newFake()).Locations["temp"]; ok {
		t.Fatal("no TEMP or TMP: want no temp location")
	}
}

func TestGatherSkipsPythonStub(t *testing.T) {
	s := newFake()
	stub := `C:\Users\محمد\AppData\Local\Microsoft\WindowsApps`
	s.env["PATH"] = stub + `;"C:\Python312"`
	s.files[stub+`\python.exe`] = ""
	s.files[`C:\Python312\python.exe`] = ""
	s.vers[`C:\Python312\python.exe`] = "3.12.3150.1013"
	f := Gather(s)
	if py := f.Tools["python"]; py.Path != `C:\Python312\python.exe` || py.Version != "3.12" {
		t.Fatalf("python = %+v", py)
	}
	if loc := f.Locations["python"]; loc.Path != `C:\Python312` {
		t.Fatalf("python location = %+v", loc)
	}

	only := newFake()
	only.env["PATH"] = stub
	only.files[stub+`\python.exe`] = ""
	if py, ok := Gather(only).Tools["python"]; ok {
		t.Fatalf("only the Store stub: want no python, got %+v", py)
	}
}

func TestGatherJava(t *testing.T) {
	s := newFake()
	javapath := `C:\Program Files (x86)\Common Files\Oracle\Java\java8path`
	s.env["PATH"] = javapath
	s.files[javapath+`\java.exe`] = ""
	s.vers[javapath+`\java.exe`] = "8.0.5010.8"
	if j := Gather(s).Tools["java"]; j.Path != javapath+`\java.exe` || j.Version != "8.0.5010" {
		t.Fatalf("java from PATH = %+v", j)
	}
	home := `C:\jdk-17`
	s.env["JAVA_HOME"] = home
	s.files[home+`\bin\java.exe`] = ""
	s.files[home+`\release`] = "IMPLEMENTOR=\"x\"\nJAVA_VERSION=\"17.0.20\"\n"
	if j := Gather(s).Tools["java"]; j.Path != home+`\bin\java.exe` || j.Version != "17.0.20" {
		t.Fatalf("java from JAVA_HOME = %+v", j)
	}
}

func TestGatherFlutterAndDart(t *testing.T) {
	s := newFake()
	s.env["PATH"] = `C:\flutter\bin`
	s.env["PATHEXT"] = ".COM;.EXE;.BAT;.CMD"
	s.files[`C:\flutter\bin\flutter.bat`] = ""
	s.files[`C:\flutter\bin\dart.bat`] = ""
	s.files[`C:\flutter\bin\cache\flutter.version.json`] = `{"frameworkVersion": "3.47.1", "channel": "stable"}`
	s.files[`C:\flutter\bin\cache\dart-sdk\version`] = "3.13.1\n"
	f := Gather(s)
	if fl := f.Tools["flutter"]; fl.Path != `C:\flutter\bin\flutter.bat` || fl.Version != "3.47.1" {
		t.Fatalf("flutter = %+v", fl)
	}
	if d := f.Tools["dart"]; d.Version != "3.13.1" {
		t.Fatalf("dart in flutter = %+v", d)
	}

	old := newFake()
	old.env["PATH"] = `C:\flutter\bin`
	old.files[`C:\flutter\bin\flutter.bat`] = ""
	old.files[`C:\flutter\version`] = "3.22.0\n"
	if fl := Gather(old).Tools["flutter"]; fl.Version != "3.22.0" {
		t.Fatalf("flutter from version file = %+v", fl)
	}

	alone := newFake()
	alone.env["PATH"] = `C:\tools\dart-sdk\bin`
	alone.files[`C:\tools\dart-sdk\bin\dart.exe`] = ""
	alone.files[`C:\tools\dart-sdk\version`] = "3.9.0"
	if d := Gather(alone).Tools["dart"]; d.Path != `C:\tools\dart-sdk\bin\dart.exe` || d.Version != "3.9.0" {
		t.Fatalf("standalone dart = %+v", d)
	}
}

func TestGatherProjectWalksUp(t *testing.T) {
	s := newFake()
	s.cwd = `C:\Users\محمد\app\lib`
	s.files[`C:\Users\محمد\app\pubspec.yaml`] = ""
	s.files[`C:\Users\محمد\app\android\gradlew.bat`] = ""
	f := Gather(s)
	if f.Project != `C:\Users\محمد\app` {
		t.Fatalf("Project = %q", f.Project)
	}
	if !f.Markers["pubspec.yaml"] || !f.Markers["android/gradlew.bat"] || f.Markers["gradlew.bat"] {
		t.Fatalf("Markers = %v", f.Markers)
	}
	if loc := f.Locations["project"]; loc.Path != `C:\Users\محمد\app` || loc.Label != "Project" {
		t.Fatalf("project location = %+v", loc)
	}

	none := newFake()
	none.cwd = `C:\Users\محمد\Desktop`
	nf := Gather(none)
	if nf.Project != "" {
		t.Fatalf("no markers: Project = %q", nf.Project)
	}
	if _, ok := nf.Locations["project"]; ok {
		t.Fatal("no markers: want no project location")
	}
}

func TestGatherBindsCodePage(t *testing.T) {
	s := newFake()
	f := Gather(s)
	if f.CodePage != 1252 || f.Represents("ابوي") || !f.Represents("café") {
		t.Fatalf("CodePage=%d Represents(ابوي)=%v", f.CodePage, f.Represents("ابوي"))
	}
}

func TestRealGatherUnder2s(t *testing.T) {
	if testing.Short() {
		t.Skip("real machine")
	}
	start := time.Now()
	f := Gather(Real())
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Gather took %v", d)
	}
	t.Logf("tools %v, project %q, took %v", f.Tools, f.Project, time.Since(start))
}
