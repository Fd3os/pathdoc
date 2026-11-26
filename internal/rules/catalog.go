package rules

import "github.com/Fd3os/pathdoc/internal/facts"

const jdkUnicodeBug = "https://bugs.openjdk.org/browse/JDK-4488646"

func reproURL(file string) string { return "https://github.com/Fd3os/pathdoc/blob/main/" + file }

// Catalog is every rule pathdoc ships. Each one was reproduced on Windows
// (code page 1252) on 2026-11-26 with three folders — plain, café and ابوي —
// and each is re-proven by its TestRepro_ test in repro/.
var Catalog = []Rule{
	{
		ID:        "gradle-project-path",
		Title:     "Gradle can't start from this project folder",
		Locations: []string{"project"},
		Trigger:   NotInCodePage,
		Applies: func(f *facts.Facts, _ facts.Location) bool {
			// Flutter keeps android\gradlew.bat out of git and writes it on the
			// first Android build, so the Android settings file counts too.
			// Flutter builds with Android Studio's bundled JDK, often with no
			// JAVA_HOME and no java on PATH: Flutter counts as a JDK here.
			return (installed(f, "java") || installed(f, "flutter")) && (f.Markers["gradlew.bat"] || f.Markers["android/gradlew.bat"] ||
				f.Markers["android/settings.gradle"] || f.Markers["android/settings.gradle.kts"])
		},
		Symptom:  "Error: Could not find or load main class org.gradle.wrapper.GradleWrapperMain",
		Breaks:   "gradlew, flutter build apk, flutter run on Android",
		Advice:   "Gradle's own 9.x gradlew.bat says: Error: Unable to access jarfile ...gradle-wrapper.jar",
		Fix:      FixHere,
		Repro:    "repro/gradle_test.go",
		Evidence: []string{jdkUnicodeBug, reproURL("repro/gradle_test.go")},
	},
	{
		ID:        "android-project-path",
		Title:     "Android builds refuse this project folder",
		Locations: []string{"project"},
		Trigger:   AnyNonASCII,
		Applies: func(f *facts.Facts, _ facts.Location) bool {
			// The Android Gradle Plugin checks the path itself, so a letter the
			// code page can write (café) stops it as surely as one it can't.
			return (installed(f, "java") || installed(f, "flutter")) &&
				(f.Markers["android/app/src/main/AndroidManifest.xml"] || f.Markers["app/src/main/AndroidManifest.xml"])
		},
		Symptom:  "Your project path contains non-ASCII characters. This will most likely cause the build to fail on Windows.",
		Breaks:   "flutter build apk, flutter run on Android, Android Studio builds",
		Advice:   "this is the Android Gradle Plugin's own check: any letter outside ASCII stops it, even é",
		Fix:      FixHere,
		Repro:    "repro/android_test.go",
		Evidence: []string{reproURL("repro/android_test.go")},
	},
	{
		ID:        "gradle-user-home",
		Title:     "Gradle can't start its daemon from this Gradle home",
		Locations: []string{"gradle-home"},
		Trigger:   NotInCodePage,
		Applies: func(f *facts.Facts, where facts.Location) bool {
			return installed(f, "flutter") || (installed(f, "java") && (where.Exists || !where.Default))
		},
		Symptom:  "Error opening zip file or JAR manifest missing : ...gradle-instrumentation-agent-....jar",
		Breaks:   "every Gradle build: gradlew, flutter build apk, Android Studio sync",
		Advice:   "then agent library failed to init: instrument (Java 17) or agent library failed Agent_OnLoad: instrument (Java 21); older Gradle says: Could not find or load main class org.gradle.launcher.daemon.bootstrap.GradleDaemon",
		Fix:      FixEnv,
		Env:      []EnvSet{{Var: "GRADLE_USER_HOME", Sub: "gradle"}},
		Repro:    "repro/gradle_test.go",
		Evidence: []string{jdkUnicodeBug, "https://github.com/godotengine/godot/issues/84196", reproURL("repro/gradle_test.go")},
	},
	{
		ID:        "dart-temp",
		Title:     "Dart can't compile ahead-of-time with this temp folder",
		Locations: []string{"temp"},
		Trigger:   AnyNonASCII,
		Applies: func(f *facts.Facts, _ facts.Location) bool {
			return installed(f, "dart") || installed(f, "flutter")
		},
		Symptom:  "Error: AOT compilation failed / Bad state: Generating AOT snapshot failed!",
		Breaks:   "dart compile exe",
		Fix:      FixEnv,
		Env:      []EnvSet{{Var: "TEMP", Sub: "temp"}, {Var: "TMP", Sub: "temp"}},
		Repro:    "repro/dart_test.go",
		Evidence: []string{reproURL("repro/dart_test.go")},
	},
	{
		ID:        "dart-aot-output",
		Title:     "Dart can't write AOT snapshots into this project",
		Locations: []string{"project"},
		Trigger:   AnyNonASCII,
		Applies: func(f *facts.Facts, _ facts.Location) bool {
			return (installed(f, "dart") || installed(f, "flutter")) && f.Markers["pubspec.yaml"]
		},
		Symptom:  "Unable to write file ... Bad state: Generating AOT snapshot failed!",
		Breaks:   "dart compile aot-snapshot, dart run build_runner",
		Advice:   "build_runner says: Unable to write file: ...build.dart.aot, then Failed to compile build script",
		Fix:      FixHere,
		Repro:    "repro/dart_test.go",
		Evidence: []string{reproURL("repro/dart_test.go")},
	},
	{
		ID:        "flutter-analyze",
		Title:     "flutter analyze crashes in this project",
		Locations: []string{"project"},
		Trigger:   AnyNonASCII,
		Applies: func(f *facts.Facts, _ facts.Location) bool {
			return installed(f, "flutter") && Less(f.Tools["flutter"].Version, "3.50.0") && f.Markers["pubspec.yaml"]
		},
		Symptom: "FormatException: Unexpected end of input",
		Breaks:  "flutter analyze (Flutter before 3.50.0)",
		Advice:  "or upgrade: Flutter 3.50.0 fixed it (flutter upgrade)",
		Fix:     FixHere,
		Repro:   "repro/flutter_test.go",
		Evidence: []string{
			"https://github.com/flutter/flutter/issues/191309",
			"https://github.com/flutter/flutter/issues/192079",
			"https://github.com/flutter/flutter/pull/191377",
		},
	},
	{
		ID:        "python-pipe-encoding",
		Title:     "Python crashes printing this path when its output is captured",
		Locations: []string{"python", "project"},
		Trigger:   NotInCodePage,
		Applies: func(f *facts.Facts, where facts.Location) bool {
			if !installed(f, "python") || !Less(f.Tools["python"].Version, "3.15") {
				return false
			}
			return where.Key != "project" || f.Markers["pyproject.toml"] || f.Markers["requirements.txt"] || f.Markers["setup.py"] || f.Markers[".venv"]
		},
		Symptom:  "UnicodeEncodeError: 'charmap' codec can't encode characters",
		Breaks:   "pip and Python tools run from an IDE, CI or a pipe (Python before 3.15)",
		Advice:   "Python 3.15 made UTF-8 the default (PEP 686)",
		Fix:      FixEnv,
		Env:      []EnvSet{{Var: "PYTHONUTF8", Value: "1"}},
		Repro:    "repro/python_test.go",
		Evidence: []string{"https://peps.python.org/pep-0686/", "https://github.com/pypa/pip/issues/11798", reproURL("repro/python_test.go")},
	},
}

func installed(f *facts.Facts, tool string) bool {
	_, ok := f.Tools[tool]
	return ok
}
