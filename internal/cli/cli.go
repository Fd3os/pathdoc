// Package cli is pathdoc's command line: scan (the default), fix, undo, here.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fd3os/pathdoc/internal/facts"
	"github.com/Fd3os/pathdoc/internal/fix"
	"github.com/Fd3os/pathdoc/internal/report"
	"github.com/Fd3os/pathdoc/internal/rules"
	"github.com/Fd3os/pathdoc/internal/win"
)

// Version is set at release time with -ldflags.
var Version = "dev"

const defaultRoot = `C:\dev`

const usage = `pathdoc finds where a non-ASCII path will break your build, and fixes it.

usage:
  pathdoc [--json] [--root DIR]      scan this machine and the project in this folder
  pathdoc fix [--yes] [--root DIR]   set the user environment variables that cure what scan found
  pathdoc undo [--force]             put back everything fix and here changed
  pathdoc here [--name NAME]         make an ASCII junction to this project and say where to build from
  pathdoc --version

exit codes: 0 clean, 1 problems found, 2 pathdoc itself failed
`

type Env struct {
	Sys            facts.System
	Store          fix.EnvStore
	Stdin          io.Reader
	Interactive    bool // stdin is a console, so fix may ask
	Stdout, Stderr io.Writer
	Color          bool
}

// Real is this process: the machine, the user's environment in the
// registry, the console, colors when stdout is a console that accepts them
// and NO_COLOR is unset.
func Real() Env {
	color := os.Getenv("NO_COLOR") == "" && win.IsConsole(os.Stdout) && win.EnableColor(os.Stdout)
	return Env{
		Sys: facts.Real(), Store: win.UserEnv{Announce: true},
		Stdin: os.Stdin, Interactive: win.IsConsole(os.Stdin),
		Stdout: os.Stdout, Stderr: os.Stderr, Color: color,
	}
}

func Run(args []string, env Env) int {
	cmd := "scan"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "scan":
		return scan(args, env)
	case "here":
		return here(args, env)
	case "fix":
		return fixCmd(args, env)
	case "undo":
		return undo(args, env)
	case "help":
		fmt.Fprint(env.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(env.Stderr, "pathdoc: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}

func flags(name string, env Env) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	fs.Usage = func() { fmt.Fprint(env.Stderr, usage) }
	return fs
}

// parsed turns a flag error into an exit code: help is not a failure.
func parsed(fs *flag.FlagSet, args []string) (code int, ok bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, false
		}
		return 2, false
	}
	return 0, true
}

func scan(args []string, env Env) int {
	fs := flags("scan", env)
	asJSON := fs.Bool("json", false, "print JSON")
	root := fs.String("root", defaultRoot, "folder pathdoc fix would use")
	version := fs.Bool("version", false, "print the version")
	if code, ok := parsed(fs, args); !ok {
		return code
	}
	if *version {
		fmt.Fprintln(env.Stdout, "pathdoc "+Version)
		return 0
	}
	start := time.Now()
	f := facts.Gather(env.Sys)
	res := report.Result{
		Version: Version, CodePage: f.CodePage, Root: *root, Tools: f.Tools,
		Findings: rules.Evaluate(f, rules.Catalog),
	}
	res.Elapsed = time.Since(start)
	if *asJSON {
		if err := report.JSON(env.Stdout, res); err != nil {
			fmt.Fprintln(env.Stderr, "pathdoc:", err)
			return 2
		}
	} else {
		report.Text(env.Stdout, res, env.Color)
	}
	if len(res.Findings) > 0 {
		return 1
	}
	return 0
}

func here(args []string, env Env) int {
	fs := flags("here", env)
	root := fs.String("root", defaultRoot, "folder that holds the junctions")
	name := fs.String("name", "", "junction name")
	if code, ok := parsed(fs, args); !ok {
		return code
	}
	cwd, err := env.Sys.Getwd()
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc here:", err)
		return 2
	}
	f := facts.Gather(env.Sys)
	st, err := fix.LoadState(statePath(env))
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc here:", err)
		return 2
	}
	link, cd, err := fix.Here(f.Project, cwd, *root, *name, st)
	if errors.Is(err, fix.ErrAlreadyASCII) {
		fmt.Fprintln(env.Stdout, "this project's path is plain ASCII; nothing to do")
		return 0
	}
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc here:", err)
		return 2
	}
	st.Root = *root
	if err := st.Save(); err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc here:", err)
		return 2
	}
	project := f.Project
	if project == "" {
		project = cwd
	}
	fmt.Fprintf(env.Stdout, "%s -> %s\n\n", link, project)
	fmt.Fprintln(env.Stdout, "build from it in PowerShell or cmd (Git Bash follows the junction back to the real path):")
	// Quoted by hand: %q would double every backslash.
	fmt.Fprintf(env.Stdout, "  PowerShell:  Set-Location \"%s\"\n", cd)
	fmt.Fprintf(env.Stdout, "  cmd:         cd /d \"%s\"\n", cd)
	if f.Markers["package.json"] {
		fmt.Fprintln(env.Stdout, "\nnote: run npm and Vite builds from the real folder; Node follows the junction back and the build breaks the other way round")
	}
	fmt.Fprintln(env.Stdout, "\npathdoc undo removes it")
	return 0
}

// statePath is the undo record under the user's local app data.
func statePath(env Env) string {
	base := env.Sys.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(env.Sys.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return fix.StatePath(base)
}

func fixCmd(args []string, env Env) int {
	fs := flags("fix", env)
	root := fs.String("root", defaultRoot, "folder for the new tool homes")
	yes := fs.Bool("yes", false, "apply without asking")
	if code, ok := parsed(fs, args); !ok {
		return code
	}
	f := facts.Gather(env.Sys)
	findings := rules.Evaluate(f, rules.Catalog)
	steps, err := fix.Plan(findings, *root, env.Store)
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc fix:", err)
		return 2
	}
	var here []string
	for _, fd := range findings {
		if fd.Rule.Fix == rules.FixHere {
			here = append(here, fd.Rule.Title)
		}
	}
	if len(steps) == 0 {
		fmt.Fprintln(env.Stdout, "nothing for pathdoc fix to change")
		hereNote(env, here)
		return 0
	}
	fmt.Fprintln(env.Stdout, "pathdoc fix will set these user environment variables:")
	var dirs []string
	for _, s := range steps {
		now := "not set"
		if cur, err := env.Store.Get(s.Var); err == nil && cur.Exists {
			now = cur.Data
		}
		fmt.Fprintf(env.Stdout, "  %-17s = %s   (now: %s)\n", s.Var, s.New.Data, now)
		if s.Dir != "" && (len(dirs) == 0 || dirs[len(dirs)-1] != s.Dir) {
			dirs = append(dirs, s.Dir)
		}
	}
	if len(dirs) > 0 {
		fmt.Fprintf(env.Stdout, "and create %s\n", strings.Join(dirs, ", "))
	}
	fmt.Fprintln(env.Stdout, "Nothing else is touched; pathdoc undo puts it all back.")
	if !*yes {
		if !env.Interactive {
			fmt.Fprintln(env.Stderr, "pathdoc fix: not asking without a console; run pathdoc fix --yes to apply")
			return 2
		}
		fmt.Fprint(env.Stdout, "\nApply these changes? [y/N] ")
		answer, _ := bufio.NewReader(env.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Fprintln(env.Stdout, "nothing changed")
			return 1
		}
	}
	st, err := fix.LoadState(statePath(env))
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc fix:", err)
		return 2
	}
	applyErr := fix.Apply(steps, st, env.Store)
	st.Root = *root
	if err := errors.Join(applyErr, st.Save()); err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc fix:", err)
		return 2
	}
	fmt.Fprintln(env.Stdout, "\ndone. Open a new terminal and restart your IDE so they see the new values.")
	fmt.Fprintln(env.Stdout, "Tools rebuild their caches in the new folders, so the next build is slower once.")
	hereNote(env, here)
	return 0
}

func hereNote(env Env, titles []string) {
	if len(titles) == 0 {
		return
	}
	fmt.Fprintf(env.Stdout, "\nstill to do, inside the project: pathdoc here (for: %s)\n", strings.Join(titles, "; "))
}

func undo(args []string, env Env) int {
	fs := flags("undo", env)
	force := fs.Bool("force", false, "restore values changed since fix too")
	if code, ok := parsed(fs, args); !ok {
		return code
	}
	st, err := fix.LoadState(statePath(env))
	if err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc undo:", err)
		return 2
	}
	if len(st.Changes) == 0 {
		fmt.Fprintln(env.Stdout, "nothing to undo")
		return 0
	}
	skipped, undoErr := fix.Undo(st, env.Store, *force, env.Stdout)
	if err := errors.Join(undoErr, st.Save()); err != nil {
		fmt.Fprintln(env.Stderr, "pathdoc undo:", err)
		return 2
	}
	if skipped > 0 {
		fmt.Fprintf(env.Stdout, "\n%d left as they are; pathdoc undo --force puts them back anyway\n", skipped)
		return 1
	}
	fmt.Fprintln(env.Stdout, "\neverything pathdoc changed is back. Open a new terminal to see it.")
	return 0
}
