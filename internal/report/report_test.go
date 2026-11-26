package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/rules"
)

func rule(id string) rules.Rule {
	for _, r := range rules.Catalog {
		if r.ID == id {
			return r
		}
	}
	panic("no rule " + id)
}

func sample() Result {
	return Result{
		Version: "0.1.0", CodePage: 1252, Elapsed: 70 * time.Millisecond, Root: `C:\dev`,
		Tools: map[string]facts.Tool{"java": {Name: "Java", Version: "17.0.20"}, "flutter": {Name: "Flutter", Version: "3.47.1"}},
		Findings: []rules.Finding{
			{Rule: rule("gradle-user-home"), Where: facts.Location{Key: "gradle-home", Label: "Gradle home", Path: `C:\Users\محمد\.gradle`, EnvVar: "GRADLE_USER_HOME", Default: true}, Offending: []string{"محمد"}},
			{Rule: rule("flutter-analyze"), Where: facts.Location{Key: "project", Label: "Project", Path: `C:\Users\محمد\app`}, Offending: []string{"محمد"}},
		},
	}
}

func TestTextShowsEverythingTheUserNeeds(t *testing.T) {
	var b bytes.Buffer
	Text(&b, sample(), false)
	out := b.String()
	for _, want := range []string{
		`C:\Users\محمد\.gradle`, `"محمد" can't be written in code page 1252`,
		"agent library failed to init: instrument", `GRADLE_USER_HOME=C:\dev\gradle`, "pathdoc fix",
		"https://bugs.openjdk.org/browse/JDK-4488646", "GRADLE_USER_HOME is not set",
		"FormatException: Unexpected end of input", "pathdoc here", "Flutter 3.50.0 fixed it",
		"2 problems", "1 fixed by pathdoc fix", "1 by pathdoc here", "0.07 s",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("text report lacks %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("color=false but the report has escape codes")
	}
	var c bytes.Buffer
	Text(&c, sample(), true)
	if !strings.Contains(c.String(), "\x1b[") {
		t.Error("color=true but no escape codes")
	}
}

func TestTextCleanNamesWhatWasChecked(t *testing.T) {
	r := sample()
	r.Findings = nil
	var b bytes.Buffer
	Text(&b, r, false)
	if out := b.String(); !strings.Contains(out, "no path problems for Flutter 3.47.1, Java 17.0.20") {
		t.Errorf("clean report:\n%s", out)
	}
	r.Tools = map[string]facts.Tool{}
	b.Reset()
	Text(&b, r, false)
	if out := b.String(); !strings.Contains(out, "nothing to check") {
		t.Errorf("no tools report:\n%s", out)
	}
}

func TestJSONRoundTripsWithArabicIntact(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, sample()); err != nil {
		t.Fatal(err)
	}
	var got struct {
		CodePage  uint32 `json:"codePage"`
		ElapsedMs int64  `json:"elapsedMs"`
		Findings  []struct {
			Rule      string   `json:"rule"`
			Path      string   `json:"path"`
			Offending []string `json:"offending"`
			Trigger   string   `json:"trigger"`
			Fix       struct {
				Kind string            `json:"kind"`
				Env  map[string]string `json:"env"`
			} `json:"fix"`
			Evidence []string `json:"evidence"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, b.String())
	}
	f := got.Findings[0]
	if got.CodePage != 1252 || got.ElapsedMs != 70 || f.Rule != "gradle-user-home" || f.Path != `C:\Users\محمد\.gradle` ||
		f.Offending[0] != "محمد" || f.Trigger != "not-in-code-page" || f.Fix.Kind != "env" || f.Fix.Env["GRADLE_USER_HOME"] != `C:\dev\gradle` {
		t.Fatalf("decoded %+v", got)
	}
	if got.Findings[1].Fix.Kind != "here" {
		t.Errorf("flutter-analyze fix = %q", got.Findings[1].Fix.Kind)
	}
	r := sample()
	r.Findings = nil
	b.Reset()
	JSON(&b, r)
	if !strings.Contains(b.String(), `"findings": []`) {
		t.Errorf("clean JSON should say findings: [] — got\n%s", b.String())
	}
}
