package win

import "testing"

func TestRepresents(t *testing.T) {
	cases := []struct {
		s    string
		cp   uint32
		want bool
	}{
		{"plain", 1252, true},
		{"café", 1252, true},
		{"ابوي", 1252, false},
		{"ابوي", 1256, true},
		{"张伟", 936, true},
		{"张伟", 1252, false},
		{"ě", 1252, false},
		{"ě", 1250, true},
		{"ā", 1252, false}, // best-fit would turn it into "a" — a different path
		{"ابوي", 65001, true},
		{"", 1252, true},
	}
	for _, c := range cases {
		if got := Represents(c.s, c.cp); got != c.want {
			t.Errorf("Represents(%q, %d) = %v, want %v", c.s, c.cp, got, c.want)
		}
	}
}

func TestActiveCodePage(t *testing.T) {
	if cp := ActiveCodePage(); cp == 0 {
		t.Fatal("ActiveCodePage returned 0")
	}
}
