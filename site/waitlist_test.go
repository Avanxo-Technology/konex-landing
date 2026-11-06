package site

import "testing"

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
