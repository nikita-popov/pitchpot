package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_artifact(t *testing.T) {
	dir := t.TempDir()

	// Write a minimal manifest and one artifact.
	manifest := `{
		"version": "1",
		"profile": "test",
		"entries": [
			{"pattern": "/.env", "kind": "artifact", "file": "artifacts/env.txt",
			 "content_type": "text/plain", "labels": ["probe:env"], "risk": "high"}
		]
	}`

	_ = os.MkdirAll(filepath.Join(dir, "artifacts"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)
	_ = os.WriteFile(filepath.Join(dir, "artifacts/env.txt"), []byte("APP_KEY=fake\n"), 0644)

	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	e, ok := p.Match("/.env")
	if !ok {
		t.Fatal("expected /.env to match")
	}

	data, err := p.Resolve(e)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "APP_KEY=fake\n" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestMatch_prefix(t *testing.T) {
	p := &Pack{
		Manifest: Manifest{
			Entries: []ManifestEntry{
				{Pattern: "/.git/", Kind: KindArtifact, File: "artifacts/git-head.txt"},
			},
		},
		artifacts: map[string][]byte{
			"artifacts/git-head.txt": []byte("ref: refs/heads/main\n"),
		},
	}

	e, ok := p.Match("/.git/HEAD")
	if !ok {
		t.Fatal("expected /.git/HEAD to match prefix /.git/")
	}
	if e.File != "artifacts/git-head.txt" {
		t.Errorf("unexpected file: %s", e.File)
	}
}
