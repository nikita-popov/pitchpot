// Package corpus handles loading and serving deception content packs.
//
// A corpus pack is a directory with the following layout:
//
//	artifacts/      - static pre-generated files served verbatim (or slow-dripped)
//	templates/      - text/template files; runtime mutator fills in placeholders
//	manifest.json   - maps URL patterns to artifact/template paths and profiles
package corpus

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"text/template"
)

// EntryKind distinguishes artifact (static) from template (runtime-generated).
type EntryKind string

const (
	KindArtifact EntryKind = "artifact"
	KindTemplate EntryKind = "template"
)

// ManifestEntry maps a URL pattern to a content source.
type ManifestEntry struct {
	// Pattern is matched against the request path (exact or prefix).
	Pattern string `json:"pattern"`

	// Kind selects static file or runtime template.
	Kind EntryKind `json:"kind"`

	// File is a relative path inside the pack directory.
	File string `json:"file"`

	// ContentType is the MIME type to advertise in the fake response.
	ContentType string `json:"content_type"`

	// Labels are appended to the Event when this entry is matched.
	Labels []string `json:"labels,omitempty"`

	// Risk overrides the default risk level for matched events.
	Risk string `json:"risk,omitempty"`
}

// Manifest is the corpus pack index.
type Manifest struct {
	Version string          `json:"version"`
	Profile string          `json:"profile"`
	Entries []ManifestEntry `json:"entries"`
}

// Pack is a loaded, ready-to-serve corpus pack.
type Pack struct {
	Manifest  Manifest
	artifacts map[string][]byte             // file -> content
	templates map[string]*template.Template  // file -> parsed template
}

// Load reads a corpus pack from dir.
func Load(dir string) (*Pack, error) {
	mf, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("corpus: read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(mf, &m); err != nil {
		return nil, fmt.Errorf("corpus: parse manifest: %w", err)
	}

	p := &Pack{
		Manifest:  m,
		artifacts: make(map[string][]byte),
		templates: make(map[string]*template.Template),
	}

	for _, e := range m.Entries {
		path := filepath.Join(dir, e.File)
		switch e.Kind {
		case KindArtifact:
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("corpus: artifact %q: %w", e.File, err)
			}
			p.artifacts[e.File] = data
		case KindTemplate:
			tmpl, err := template.New(e.File).Funcs(templateFuncs).ParseFiles(path)
			if err != nil {
				return nil, fmt.Errorf("corpus: template %q: %w", e.File, err)
			}
			p.templates[e.File] = tmpl
		}
	}

	return p, nil
}

// Resolve returns the content bytes for a given ManifestEntry.
// Templates are executed with a fresh MutatorData seed.
func (p *Pack) Resolve(e ManifestEntry) ([]byte, error) {
	switch e.Kind {
	case KindArtifact:
		data, ok := p.artifacts[e.File]
		if !ok {
			return nil, fmt.Errorf("corpus: artifact not loaded: %s", e.File)
		}
		return data, nil
	case KindTemplate:
		tmpl, ok := p.templates[e.File]
		if !ok {
			return nil, fmt.Errorf("corpus: template not loaded: %s", e.File)
		}
		var buf []byte
		w := &appendWriter{buf: &buf}
		if err := tmpl.Execute(w, newMutatorData()); err != nil {
			return nil, fmt.Errorf("corpus: execute template %s: %w", e.File, err)
		}
		return buf, nil
	}
	return nil, fmt.Errorf("corpus: unknown kind %q", e.Kind)
}

// Match returns the first ManifestEntry whose pattern matches path.
// Returns false if no entry matches.
func (p *Pack) Match(path string) (ManifestEntry, bool) {
	for _, e := range p.Manifest.Entries {
		if matchPattern(e.Pattern, path) {
			return e, true
		}
	}
	return ManifestEntry{}, false
}

// matchPattern checks if path matches the pattern.
// Patterns ending in '/' are prefix-matched; others are exact.
func matchPattern(pattern, path string) bool {
	if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
		return len(path) >= len(pattern) && path[:len(pattern)] == pattern
	}
	return pattern == path
}

// appendWriter is an io.Writer that appends to a byte slice.
type appendWriter struct{ buf *[]byte }

func (w *appendWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// MutatorData is passed to templates at render time.
type MutatorData struct {
	Hostname  string
	IP        string
	Token     string
	Hash      string
	Date      string
	Version   string
	Branch    string
	Username  string
	Port      int
}

func newMutatorData() MutatorData {
	return MutatorData{
		Hostname: randHostname(),
		IP:       randPrivateIP(),
		Token:    randHex(32),
		Hash:     randHex(40),
		Date:     randDateStr(),
		Version:  randVersion(),
		Branch:   randBranch(),
		Username: randUsername(),
		Port:     randPort(),
	}
}

// templateFuncs are available inside all corpus templates.
var templateFuncs = template.FuncMap{
	"randHex":   func(n int) string { return randHex(n) },
	"randIP":    func() string { return randPrivateIP() },
	"pick":      func(vals ...string) string { return vals[rand.Intn(len(vals))] },
}
