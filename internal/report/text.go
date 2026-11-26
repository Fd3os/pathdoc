// Package report prints a scan: as text for people, as JSON for scripts.
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/rules"
)

type Result struct {
	Version  string
	CodePage uint32
	Elapsed  time.Duration
	Root     string
	Tools    map[string]facts.Tool
	Findings []rules.Finding
}

const (
	red   = "\x1b[31m"
	green = "\x1b[32m"
	bold  = "\x1b[1m"
	dim   = "\x1b[2m"
	reset = "\x1b[0m"
)

func Text(w io.Writer, r Result, color bool) {
	paint := func(code, s string) string {
		if !color {
			return s
		}
		return code + s + reset
	}
	page := fmt.Sprint(r.CodePage)
	if r.CodePage == 65001 {
		page += " (UTF-8)"
	}
	fmt.Fprintf(w, "pathdoc %s · code page %s\n\n", r.Version, page)
	took := fmt.Sprintf("scanned in %.2f s", r.Elapsed.Seconds())
	if len(r.Findings) == 0 {
		if len(r.Tools) == 0 {
			fmt.Fprintf(w, "%s nothing to check: none of Java, Flutter, Dart or Python is installed · %s\n", paint(green, "ok"), took)
			return
		}
		fmt.Fprintf(w, "%s no path problems for %s · %s\n", paint(green, "ok"), toolList(r.Tools), took)
		return
	}
	row := func(label, value string) {
		fmt.Fprintf(w, "    %s %s\n", paint(dim, fmt.Sprintf("%-6s", label)), value)
	}
	byFix, byHere := 0, 0
	for _, f := range r.Findings {
		fmt.Fprintf(w, "%s %s\n", paint(red+bold, "x"), paint(bold, f.Rule.Title))
		row("where", where(f.Where))
		row("why", why(f, r.CodePage))
		row("breaks", f.Rule.Breaks)
		row("error", f.Rule.Symptom)
		if f.Rule.Advice != "" {
			row("note", f.Rule.Advice)
		}
		if f.Rule.Fix == rules.FixEnv {
			byFix++
			row("fix", "pathdoc fix  ->  "+strings.Join(envPairs(f.Rule, r.Root), ", "))
		} else {
			byHere++
			row("fix", "pathdoc here  ->  build from an ASCII junction to this project")
		}
		for i, e := range f.Rule.Evidence {
			label := ""
			if i == 0 {
				label = "proof"
			}
			row(label, e)
		}
		fmt.Fprintln(w)
	}
	parts := []string{plural(len(r.Findings), "problem")}
	if byFix > 0 {
		parts = append(parts, fmt.Sprintf("%d fixed by pathdoc fix", byFix))
	}
	if byHere > 0 {
		parts = append(parts, fmt.Sprintf("%d by pathdoc here", byHere))
	}
	fmt.Fprintln(w, strings.Join(append(parts, took), " · "))
}

func where(l facts.Location) string {
	s := l.Label + "  " + l.Path
	switch {
	case l.Default:
		s += "  (default; " + l.EnvVar + " is not set)"
	case l.EnvVar != "":
		s += "  (from " + l.EnvVar + ")"
	}
	return s
}

func why(f rules.Finding, cp uint32) string {
	quoted := make([]string, len(f.Offending))
	for i, s := range f.Offending {
		quoted[i] = `"` + s + `"`
	}
	names := strings.Join(quoted, ", ")
	if f.Rule.Trigger == rules.AnyNonASCII {
		verb := "is"
		if len(quoted) > 1 {
			verb = "are"
		}
		return names + " " + verb + " not plain ASCII, and this tool breaks on any letter outside it"
	}
	return fmt.Sprintf("%s can't be written in code page %d", names, cp)
}

func envPairs(r rules.Rule, root string) []string {
	var out []string
	for _, e := range r.Env {
		out = append(out, e.Var+"="+e.Under(root))
	}
	return out
}

func toolList(tools map[string]facts.Tool) string {
	var names []string
	for _, t := range tools {
		names = append(names, strings.TrimSpace(t.Name+" "+t.Version))
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
