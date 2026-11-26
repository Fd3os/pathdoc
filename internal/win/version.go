package win

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FileVersion reads the fixed version from an executable's version resource,
// as "major.minor.build.revision". It never runs the file.
func FileVersion(path string) (string, error) {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil {
		return "", err
	}
	buf := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return "", err
	}
	var fixed *windows.VS_FIXEDFILEINFO
	var n uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\`, unsafe.Pointer(&fixed), &n); err != nil {
		return "", err
	}
	if fixed == nil || n == 0 {
		return "", errors.New("no fixed version info in " + path)
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		fixed.FileVersionMS>>16, fixed.FileVersionMS&0xffff,
		fixed.FileVersionLS>>16, fixed.FileVersionLS&0xffff), nil
}
