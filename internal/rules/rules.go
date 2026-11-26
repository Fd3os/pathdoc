// Package rules holds what pathdoc knows breaks — each rule reproduced on a
// real Windows machine before it ships — and checks a machine's facts
// against it.
package rules

import (
	"path/filepath"

	"github.com/Fd3os/pathdoc/internal/facts"
)

// Trigger says which characters break a tool.
type Trigger int

const (
	// NotInCodePage: the tool passes the path through the ANSI code page, so
	// only characters that code page cannot hold break it.
	NotInCodePage Trigger = iota
	// AnyNonASCII: any character outside ASCII breaks the tool.
	AnyNonASCII
)

func (t Trigger) String() string {
	if t == AnyNonASCII {
		return "any-non-ascii"
	}
	return "not-in-code-page"
}

// FixKind says how pathdoc cures a finding.
type FixKind int

const (
	FixEnv  FixKind = iota // set user environment variables (pathdoc fix)
	FixHere                // build from an ASCII junction (pathdoc here)
)

func (k FixKind) String() string {
	if k == FixHere {
		return "here"
	}
	return "env"
}

// EnvSet is one variable a fix sets: to a folder under the root (Sub) or to
// a literal Value.
type EnvSet struct{ Var, Sub, Value string }

type Rule struct {
	ID, Title, Symptom, Breaks, Advice string
	Locations                          []string // location keys checked in order; the first that breaks wins
	Trigger                            Trigger
	Applies                            func(f *facts.Facts, where facts.Location) bool
	Fix                                FixKind
	Env                                []EnvSet
	Repro                              string // repo-relative file holding TestRepro_<ID>
	Evidence                           []string
}

// Finding is a rule that fires on this machine, where, and the path segments
// that break it.
type Finding struct {
	Rule      Rule
	Where     facts.Location
	Offending []string
}

// Under gives the value this variable gets when pathdoc's root is root.
func (e EnvSet) Under(root string) string {
	if e.Value != "" {
		return e.Value
	}
	return filepath.Join(root, e.Sub)
}
