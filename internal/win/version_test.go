package win

import (
	"regexp"
	"testing"
)

func TestFileVersion(t *testing.T) {
	v, err := FileVersion(`C:\Windows\System32\kernel32.dll`)
	if err != nil {
		t.Fatalf("FileVersion(kernel32) error: %v", err)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(v) {
		t.Fatalf("FileVersion(kernel32) = %q, want four numbers", v)
	}
	if _, err := FileVersion(`C:\Windows\System32\does-not-exist.dll`); err == nil {
		t.Fatal("FileVersion of a missing file: want an error")
	}
}
