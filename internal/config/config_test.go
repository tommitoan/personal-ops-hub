package config

import "testing"

func TestPublicURL(t *testing.T) {
	for _, s := range []string{"http://example.com", "https://localhost/x", "https://api.internal/x", "https://user:pass@example.com"} {
		if publicURL(s) {
			t.Fatalf("accepted %s", s)
		}
	}
	if !publicURL("https://example.com/health") {
		t.Fatal("public URL rejected")
	}
}
