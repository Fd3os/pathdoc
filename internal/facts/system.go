package facts

import (
	"os"

	"github.com/Fd3os/pathdoc/internal/win"
)

type realSystem struct{}

// Real reads this machine.
func Real() System { return realSystem{} }

func (realSystem) Getenv(k string) string               { return os.Getenv(k) }
func (realSystem) Getwd() (string, error)               { return os.Getwd() }
func (realSystem) ReadFile(p string) ([]byte, error)    { return os.ReadFile(p) }
func (realSystem) FileVersion(p string) (string, error) { return win.FileVersion(p) }
func (realSystem) CodePage() uint32                     { return win.ActiveCodePage() }
func (realSystem) Represents(s string, cp uint32) bool  { return win.Represents(s, cp) }
func (realSystem) Exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
