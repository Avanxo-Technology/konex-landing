package site

import (
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/Avanxo-Technology/gosite-cli/core"
)

const (
	// Waitlist entries live in r.State() (Redis, never purged):
	//   waitlist:email:<address>  ->  RFC 3339 UTC time of the first sign-up
	// SetNX makes the address unique and keeps the first date.
	waitlistKeyPrefix = "waitlist:email:"

	// At most waitlistMaxPerHour sign-up attempts per IP per hour.
	waitlistRateKeyPrefix = "waitlist:rl:"
	waitlistMaxPerHour    = 10

	maxEmailLen = 254
)

const (
	msgInvalidEmail = "Escribe un correo válido."
	msgTooMany      = "Demasiados intentos. Intenta más tarde."
	msgServer       = "No pudimos guardar tu correo. Intenta de nuevo."
)

// normalizeEmail returns the lower-cased address, or false when s is not a
// plain "user@host.tld" address ("Name <a@b.c>" is rejected).
func normalizeEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > maxEmailLen {
		return "", false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", false
	}
	_, domain, _ := strings.Cut(s, "@")
	if !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", false
	}
	return s, true
}

// clientIP is the visitor's address for rate limiting. Behind the reverse
// proxy RemoteAddr is the proxy itself (every visitor would share one counter),
// so the first X-Forwarded-For entry is used when it is a valid IP. Traefik
// replaces that header for clients it does not trust, so it cannot be forged
// through the proxy.
func clientIP(c *gosite.Context) string {
	req := c.Request()
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// wantsJSON is true for the page's fetch() call; a plain form POST gets the
// page back instead.
func wantsJSON(c *gosite.Context) bool {
	return strings.Contains(c.Request().Header.Get("Accept"), "application/json")
}

// joinWaitlist handles POST /lista-espera.
func joinWaitlist(r gosite.Router) gosite.HandlerFunc {
	fail := func(c *gosite.Context, status int, msg string) error {
		if wantsJSON(c) {
			return c.JSON(status, map[string]any{"ok": false, "error": msg})
		}
		return r.Render(c, status, "construccion", gosite.Page{
			Title:   "Próximamente | konex-landing",
			SEOData: map[string]any{"noIndex": true},
			Data:    map[string]any{"error": msg},
		})
	}

	return func(c *gosite.Context) error {
		ctx := c.Request().Context()
		st := r.State()

		// Rate limit first, so junk cannot fill Redis.
		rateKey := waitlistRateKeyPrefix + clientIP(c)
		n, err := st.Incr(ctx, rateKey)
		if err != nil {
			return fail(c, http.StatusInternalServerError, msgServer)
		}
		if n == 1 {
			_ = st.Expire(ctx, rateKey, time.Hour)
		}
		if n > waitlistMaxPerHour {
			return fail(c, http.StatusTooManyRequests, msgTooMany)
		}

		email, ok := normalizeEmail(c.FormValue("email"))
		if !ok {
			return fail(c, http.StatusBadRequest, msgInvalidEmail)
		}

		// A repeated address is not an error: the visitor sees the same
		// confirmation and the first sign-up date is kept.
		if _, err := st.SetNX(ctx, waitlistKeyPrefix+email, time.Now().UTC().Format(time.RFC3339), 0); err != nil {
			return fail(c, http.StatusInternalServerError, msgServer)
		}

		if wantsJSON(c) {
			return c.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		return r.Render(c, http.StatusOK, "construccion", gosite.Page{
			Title:   "Próximamente | konex-landing",
			SEOData: map[string]any{"noIndex": true},
			Data:    map[string]any{"anotado": true},
		})
	}
}
