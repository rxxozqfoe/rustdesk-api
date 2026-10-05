package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePasswordFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write password file: %v", err)
	}
	return p
}

func TestAdminLoadPassword_Unset(t *testing.T) {
	a := &Admin{}
	if err := a.LoadPassword(); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if a.Password != "" {
		t.Errorf("got password %q, want empty", a.Password)
	}
}

func TestAdminLoadPassword_FromFileTrimsNewline(t *testing.T) {
	a := &Admin{PasswordFile: writePasswordFile(t, "fromsecret\n")}
	if err := a.LoadPassword(); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if a.Password != "fromsecret" {
		t.Errorf("got password %q, want %q", a.Password, "fromsecret")
	}
}

func TestAdminLoadPassword_ValueWinsOverFile(t *testing.T) {
	a := &Admin{Password: "fromvalue", PasswordFile: writePasswordFile(t, "fromfile")}
	if err := a.LoadPassword(); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if a.Password != "fromvalue" {
		t.Errorf("got password %q, want %q", a.Password, "fromvalue")
	}
}

func TestAdminLoadPassword_Errors(t *testing.T) {
	cases := map[string]*Admin{
		"missing file": {PasswordFile: filepath.Join(t.TempDir(), "absent")},
		"empty file":   {PasswordFile: writePasswordFile(t, "\n")},
		"too short":    {Password: strings.Repeat("a", AdminPasswordMinLen-1)},
		"too long":     {Password: strings.Repeat("a", AdminPasswordMaxLen+1)},
	}
	for name, a := range cases {
		t.Run(name, func(t *testing.T) {
			if err := a.LoadPassword(); err == nil {
				t.Error("got nil, want an error")
			}
		})
	}
}

// Bounds are inclusive and count characters, not bytes, like the forms do.
func TestAdminLoadPassword_LengthBoundsAreInclusive(t *testing.T) {
	for _, pwd := range []string{
		strings.Repeat("密", AdminPasswordMinLen),
		strings.Repeat("a", AdminPasswordMaxLen),
	} {
		a := &Admin{Password: pwd}
		if err := a.LoadPassword(); err != nil {
			t.Errorf("%q: got %v, want nil", pwd, err)
		}
	}
}
