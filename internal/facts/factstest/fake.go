// Package factstest is a machine made of maps, for tests: files (content),
// folders, version resources, environment, current folder and code page.
package factstest

import (
	"errors"
	"os"

	"github.com/Fd3os/pathdoc/internal/win"
)

type Fake struct {
	Env   map[string]string
	Files map[string]string
	Dirs  map[string]bool
	Vers  map[string]string
	Cwd   string
	CP    uint32
}

func New() *Fake {
	return &Fake{Env: map[string]string{}, Files: map[string]string{}, Dirs: map[string]bool{}, Vers: map[string]string{}, Cwd: `C:\work`, CP: 1252}
}

func (f *Fake) Getenv(k string) string { return f.Env[k] }
func (f *Fake) Getwd() (string, error) { return f.Cwd, nil }
func (f *Fake) ReadFile(p string) ([]byte, error) {
	if c, ok := f.Files[p]; ok {
		return []byte(c), nil
	}
	return nil, os.ErrNotExist
}
func (f *Fake) Exists(p string) bool {
	_, ok := f.Files[p]
	return ok || f.Dirs[p]
}
func (f *Fake) FileVersion(p string) (string, error) {
	if v, ok := f.Vers[p]; ok {
		return v, nil
	}
	return "", errors.New("no version resource")
}
func (f *Fake) CodePage() uint32                    { return f.CP }
func (f *Fake) Represents(s string, cp uint32) bool { return win.Represents(s, cp) }
