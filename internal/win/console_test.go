package win

import (
	"os"
	"testing"
)

func TestConsoleChecksSayNoForPipes(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if IsConsole(w) {
		t.Error("IsConsole(pipe) = true, want false")
	}
	if EnableColor(w) {
		t.Error("EnableColor(pipe) = true, want false")
	}
}
