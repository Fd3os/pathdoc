package win

import (
	"os"
	"path/filepath"
	"testing"
)

// A folder pathdoc makes for TEMP or a Gradle home must not be writable by
// other accounts on the PC: C:\ hands Modify to every authenticated user.
func TestMakePrivateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := MakePrivateDir(dir); err != nil {
		t.Fatal(err)
	}
	ok, err := IsPrivate(dir)
	if err != nil || !ok {
		t.Fatalf("IsPrivate = %v, %v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatalf("the owner must still write: %v", err)
	}
	if ok, _ := IsPrivate(t.TempDir()); ok {
		t.Log("note: this temp folder already inherits a private DACL")
	}
	if err := MakePrivateDir(dir); err == nil {
		t.Fatal("an existing folder: want an error")
	}
}

func TestIsPrivateSaysNoForCDrive(t *testing.T) {
	if ok, err := IsPrivate(`C:\`); err != nil || ok {
		t.Fatalf(`IsPrivate(C:\) = %v, %v; want false`, ok, err)
	}
}
