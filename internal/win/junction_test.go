package win

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJunctionRoundTrip(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "ابوي")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "a.txt"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := CreateJunction(link, target); err != nil {
		t.Fatalf("CreateJunction: %v", err)
	}
	if err := os.WriteFile(filepath.Join(link, "b.txt"), []byte("via link"), 0o644); err != nil {
		t.Fatalf("write through the junction: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(target, "b.txt")); err != nil || string(b) != "via link" {
		t.Fatalf("read from target: %q %v", b, err)
	}
	if got, err := JunctionTarget(link); err != nil || got != target {
		t.Fatalf("JunctionTarget = %q, %v; want %q", got, err, target)
	}
	if err := os.Remove(link); err != nil {
		t.Fatalf("remove junction: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "a.txt")); err != nil {
		t.Fatalf("removing the junction touched the target: %v", err)
	}
}

func TestJunctionTargetOfDanglingJunction(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "gone")
	os.Mkdir(target, 0o755)
	link := filepath.Join(base, "link")
	if err := CreateJunction(link, target); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(link)
	os.Remove(target)
	if got, err := JunctionTarget(link); err != nil || got != target {
		t.Fatalf("dangling: JunctionTarget = %q, %v", got, err)
	}
}

func TestJunctionTargetOfPlainFolderFails(t *testing.T) {
	if _, err := JunctionTarget(t.TempDir()); err == nil {
		t.Fatal("a plain folder is not a junction: want an error")
	}
}

func TestCreateJunctionNeverReplaces(t *testing.T) {
	base := t.TempDir()
	taken := filepath.Join(base, "taken")
	os.Mkdir(taken, 0o755)
	if err := CreateJunction(taken, base); err == nil {
		t.Fatal("an existing folder at the link path: want an error")
	}
}
