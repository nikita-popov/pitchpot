package nginx

import (
	"fmt"
	"strings"
)

// HoneypotTarget is a well-known decoy path that should not exist on clean sites.
type HoneypotTarget struct {
	Pattern     string // nginx location pattern
	Modifier    string // nginx modifier ("=", "^~", "~", "")
	Label       string // event label, e.g. "probe:env"
	Risk        string // "medium", "high", "critical"
	Description string // human-readable comment
}

// DefaultTargets is the curated list of decoy locations.
// Add new targets here; the generator will skip any that already exist in the
// real nginx config.
var DefaultTargets = []HoneypotTarget{
	// --- Common / protocol-agnostic ---
	{"/.env", "=", "probe:env", "high", "dotenv config file probe"},
	{"/.env.bak", "=", "probe:env", "high", "dotenv backup probe"},
	{"/.env.local", "=", "probe:env", "high", "dotenv local override probe"},
	{"/.git/", "^~", "probe:git", "high", "git repository probe"},
	{"/.git/HEAD", "=", "probe:git", "high", "git HEAD probe"},
	{"/.git/config", "=", "probe:git", "high", "git config probe"},
	{"/.svn/", "^~", "probe:svn", "medium", "SVN repository probe"},
	{"/.DS_Store", "=", "probe:misc", "medium", "macOS DS_Store probe"},
	{"/robots.txt", "=", "probe:misc", "low", "robots.txt probe (may be legitimate)"},
	{"/security.txt", "=", "probe:misc", "low", "security.txt probe"},
	{"/.htpasswd", "=", "probe:misc", "high", "Apache htpasswd probe"},
	{"/.htaccess", "=", "probe:misc", "medium", "Apache htaccess probe"},

	// --- WordPress ---
	{"/wp-login.php", "=", "probe:wp", "high", "WordPress login probe"},
	{"/wp-admin/", "^~", "probe:wp", "high", "WordPress admin probe"},
	{"/wp-config.php", "=", "probe:wp", "critical", "WordPress config probe"},
	{"/xmlrpc.php", "=", "probe:wp", "high", "WordPress XML-RPC probe"},
	{"/wp-includes/", "^~", "probe:wp", "medium", "WordPress includes probe"},

	// --- PHP / generic CMS ---
	{"/phpinfo.php", "=", "probe:php", "high", "phpinfo probe"},
	{"/phpmyadmin/", "^~", "probe:phpmyadmin", "critical", "phpMyAdmin probe"},
	{"/pma/", "^~", "probe:phpmyadmin", "critical", "phpMyAdmin alias probe"},
	{"/adminer.php", "=", "probe:phpmyadmin", "critical", "Adminer probe"},

	// --- Composer / package managers ---
	{"/composer.json", "=", "probe:composer", "medium", "Composer manifest probe"},
	{"/composer.lock", "=", "probe:composer", "medium", "Composer lockfile probe"},
	{"/vendor/phpunit/", "^~", "probe:phpunit", "critical", "PHPUnit RCE probe"},
	{"/vendor/autoload.php", "=", "probe:composer", "high", "Composer autoload probe"},
	{"/package.json", "=", "probe:node", "medium", "npm package manifest probe"},

	// --- Spring / Java actuators ---
	{"/actuator/", "^~", "probe:actuator", "high", "Spring Boot actuator probe"},
	{"/actuator/env", "=", "probe:actuator", "critical", "Spring Boot env actuator probe"},
	{"/actuator/health", "=", "probe:actuator", "medium", "Spring Boot health probe"},

	// --- AWS / cloud ---
	{"/.aws/credentials", "=", "probe:cloud", "critical", "AWS credentials probe"},
	{"/config/database.yml", "=", "probe:rails", "high", "Rails DB config probe"},

	// --- Admin panels ---
	{"/admin/", "^~", "probe:admin", "medium", "Generic admin panel probe"},
	{"/administrator/", "^~", "probe:joomla", "medium", "Joomla admin probe"},
	{"/manage/", "^~", "probe:admin", "medium", "Generic manage panel probe"},

	// --- Router / IoT ---
	{"/boaform/", "^~", "probe:iot", "high", "Boa/router RCE probe"},
	{"/cgi-bin/", "^~", "probe:cgi", "high", "CGI probe"},

	// --- Backup / temp files ---
	{"/backup/", "^~", "probe:backup", "high", "Backup directory probe"},
	{"/dump.sql", "=", "probe:backup", "critical", "SQL dump probe"},
	{"/db.sql", "=", "probe:backup", "critical", "SQL dump probe"},
}

// GenerateInclude produces an nginx include file content that proxies
// all honeypot targets (not already present in existing) to the tarpit address.
func GenerateInclude(existing []Location, tarpitAddr string) string {
	existingSet := make(map[string]bool, len(existing))
	for _, l := range existing {
		existingSet[l.Modifier+"|"+l.Pattern] = true
	}

	var sb strings.Builder
	sb.WriteString("# Pitchpot honeypot locations — auto-generated, do not edit manually.\n")
	sb.WriteString("# Include this file inside a server {} block:\n")
	sb.WriteString("#   include /etc/pitchpot/honeypot.conf;\n\n")

	for _, t := range DefaultTargets {
		key := t.Modifier + "|" + t.Pattern
		if existingSet[key] {
			continue
		}

		// Write location block.
		locLine := buildLocationLine(t.Modifier, t.Pattern)
		sb.WriteString(fmt.Sprintf("# %s\n", t.Description))
		sb.WriteString(fmt.Sprintf("%s {\n", locLine))
		sb.WriteString(fmt.Sprintf("    proxy_pass http://%s;\n", tarpitAddr))
		sb.WriteString("    proxy_set_header X-Real-IP $remote_addr;\n")
		sb.WriteString("    proxy_set_header X-Pitchpot-Label " + t.Label + ";\n")
		sb.WriteString("    proxy_read_timeout 600s;\n")
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func buildLocationLine(modifier, pattern string) string {
	if modifier == "" {
		return "location " + pattern
	}
	return "location " + modifier + " " + pattern
}
