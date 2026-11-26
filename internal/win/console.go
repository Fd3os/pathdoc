package win

import (
	"os"

	"golang.org/x/sys/windows"
)

// IsConsole reports whether f is a console, not a pipe or a file.
func IsConsole(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

// EnableColor turns on escape-sequence processing for f's console. It returns
// false when f is not a console or the console refuses (older Windows 10).
func EnableColor(f *os.File) bool {
	h := windows.Handle(f.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
