package site

import "testing"

func webapp(path, siteURL string) map[string]any {
	m := map[string]any{"siteUrl": siteURL}
	if path != "" {
		m["defaultImage"] = map[string]any{"path": path}
	}
	return m
}

func TestShareImageURL(t *testing.T) {
	const rel = "2026/11/06/konex-og_uid_1.jpg"
	cases := []struct {
		name            string
		webapp          map[string]any
		adapter, public string
		want            string
	}{
		{"local storage + site url", webapp(rel, "https://konexobra.com"), "local", "", "https://konexobra.com/storage/uploads/" + rel},
		{"site url with trailing slash", webapp(rel, "https://konexobra.com/"), "", "", "https://konexobra.com/storage/uploads/" + rel},
		{"leading slash on the path", webapp("/"+rel, "https://konexobra.com"), "local", "", "https://konexobra.com/storage/uploads/" + rel},
		{"s3 public url is already absolute", webapp(rel, "https://konexobra.com"), "s3", "https://cdn.example.com/assets/", "https://cdn.example.com/assets/" + rel},
		{"s3 without public url falls back to local", webapp(rel, "https://konexobra.com"), "s3", "", "https://konexobra.com/storage/uploads/" + rel},
		{"cms already gave a full url", webapp("https://cdn.example.com/a.jpg", ""), "local", "", "https://cdn.example.com/a.jpg"},
		{"protocol-relative url", webapp("//cdn.example.com/a.jpg", ""), "local", "", "https://cdn.example.com/a.jpg"},
		{"relative image and no site url: no override", webapp(rel, ""), "local", "", ""},
		{"no image", webapp("", "https://konexobra.com"), "local", "", ""},
		{"no webapp content", nil, "local", "", ""},
	}
	for _, c := range cases {
		if got := shareImageURL(c.webapp, c.adapter, c.public); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
