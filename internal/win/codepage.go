// Package win holds the few Windows calls pathdoc needs: the active code page
// and whether a string survives it, file versions, the console, junctions and
// the user's environment in the registry.
package win

import (
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procWideCharToMultiByte = windows.NewLazySystemDLL("kernel32.dll").NewProc("WideCharToMultiByte")

const (
	cpUTF7 = 65000
	cpUTF8 = 65001
)

// ActiveCodePage is the ANSI code page Windows hands to programs that are not
// Unicode-aware. 65001 means the "Beta: Use Unicode UTF-8" switch is on.
func ActiveCodePage() uint32 { return windows.GetACP() }

// Represents reports whether s comes back unchanged after a trip through code
// page cp. A character with no exact mapping fails, including one Windows
// would quietly swap for a look-alike ("ā" becomes "a"): a program that sees
// the swapped name is looking for a path that does not exist.
func Represents(s string, cp uint32) bool {
	if cp == cpUTF8 || cp == cpUTF7 || isASCII(s) {
		return true
	}
	wide := utf16.Encode([]rune(s))
	narrow := make([]byte, len(wide)*4)
	n, _, _ := procWideCharToMultiByte.Call(uintptr(cp), 0,
		uintptr(unsafe.Pointer(&wide[0])), uintptr(len(wide)),
		uintptr(unsafe.Pointer(&narrow[0])), uintptr(len(narrow)), 0, 0)
	if n == 0 {
		return false
	}
	back := make([]uint16, len(wide)*2)
	m, err := windows.MultiByteToWideChar(cp, 0, &narrow[0], int32(n), &back[0], int32(len(back)))
	if err != nil || m == 0 {
		return false
	}
	return string(utf16.Decode(back[:m])) == s
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
