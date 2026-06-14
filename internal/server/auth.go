package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "frp_session"
const sessionTTL = 7 * 24 * time.Hour

// AuthManager holds active panel sessions in memory. Sessions do not survive a
// restart, which is acceptable for a single-operator admin panel.
type AuthManager struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

// NewAuthManager creates an empty session store.
func NewAuthManager() *AuthManager {
	return &AuthManager{sessions: make(map[string]time.Time)}
}

// Create issues a new session token.
func (m *AuthManager) Create() string {
	tok := randToken()
	m.mu.Lock()
	m.sessions[tok] = time.Now().Add(sessionTTL)
	m.mu.Unlock()
	return tok
}

// Valid reports whether tok is a live session, pruning it if expired.
func (m *AuthManager) Valid(tok string) bool {
	if tok == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.sessions[tok]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(m.sessions, tok)
		return false
	}
	return true
}

// Revoke drops a single session.
func (m *AuthManager) Revoke(tok string) {
	m.mu.Lock()
	delete(m.sessions, tok)
	m.mu.Unlock()
}

// RevokeAll invalidates every session (used after a password change).
func (m *AuthManager) RevokeAll() {
	m.mu.Lock()
	m.sessions = make(map[string]time.Time)
	m.mu.Unlock()
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HashPassword returns a bcrypt hash of pw.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

// CheckPassword reports whether pw matches the bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func setSessionCookie(w http.ResponseWriter, tok string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
