package site

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/Avanxo-Technology/gosite-cli/core"
)

// The waitlist is stored by Cockpit's Forms addon: every sign-up is a
// submission of the form "lista-espera" (collection formSubmissions), so it
// shows up in the CMS panel under Forms, with its date, CSV export, e-mail
// notifications and the addon's own anti-spam (honeypot, per-IP throttle).
//
// Forms does not reject duplicates, so this handler does: before forwarding it
// claims the address in r.State() with SetNX (atomic, keeps the first sign-up
// date). That key is only a guard; the data lives in Cockpit.
const (
	waitlistForm      = "lista-espera"
	waitlistKeyPrefix = "waitlist:email:"
	maxEmailLen       = 254
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

// clientIP is the visitor's address. Behind the reverse proxy RemoteAddr is
// the proxy itself, so the first X-Forwarded-For entry is used when it is a
// valid IP. Traefik replaces that header for clients it does not trust, so it
// cannot be forged through the proxy.
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

// submitError is a rejection from the Forms receiver.
type submitError struct{ status int }

func (e *submitError) Error() string { return fmt.Sprintf("forms receiver answered %d", e.status) }

// submitToCockpit posts one sign-up to the Forms addon's public receiver
// (POST /forms/api/submit) on the CMS, server to server.
//
// The visitor's address and browser travel in X-Forwarded-For and User-Agent.
// Without them the addon would see this container as the client, and its
// per-IP throttle would be shared by every visitor. cockpit/config.core.php
// sets forms.trustedProxies = 1, so the addon reads the LAST X-Forwarded-For
// entry: the one set here, never one a visitor could have sent.
func submitToCockpit(ctx context.Context, client *http.Client, baseURL, email, ip, userAgent string) error {
	body, err := json.Marshal(map[string]any{
		"form": waitlistForm,
		"data": map[string]string{"email": email},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/forms/api/submit", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Success bool `json:"success"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || !out.Success {
		return &submitError{status: resp.StatusCode}
	}
	return nil
}

// joinWaitlist handles POST /lista-espera.
func joinWaitlist(r gosite.Router) gosite.HandlerFunc {
	client := &http.Client{Timeout: 8 * time.Second}

	render := func(c *gosite.Context, status int, data map[string]any) error {
		return r.Render(c, status, "construccion", gosite.Page{
			Title:   "Próximamente | konex-landing",
			SEOData: map[string]any{"noIndex": true},
			Data:    data,
		})
	}
	fail := func(c *gosite.Context, status int, msg string) error {
		if wantsJSON(c) {
			return c.JSON(status, map[string]any{"ok": false, "error": msg})
		}
		return render(c, status, map[string]any{"error": msg})
	}
	succeed := func(c *gosite.Context) error {
		if wantsJSON(c) {
			return c.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		return render(c, http.StatusOK, map[string]any{"anotado": true})
	}

	return func(c *gosite.Context) error {
		// Honeypot: a hidden input real people never fill in. Bots get the
		// same confirmation and nothing is stored.
		if strings.TrimSpace(c.FormValue("_hp")) != "" {
			return succeed(c)
		}

		email, ok := normalizeEmail(c.FormValue("email"))
		if !ok {
			return fail(c, http.StatusBadRequest, msgInvalidEmail)
		}

		cockpitURL := os.Getenv("COCKPIT_URL")
		if cockpitURL == "" {
			slog.Error("waitlist: COCKPIT_URL is not set")
			return fail(c, http.StatusInternalServerError, msgServer)
		}

		ctx := c.Request().Context()
		guard := waitlistKeyPrefix + email

		// A repeated address is not an error: the visitor sees the same
		// confirmation, and nothing new is stored.
		first, err := r.State().SetNX(ctx, guard, time.Now().UTC().Format(time.RFC3339), 0)
		if err != nil {
			slog.Error("waitlist: state unavailable", "err", err)
			return fail(c, http.StatusInternalServerError, msgServer)
		}
		if !first {
			return succeed(c)
		}

		err = submitToCockpit(ctx, client, cockpitURL, email, clientIP(c), c.Request().UserAgent())
		if err != nil {
			// Not stored: release the address so the visitor can retry.
			_ = r.State().Del(ctx, guard)
			var se *submitError
			if errors.As(err, &se) && se.status == http.StatusTooManyRequests {
				return fail(c, http.StatusTooManyRequests, msgTooMany)
			}
			slog.Error("waitlist: could not store the sign-up in Cockpit Forms", "err", err)
			return fail(c, http.StatusBadGateway, msgServer)
		}
		return succeed(c)
	}
}
