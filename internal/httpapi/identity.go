package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/taolicus/komikan/internal/store"
)

const (
	// sessionCookieName holds the browser's secret token. It is never
	// readable by scripts.
	sessionCookieName = "komikan_session"
	// csrfCookieName holds the token the client echoes back in a header on
	// state-changing requests. It is readable by scripts on purpose: the
	// server checks it against the session token it holds.
	csrfCookieName = "komikan_csrf"
	// sessionMaxAge is how long a browser identity lasts, in seconds.
	sessionMaxAge = 60 * 60 * 24 * 365
)

// errCSRF is returned when a state-changing request fails the CSRF check.
var errCSRF = errors.New("httpapi: csrf check failed")

// playerKey carries the authenticated player through a request context.
type playerKey struct{}

// playerFrom returns the player attached to a request context.
func playerFrom(ctx context.Context) (store.Player, bool) {
	p, ok := ctx.Value(playerKey{}).(store.Player)
	return p, ok
}

// newToken returns a random 256-bit credential.
func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashToken returns the hash stored in the database. The raw token is never
// persisted.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// csrfFor derives the CSRF token of a session. The server can always
// recompute it from the session token it receives, so nothing extra has to
// be stored.
func csrfFor(session string) string {
	mac := hmac.New(sha256.New, []byte(session))
	mac.Write([]byte("komikan-csrf"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// secureRequest reports whether the client reached the server over TLS,
// directly or through a proxy that sets X-Forwarded-Proto.
func secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// setIdentityCookies hands the browser its credentials.
func (s *Server) setIdentityCookies(w http.ResponseWriter, r *http.Request, session string) {
	secure := secureRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionMaxAge,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfFor(session),
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionMaxAge,
	})
}

// identify resolves the browser identity, creating one when the browser has
// none yet or presents a token the server does not know. The returned bool
// tells whether the browser already had a session.
func (s *Server) identify(w http.ResponseWriter, r *http.Request) (store.Player, string, bool, error) {
	ctx := r.Context()

	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		hash := hashToken(cookie.Value)
		player, err := s.store.PlayerByTokenHash(ctx, hash)
		if err == nil {
			return player, cookie.Value, true, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return store.Player{}, "", false, err
		}
		// An unknown token means a fresh database or a cleared session: the
		// browser gets a new identity.
	}

	session, err := newToken()
	if err != nil {
		return store.Player{}, "", false, err
	}
	player, err := s.store.CreatePlayer(ctx, hashToken(session))
	if err != nil {
		return store.Player{}, "", false, err
	}
	s.setIdentityCookies(w, r, session)
	return player, session, false, nil
}

// withIdentity attaches a player to the request and runs the CSRF check on
// state-changing requests.
func (s *Server) withIdentity(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		player, session, hadSession, err := s.identify(w, r)
		if err != nil {
			s.writeError(w, err)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if err := s.checkCSRF(r, session, hadSession); err != nil {
				s.writeError(w, err)
				return
			}
		}

		ctx := context.WithValue(r.Context(), playerKey{}, player)
		next(w, r.WithContext(ctx))
	}
}

// checkCSRF rejects a state-changing request that does not prove it comes
// from the same site.
//
// A browser that already has a session echoes the CSRF cookie in a header,
// and the server compares it with the token derived from the session it
// holds. A browser that has no session yet is a first contact: it has no
// token to send, so the Origin header is the proof, and browsers always send
// it on a write.
func (s *Server) checkCSRF(r *http.Request, session string, hadSession bool) error {
	origin := r.Header.Get("Origin")
	if origin != "" && !sameOrigin(origin, r.Host) {
		return errCSRF
	}
	if hadSession {
		header := r.Header.Get("X-CSRF-Token")
		if header == "" {
			return errCSRF
		}
		if !hmac.Equal([]byte(header), []byte(csrfFor(session))) {
			return errCSRF
		}
		return nil
	}
	if origin == "" {
		return errCSRF
	}
	return nil
}

// sameOrigin reports whether an Origin header points at the request host.
func sameOrigin(origin, host string) bool {
	trimmed := strings.TrimSuffix(origin, "/")
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.EqualFold(strings.TrimPrefix(trimmed, prefix), host)
		}
	}
	return false
}
