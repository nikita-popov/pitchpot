package nginx

import (
	"os"
	"testing"
)

const testConf = `
server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://backend;
    }

    location = /favicon.ico { return 204; }

    location ^~ /static/ {
        root /var/www;
    }

    # This is a comment
    location ~ \.php$ {
        fastcgi_pass php-fpm;
    }
}
`

func TestParseLocations(t *testing.T) {
	tmpFile := t.TempDir() + "/test.conf"
	if err := os.WriteFile(tmpFile, []byte(testConf), 0644); err != nil {
		t.Fatal(err)
	}

	locs, err := ParseLocations(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(locs) != 4 {
		t.Fatalf("expected 4 locations, got %d: %v", len(locs), locs)
	}

	// Check specific entries.
	found := make(map[string]bool)
	for _, l := range locs {
		found[l.Modifier+"|"+l.Pattern] = true
	}

	cases := []struct{ mod, pat string }{
		{"", "/"},
		{"=", "/favicon.ico"},
		{"^~", "/static/"},
		{"~", `\.php$`},
	}
	for _, c := range cases {
		if !found[c.mod+"|"+c.pat] {
			t.Errorf("missing location modifier=%q pattern=%q", c.mod, c.pat)
		}
	}
}
