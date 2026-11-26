package fix

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Fd3os/pathdoc/internal/rules"
	"github.com/Fd3os/pathdoc/internal/win"
	"golang.org/x/sys/windows"
)

// EnvStore is where user environment variables live: the registry for
// real, a map in tests.
type EnvStore interface {
	Get(name string) (Value, error)
	Set(name string, v Value) error
	Delete(name string) error
	Broadcast()
}

// Step is one variable fix will set, and the folder it makes first.
type Step struct {
	Var string
	New Value
	Dir string
}

// Plan turns findings into the variables to set, once each, skipping any
// already at its target. Findings cured by here are not fix's to plan.
func Plan(findings []rules.Finding, root string, store EnvStore) ([]Step, error) {
	if err := ValidateRoot(root); err != nil {
		return nil, err
	}
	var steps []Step
	seen := map[string]bool{}
	for _, f := range findings {
		if f.Rule.Fix != rules.FixEnv {
			continue
		}
		for _, e := range f.Rule.Env {
			if seen[e.Var] {
				continue
			}
			seen[e.Var] = true
			val := e.Under(root)
			cur, err := store.Get(e.Var)
			if err != nil {
				return nil, err
			}
			if cur.Exists && cur.Data == val {
				continue
			}
			step := Step{Var: e.Var, New: Value{Exists: true, Data: val}}
			if e.Sub != "" {
				step.Dir = val
			}
			steps = append(steps, step)
		}
	}
	return steps, nil
}

// Apply makes each step's folder, sets its variable, and records both. A
// variable pathdoc already changed keeps the value it had before the first
// change, so undo always goes back to the user's own.
func Apply(steps []Step, st *State, store EnvStore) error {
	defer store.Broadcast()
	for _, s := range steps {
		if s.Dir != "" {
			if err := ensureDir(s.Dir, st); err != nil {
				return err
			}
		}
		old, err := store.Get(s.Var)
		if err != nil {
			return err
		}
		if err := store.Set(s.Var, s.New); err != nil {
			return err
		}
		recordEnv(st, s.Var, old, s.New.Data)
	}
	return nil
}

func recordEnv(st *State, name string, old Value, set string) {
	now := time.Now().UTC()
	for i := range st.Changes {
		if c := &st.Changes[i]; c.Kind == "env" && strings.EqualFold(c.Name, name) {
			c.New, c.At = set, now
			return
		}
	}
	st.Changes = append(st.Changes, Change{Kind: "env", Name: name, Old: old, New: set, At: now})
}

// Undo walks the record newest first. A variable someone changed since fix
// is left alone (and kept in the record) unless force. Junctions are removed
// only while they still lead where pathdoc pointed them; folders only when
// empty — a full one is kept on disk and dropped from the record. It never
// deletes a file.
func Undo(st *State, store EnvStore, force bool, w io.Writer) (skipped int, err error) {
	var keep []Change
	touchedEnv := false
	for i := len(st.Changes) - 1; i >= 0; i-- {
		c := st.Changes[i]
		switch c.Kind {
		case "env":
			cur, gerr := store.Get(c.Name)
			if gerr != nil {
				err = errors.Join(err, gerr)
				keep = append(keep, c)
				continue
			}
			if !force && (!cur.Exists || cur.Data != c.New) {
				now := "unset"
				if cur.Exists {
					now = cur.Data
				}
				fmt.Fprintf(w, "left %s alone: it is now %s, not the %s pathdoc set (undo --force puts back the old value)\n", c.Name, now, c.New)
				keep = append(keep, c)
				skipped++
				continue
			}
			var serr error
			if c.Old.Exists {
				serr = store.Set(c.Name, c.Old)
				fmt.Fprintf(w, "restored %s = %s\n", c.Name, c.Old.Data)
			} else {
				serr = store.Delete(c.Name)
				fmt.Fprintf(w, "removed %s (it was not set before)\n", c.Name)
			}
			if serr != nil {
				err = errors.Join(err, serr)
				keep = append(keep, c)
				continue
			}
			touchedEnv = true
		case "junction":
			target, terr := win.JunctionTarget(c.Path)
			switch {
			case errors.Is(terr, os.ErrNotExist):
				// already gone
			case terr != nil || !samePath(target, c.Target):
				fmt.Fprintf(w, "left %s alone: it is no longer the junction pathdoc made\n", c.Path)
			default:
				if rerr := win.RemoveEmptyDir(c.Path); rerr != nil { // removes the junction itself, never a file
					err = errors.Join(err, rerr)
					keep = append(keep, c)
					continue
				}
				fmt.Fprintf(w, "removed junction %s\n", c.Path)
			}
		case "dir":
			// RemoveDirectory, not os.Remove: a file now at this path is the
			// user's, and os.Remove would delete it.
			switch rerr := win.RemoveEmptyDir(c.Path); {
			case rerr == nil, errors.Is(rerr, os.ErrNotExist):
			case errors.Is(rerr, windows.ERROR_DIR_NOT_EMPTY):
				fmt.Fprintf(w, "kept %s: it is not empty\n", c.Path)
			case errors.Is(rerr, windows.ERROR_DIRECTORY):
				fmt.Fprintf(w, "left %s alone: it is no longer a folder pathdoc made\n", c.Path)
			default:
				fmt.Fprintf(w, "kept %s: %v\n", c.Path, rerr)
			}
		}
	}
	for l, r := 0, len(keep)-1; l < r; l, r = l+1, r-1 {
		keep[l], keep[r] = keep[r], keep[l]
	}
	st.Changes = keep
	if touchedEnv {
		store.Broadcast()
	}
	return skipped, err
}
