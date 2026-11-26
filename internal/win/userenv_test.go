package win

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// Real registry, on a throwaway key — never HKCU\Environment.
func TestUserEnvRoundTrip(t *testing.T) {
	u := UserEnv{Key: `Software\pathdoc-test\Environment`}
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, `Software\pathdoc-test\Environment`)
		registry.DeleteKey(registry.CURRENT_USER, `Software\pathdoc-test`)
	})
	if v, err := u.Get("PATHDOC_X"); err != nil || v.Exists {
		t.Fatalf("missing key: %+v %v", v, err)
	}
	want := EnvValue{Exists: true, Data: `%USERPROFILE%\ابوي`, Expand: true}
	if err := u.Set("PATHDOC_X", want); err != nil {
		t.Fatal(err)
	}
	if got, err := u.Get("PATHDOC_X"); err != nil || got != want {
		t.Fatalf("expand value = %+v, %v; want it raw, not expanded", got, err)
	}
	plain := EnvValue{Exists: true, Data: `C:\dev\temp`}
	u.Set("PATHDOC_X", plain)
	if got, _ := u.Get("PATHDOC_X"); got != plain {
		t.Fatalf("plain value = %+v", got)
	}
	if err := u.Delete("PATHDOC_X"); err != nil {
		t.Fatal(err)
	}
	if got, _ := u.Get("PATHDOC_X"); got.Exists {
		t.Fatal("deleted value still there")
	}
	if err := u.Delete("PATHDOC_X"); err != nil {
		t.Fatalf("deleting a missing value: %v", err)
	}
}
