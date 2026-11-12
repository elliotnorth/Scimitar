package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPasswordHashMatches(t *testing.T) {
	h, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Matches("s3cret") || h.Matches("s3cret ") || h.Matches("") {
		t.Fatal("hash does not match exactly the original password")
	}
	other, _ := HashPassword("s3cret")
	if string(other.Salt) == string(h.Salt) || string(other.Hash) == string(h.Hash) {
		t.Fatal("salt is not random")
	}
}

func TestLoadOrCreateGeneratesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	hash, generated, err := LoadOrCreate(path)
	if err != nil || len(generated) < 20 || !hash.Matches(generated) {
		t.Fatalf("first load: %q %v", generated, err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("auth file mode %v", info.Mode().Perm())
	}
	again, generatedAgain, err := LoadOrCreate(path)
	if err != nil || generatedAgain != "" || !again.Matches(generated) {
		t.Fatalf("second load: %q %v", generatedAgain, err)
	}
}

func TestResetReplacesPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	_, first, _ := LoadOrCreate(path)
	second, err := Reset(path)
	if err != nil {
		t.Fatal(err)
	}
	hash, _, _ := LoadOrCreate(path)
	if hash.Matches(first) || !hash.Matches(second) {
		t.Fatal("reset did not replace the password")
	}
}

func TestLoadOrCreateRejectsBadFile(t *testing.T) {
	for _, body := range []string{"{", "{}"} {
		path := filepath.Join(t.TempDir(), "auth.json")
		os.WriteFile(path, []byte(body), 0o600)
		if _, _, err := LoadOrCreate(path); err == nil {
			t.Errorf("%q: expected error", body)
		}
	}
}

func newTestGuard(t *testing.T) (*Guard, *time.Time, *int) {
	t.Helper()
	hash, err := HashPassword("pw")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sleeps := 0
	g := NewGuard(hash, time.Hour)
	g.now = func() time.Time { return now }
	g.sleep = func(time.Duration) { sleeps++ }
	return g, &now, &sleeps
}

func TestGuardSessions(t *testing.T) {
	g, now, _ := newTestGuard(t)
	token, ok, err := g.Login("pw")
	if err != nil || !ok || token == "" {
		t.Fatalf("login: %q %v %v", token, ok, err)
	}
	if !g.Valid(token) || g.Valid(token+"x") || g.Valid("") {
		t.Fatal("token validation wrong")
	}
	*now = now.Add(time.Hour)
	if g.Valid(token) {
		t.Fatal("session did not expire")
	}
}

func TestGuardLogout(t *testing.T) {
	g, _, _ := newTestGuard(t)
	token, _, _ := g.Login("pw")
	g.Logout(token)
	if g.Valid(token) {
		t.Fatal("session survived logout")
	}
}

func TestGuardSlowsFailedLogins(t *testing.T) {
	g, _, sleeps := newTestGuard(t)
	if _, ok, _ := g.Login("wrong"); ok {
		t.Fatal("wrong password accepted")
	}
	if *sleeps != 1 {
		t.Fatalf("failed login was not delayed")
	}
	g.Login("pw")
	if *sleeps != 1 {
		t.Fatalf("successful login was delayed")
	}
}

func TestGuardPrunesExpiredSessions(t *testing.T) {
	g, now, _ := newTestGuard(t)
	g.Login("pw")
	*now = now.Add(2 * time.Hour)
	g.Login("pw")
	if len(g.sessions) != 1 {
		t.Fatalf("expired session kept: %d", len(g.sessions))
	}
}
