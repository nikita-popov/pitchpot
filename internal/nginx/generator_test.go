package nginx

import (
	"strings"
	"testing"
)

func TestGenerateInclude_skipsExisting(t *testing.T) {
	existing := []Location{
		{Modifier: "=", Pattern: "/.env"},
		{Modifier: "^~", Pattern: "/wp-admin/"},
	}

	out := GenerateInclude(existing, "127.0.0.1:9999")

	if strings.Contains(out, "location = /.env") {
		t.Error("existing location /.env should be skipped")
	}
	if strings.Contains(out, "location ^~ /wp-admin/") {
		t.Error("existing location /wp-admin/ should be skipped")
	}
	// Non-existing target should be present.
	if !strings.Contains(out, "/wp-login.php") {
		t.Error("expected /wp-login.php in output")
	}
}

func TestGenerateInclude_containsProxyPass(t *testing.T) {
	out := GenerateInclude(nil, "127.0.0.1:9999")
	if !strings.Contains(out, "proxy_pass http://127.0.0.1:9999") {
		t.Error("expected proxy_pass directive")
	}
}
