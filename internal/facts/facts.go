// Package facts gathers what pathdoc checks: the code page, the folders tools
// write into, the tools on PATH with their versions (read from files — a tool
// is never run), and the project around the current folder.
package facts

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
)

// System is everything pathdoc reads from the machine.
type System interface {
	Getenv(key string) string
	Getwd() (string, error)
	ReadFile(path string) ([]byte, error)
	Exists(path string) bool
	FileVersion(path string) (string, error)
	CodePage() uint32
	Represents(s string, cp uint32) bool
}

// Location is a folder a tool reads or writes. Key is one of "temp",
// "gradle-home", "project", "python".
type Location struct {
	Key, Label, Path, EnvVar string
	Default                  bool // Path is the tool's default because EnvVar is unset
	Exists                   bool // the folder is there now
}

// Tool is an installed tool. Version is "" when no file says it.
type Tool struct{ Name, Path, Version string }

type Facts struct {
	CodePage   uint32
	Represents func(string) bool // bound to CodePage
	Locations  map[string]Location
	Tools      map[string]Tool // "java", "flutter", "dart", "python"
	Project    string          // nearest folder with a marker, "" if none
	Markers    map[string]bool
}

// markerNames, with forward slashes; a folder holding any of them is a project.
var markerNames = []string{
	"pubspec.yaml", "gradlew.bat", "android/gradlew.bat",
	"android/settings.gradle", "android/settings.gradle.kts",
	"pyproject.toml", "requirements.txt", "setup.py", ".venv", "package.json",
}

func Gather(sys System) *Facts {
	cp := sys.CodePage()
	f := &Facts{
		CodePage:   cp,
		Represents: func(s string) bool { return sys.Represents(s, cp) },
		Locations:  map[string]Location{},
		Tools:      map[string]Tool{},
	}
	for _, v := range []string{"TEMP", "TMP"} {
		if p := sys.Getenv(v); p != "" {
			f.Locations["temp"] = Location{Key: "temp", Label: "Temp folder", Path: p, EnvVar: v}
			break
		}
	}
	gradle := Location{Key: "gradle-home", Label: "Gradle home", EnvVar: "GRADLE_USER_HOME", Path: sys.Getenv("GRADLE_USER_HOME")}
	if gradle.Path == "" {
		if home := sys.Getenv("USERPROFILE"); home != "" {
			gradle.Path, gradle.Default = filepath.Join(home, ".gradle"), true
		}
	}
	if gradle.Path != "" {
		f.Locations["gradle-home"] = gradle
	}
	f.Project, f.Markers = findProject(sys)
	if f.Project != "" {
		f.Locations["project"] = Location{Key: "project", Label: "Project", Path: f.Project}
	}

	finders := map[string]func(System) (Tool, bool){
		"java": findJava, "flutter": findFlutter, "dart": findDart, "python": findPython,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for key, find := range finders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if t, ok := find(sys); ok {
				mu.Lock()
				f.Tools[key] = t
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if py, ok := f.Tools["python"]; ok {
		f.Locations["python"] = Location{Key: "python", Label: "Python", Path: filepath.Dir(py.Path)}
	}
	for k, l := range f.Locations {
		l.Exists = sys.Exists(l.Path)
		f.Locations[k] = l
	}
	return f
}

func findProject(sys System) (string, map[string]bool) {
	dir, err := sys.Getwd()
	if err != nil || dir == "" {
		return "", map[string]bool{}
	}
	for i := 0; i < 64; i++ {
		found := map[string]bool{}
		for _, m := range markerNames {
			if sys.Exists(filepath.Join(dir, filepath.FromSlash(m))) {
				found[m] = true
			}
		}
		if len(found) > 0 {
			return dir, found
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", map[string]bool{}
}

// lookAll lists every match for name on PATH, in PATH order, the way cmd
// would try them (PATHEXT extensions).
func lookAll(sys System, name string) []string {
	exts := strings.Split(sys.Getenv("PATHEXT"), ";")
	if sys.Getenv("PATHEXT") == "" {
		exts = []string{".com", ".exe", ".bat", ".cmd"}
	}
	var out []string
	for _, dir := range filepath.SplitList(sys.Getenv("PATH")) {
		dir = strings.Trim(dir, `"`)
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if ext == "" {
				continue
			}
			if p := filepath.Join(dir, name+ext); sys.Exists(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func findJava(sys System) (Tool, bool) {
	if home := sys.Getenv("JAVA_HOME"); home != "" {
		if exe := filepath.Join(home, "bin", "java.exe"); sys.Exists(exe) {
			return Tool{Name: "Java", Path: exe, Version: javaVersion(sys, home, exe)}, true
		}
	}
	for _, exe := range lookAll(sys, "java") {
		return Tool{Name: "Java", Path: exe, Version: javaVersion(sys, filepath.Dir(filepath.Dir(exe)), exe)}, true
	}
	return Tool{}, false
}

func javaVersion(sys System, home, exe string) string {
	if b, err := sys.ReadFile(filepath.Join(home, "release")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "JAVA_VERSION="); ok {
				return strings.Trim(v, `"`)
			}
		}
	}
	if v, err := sys.FileVersion(exe); err == nil {
		return firstParts(v, 3)
	}
	return ""
}

func findFlutter(sys System) (Tool, bool) {
	for _, p := range lookAll(sys, "flutter") {
		root := filepath.Dir(filepath.Dir(p))
		v := ""
		if b, err := sys.ReadFile(filepath.Join(root, "bin", "cache", "flutter.version.json")); err == nil {
			var meta struct{ FrameworkVersion string }
			if json.Unmarshal(b, &meta) == nil {
				v = meta.FrameworkVersion
			}
		}
		if v == "" {
			v = readTrim(sys, filepath.Join(root, "version"))
		}
		return Tool{Name: "Flutter", Path: p, Version: v}, true
	}
	return Tool{}, false
}

func findDart(sys System) (Tool, bool) {
	for _, p := range lookAll(sys, "dart") {
		bin := filepath.Dir(p)
		// Flutter's dart.bat sits beside flutter.bat; its SDK is in bin\cache.
		if sys.Exists(filepath.Join(bin, "flutter.bat")) || sys.Exists(filepath.Join(bin, "cache", "dart-sdk")) {
			return Tool{Name: "Dart", Path: p, Version: readTrim(sys, filepath.Join(bin, "cache", "dart-sdk", "version"))}, true
		}
		return Tool{Name: "Dart", Path: p, Version: readTrim(sys, filepath.Join(filepath.Dir(bin), "version"))}, true
	}
	return Tool{}, false
}

// findPython skips any python.exe with no version resource — the Microsoft
// Store's placeholder in WindowsApps is one.
func findPython(sys System) (Tool, bool) {
	for _, p := range lookAll(sys, "python") {
		if v, err := sys.FileVersion(p); err == nil {
			return Tool{Name: "Python", Path: p, Version: firstParts(v, 2)}, true
		}
	}
	return Tool{}, false
}

func readTrim(sys System, path string) string {
	b, err := sys.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func firstParts(v string, n int) string {
	parts := strings.Split(v, ".")
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, ".")
}
