package site

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	good := map[string]string{
		"  Ana@Obra.CO ":       "ana@obra.co",
		"a.b+c@constru.com.co": "a.b+c@constru.com.co",
	}
	for in, want := range good {
		if got, ok := normalizeEmail(in); !ok || got != want {
			t.Errorf("normalizeEmail(%q) = %q, %v; want %q, true", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "   ", "sin-arroba", "a@b", "a@b.", "Ana <ana@obra.co>", "a@@b.co", "a b@obra.co"} {
		if got, ok := normalizeEmail(in); ok {
			t.Errorf("normalizeEmail(%q) = %q, true; want invalid", in, got)
		}
	}
}

func TestSubmitToCockpit(t *testing.T) {
	var got struct {
		path, method, xff, ua, ctype string
		body                         map[string]any
	}
	status, answer := http.StatusOK, `{"success":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.method = r.URL.Path, r.Method
		got.xff, got.ua, got.ctype = r.Header.Get("X-Forwarded-For"), r.UserAgent(), r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()
	ctx := context.Background()

	if err := submitToCockpit(ctx, srv.Client(), srv.URL+"/", "ana@obra.co", "203.0.113.7", "Mozilla/5.0 test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.path != "/forms/api/submit" || got.method != http.MethodPost || got.ctype != "application/json" {
		t.Errorf("wrong request: %s %s (%s)", got.method, got.path, got.ctype)
	}
	if got.xff != "203.0.113.7" || got.ua != "Mozilla/5.0 test" {
		t.Errorf("visitor not forwarded: xff=%q ua=%q", got.xff, got.ua)
	}
	data, _ := got.body["data"].(map[string]any)
	if got.body["form"] != "lista-espera" || data["email"] != "ana@obra.co" {
		t.Errorf("wrong payload: %v", got.body)
	}

	// The receiver's rejections surface with their status, so the handler can tell 429 from the rest.
	status, answer = http.StatusTooManyRequests, `{"success":false,"error":"slow down"}`
	var se *submitError
	if err := submitToCockpit(ctx, srv.Client(), srv.URL, "a@b.co", "203.0.113.7", ""); !errors.As(err, &se) || se.status != 429 {
		t.Errorf("429 not reported: %v", err)
	}
	// A 200 that does not say success is a failure too.
	status, answer = http.StatusOK, `{"success":false}`
	if err := submitToCockpit(ctx, srv.Client(), srv.URL, "a@b.co", "203.0.113.7", ""); err == nil {
		t.Error("200 without success must be an error")
	}
	// Unreachable receiver.
	srv.Close()
	if err := submitToCockpit(ctx, http.DefaultClient, srv.URL, "a@b.co", "203.0.113.7", ""); err == nil {
		t.Error("unreachable receiver must be an error")
	}
}
