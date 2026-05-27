// Package nginx provides a minimal nginx config location extractor.
//
// It does NOT implement a full nginx config AST. It performs line-oriented
// scanning sufficient to extract declared location paths from server blocks.
// Limitations: handles one level of block nesting inside server {}; does not
// resolve include directives recursively by default.
package nginx

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Location represents a parsed nginx location directive.
type Location struct {
	Modifier string // "", "=", "~", "~*", "^~"
	Pattern  string // the path or regex pattern
}

// String returns the location as it would appear in an nginx config.
func (l Location) String() string {
	if l.Modifier == "" {
		return "location " + l.Pattern
	}
	return "location " + l.Modifier + " " + l.Pattern
}

// ParseLocations extracts all location patterns from the file at path.
// It returns a deduplicated slice of Location values.
func ParseLocations(path string) ([]Location, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("nginx parser: open %q: %w", path, err)
	}
	defer f.Close()

	var locations []Location
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Strip inline comments.
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		loc, ok := parseLocationLine(line)
		if !ok {
			continue
		}

		key := loc.Modifier + " " + loc.Pattern
		if !seen[key] {
			seen[key] = true
			locations = append(locations, loc)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("nginx parser: scan: %w", err)
	}
	return locations, nil
}

// locationRe matches: location [modifier] pattern {
// modifier is optional: = | ~ | ~* | ^~
var locationRe = regexp.MustCompile(`^location\s+(=|~\*|~|\^~)?\s*([^\s{]+)\s*\{?`)

func parseLocationLine(line string) (Location, bool) {
	m := locationRe.FindStringSubmatch(line)
	if m == nil {
		return Location{}, false
	}
	return Location{
		Modifier: strings.TrimSpace(m[1]),
		Pattern:  m[2],
	}, true
}
