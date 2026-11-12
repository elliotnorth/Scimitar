package auth

import (
	"crypto/sha256"
	"sync"
	"time"
)

const (
	sessionTokenBytes = 32
	failedLoginDelay  = time.Second
)

type Guard struct {
	password PasswordHash
	ttl      time.Duration
	now      func() time.Time
	sleep    func(time.Duration)

	loginMu  sync.Mutex
	mu       sync.Mutex
	sessions map[[sha256.Size]byte]time.Time
}

func NewGuard(password PasswordHash, sessionTTL time.Duration) *Guard {
	return &Guard{
		password: password,
		ttl:      sessionTTL,
		now:      time.Now,
		sleep:    time.Sleep,
		sessions: map[[sha256.Size]byte]time.Time{},
	}
}

func (g *Guard) SessionTTL() time.Duration {
	return g.ttl
}

// Logins run one at a time and failures are slowed down, which caps password guessing at about one try per second.
func (g *Guard) Login(password string) (token string, ok bool, err error) {
	g.loginMu.Lock()
	defer g.loginMu.Unlock()
	if !g.password.Matches(password) {
		g.sleep(failedLoginDelay)
		return "", false, nil
	}
	token, err = randomToken(sessionTokenBytes)
	if err != nil {
		return "", false, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pruneExpired()
	g.sessions[sha256.Sum256([]byte(token))] = g.now().Add(g.ttl)
	return token, true, nil
}

func (g *Guard) Valid(token string) bool {
	if token == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	expires, ok := g.sessions[sha256.Sum256([]byte(token))]
	return ok && g.now().Before(expires)
}

func (g *Guard) Logout(token string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.sessions, sha256.Sum256([]byte(token)))
}

func (g *Guard) pruneExpired() {
	now := g.now()
	for key, expires := range g.sessions {
		if !now.Before(expires) {
			delete(g.sessions, key)
		}
	}
}
