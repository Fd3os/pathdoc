// Package fix changes the machine, and remembers every change so undo can
// put it back: user environment variables, folders it made, junctions.
package fix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Value is a raw registry value: Data is never expanded, Expand says it was
// REG_EXPAND_SZ, Exists false means there was no value at all.
type Value struct {
	Exists bool   `json:"exists"`
	Data   string `json:"data,omitempty"`
	Expand bool   `json:"expand,omitempty"`
}

// Change is one thing pathdoc did. Kind is "env", "dir" or "junction".
type Change struct {
	Kind   string    `json:"kind"`
	Name   string    `json:"name,omitempty"`   // env: the variable
	Path   string    `json:"path,omitempty"`   // dir and junction
	Target string    `json:"target,omitempty"` // junction
	New    string    `json:"new,omitempty"`    // env: the value pathdoc set
	Old    Value     `json:"old"`              // env: what was there before pathdoc first touched it
	At     time.Time `json:"at"`
}

type State struct {
	Path    string   `json:"-"`
	Version int      `json:"version"`
	Root    string   `json:"root,omitempty"`
	Changes []Change `json:"changes"`
}

// StatePath is where the record lives: under the user's local app data, not
// under the root, so undo finds it whatever --root was and the root folder
// can be removed once it is empty.
func StatePath(localAppData string) string {
	return filepath.Join(localAppData, "pathdoc", "state.json")
}

// LoadState reads the record at path; a missing file is an empty record.
func LoadState(path string) (*State, error) {
	st := &State{Path: path, Version: 1}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, st); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	st.Path = path
	return st, nil
}

// Save writes the record, or deletes it (and its folder, if empty) when
// there is nothing left to undo.
func (s *State) Save() error {
	if len(s.Changes) == 0 {
		if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		os.Remove(filepath.Dir(s.Path))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
