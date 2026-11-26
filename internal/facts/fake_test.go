package facts

import (
	"errors"
	"os"

	"github.com/Fd3os/pathdoc/internal/win"
)

// fakeSys is a machine made of maps: files (content), dirs, versions, env.
type fakeSys struct {
	env   map[string]string
	files map[string]string
	dirs  map[string]bool
	vers  map[string]string
	cwd   string
	cp    uint32
}

func newFake() *fakeSys {
	return &fakeSys{env: map[string]string{}, files: map[string]string{}, dirs: map[string]bool{}, vers: map[string]string{}, cwd: `C:\work`, cp: 1252}
}

func (f *fakeSys) Getenv(k string) string { return f.env[k] }
func (f *fakeSys) Getwd() (string, error) { return f.cwd, nil }
func (f *fakeSys) ReadFile(p string) ([]byte, error) {
	if c, ok := f.files[p]; ok {
		return []byte(c), nil
	}
	return nil, os.ErrNotExist
}
func (f *fakeSys) Exists(p string) bool { _, ok := f.files[p]; return ok || f.dirs[p] }
func (f *fakeSys) FileVersion(p string) (string, error) {
	if v, ok := f.vers[p]; ok {
		return v, nil
	}
	return "", errors.New("no version resource")
}
func (f *fakeSys) CodePage() uint32                    { return f.cp }
func (f *fakeSys) Represents(s string, cp uint32) bool { return win.Represents(s, cp) }
