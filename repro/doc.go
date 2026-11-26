// Package repro proves every rule on a real Windows machine: each tool runs
// in three folders that differ only in their name — plain, café (non-ASCII
// that code page 1252 holds) and ابوي (that it does not) — and again after
// the cure. The tests carry the "repro" build tag:
//
//	go test -tags repro ./repro/ -v
//
// With PATHDOC_REPRO_STRICT=1 (CI) a missing tool fails instead of skipping.
package repro
