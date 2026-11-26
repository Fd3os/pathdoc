package rules

import (
	"strconv"
	"strings"

	"github.com/Fd3os/pathdoc/internal/facts"
)

// Evaluate returns one finding per rule that fires, in catalog order.
func Evaluate(f *facts.Facts, catalog []Rule) []Finding {
	var out []Finding
	for _, r := range catalog {
		for _, key := range r.Locations {
			where, ok := f.Locations[key]
			if !ok {
				continue
			}
			bad := Offending(where.Path, r.Trigger, f.Represents)
			if len(bad) == 0 || !r.Applies(f, where) {
				continue
			}
			out = append(out, Finding{Rule: r, Where: where, Offending: bad})
			break
		}
	}
	return out
}

// Offending lists, once each and in order, the path segments that break a
// tool with trigger t.
func Offending(path string, t Trigger, represents func(string) bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, seg := range strings.FieldsFunc(path, func(r rune) bool { return r == '\\' || r == '/' }) {
		bad := false
		if t == AnyNonASCII {
			bad = !isASCII(seg)
		} else {
			bad = !represents(seg)
		}
		if bad && !seen[seg] {
			seen[seg] = true
			out = append(out, seg)
		}
	}
	return out
}

// Less compares dotted version numbers. An unknown version ("") is never
// less: a rule gated on a version stays silent rather than guess.
func Less(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		x, y := num(pa, i), num(pb, i)
		if x != y {
			return x < y
		}
	}
	return false
}

func num(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	s := parts[i]
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(s[:end])
	return n
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
