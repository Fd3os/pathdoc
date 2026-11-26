//go:build repro

package repro

import (
	"os"
	"path/filepath"
	"testing"
)

const gradleVersion = "Gradle 9.3.1"

// gradle-project-path: the Java launcher reads its arguments in the ANSI
// code page (JDK-4488646), so a wrapper path the code page cannot hold
// arrives as "????". Flutter's gradlew.bat passes it with -classpath;
// Gradle's own 9.x script with -jar. A junction cures both.
func TestRepro_gradle_project_path(t *testing.T) {
	java := tool(t, "java").Path
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "gw")
		copyTree(t, filepath.Join("testdata", "gradle"), dir)
		classpath := func(root string) result {
			jar := filepath.Join(root, "gradle", "wrapper", "gradle-wrapper.jar")
			return run(t, root, nil, java, "-classpath", jar, "org.gradle.wrapper.GradleWrapperMain", "--version")
		}
		script := func(root string) result { return run(t, root, nil, filepath.Join(root, "gradlew.bat"), "--version") }
		if f.key != "nonacp" {
			pass(t, f.name+": classpath", classpath(dir), gradleVersion)
			pass(t, f.name+": gradlew.bat", script(dir), gradleVersion)
			continue
		}
		fail(t, f.name+": classpath", classpath(dir), "Could not find or load main class org.gradle.wrapper.GradleWrapperMain")
		fail(t, f.name+": gradlew.bat", script(dir), "Unable to access jarfile")
		link := junction(t, filepath.Join(base, "jg"), dir)
		pass(t, f.name+" through a junction: classpath", classpath(link), gradleVersion)
		pass(t, f.name+" through a junction: gradlew.bat", script(link), gradleVersion)
	}
}

// gradle-user-home: the daemon starts with -cp and -javaagent pointing into
// the Gradle home; a home the code page cannot hold breaks it. An ASCII home
// — what pathdoc fix sets — cures it.
func TestRepro_gradle_user_home(t *testing.T) {
	tool(t, "java")
	base := lab(t)
	proj := filepath.Join(base, "project")
	copyTree(t, filepath.Join("testdata", "gradle"), proj)
	gradlew := filepath.Join(proj, "gradlew.bat")
	pass(t, "default home", run(t, proj, nil, gradlew, "--version"), gradleVersion)

	home := os.Getenv("GRADLE_USER_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("USERPROFILE"), ".gradle")
	}
	dist := filepath.Join(home, "wrapper", "dists", "gradle-9.3.1-all")
	if _, err := os.Stat(dist); err != nil {
		t.Fatalf("the default run should have left the distribution at %s: %v", dist, err)
	}
	for _, f := range folders {
		gh := filepath.Join(base, f.name, "gh")
		os.MkdirAll(filepath.Join(gh, "wrapper", "dists"), 0o755)
		junction(t, filepath.Join(gh, "wrapper", "dists", "gradle-9.3.1-all"), dist) // reuse, no second download
		env := []string{"GRADLE_USER_HOME=" + gh}
		r := run(t, proj, env, gradlew, "help", "--console=plain")
		run(t, proj, env, gradlew, "--stop")
		if f.key == "nonacp" {
			fail(t, f.name+" home", r, "agent library failed to init: instrument")
		} else {
			pass(t, f.name+" home", r, "BUILD SUCCESSFUL")
		}
	}
}
