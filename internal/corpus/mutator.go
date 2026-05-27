package corpus

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	wordParts = []string{
		"app", "api", "web", "srv", "db", "cache", "proxy", "edge",
		"node", "prod", "dev", "stage", "worker", "queue", "auth",
	}
	usernames = []string{
		"admin", "deploy", "ubuntu", "www-data", "app", "service",
		"operator", "user", "root", "vagrant",
	}
	branches = []string{
		"main", "master", "develop", "release/2.1", "hotfix/sec-patch",
		"feature/auth-refactor", "refs/heads/main",
	}
	versions = []string{
		"1.0.0", "1.2.3", "2.0.0-beta", "3.1.4", "0.9.7",
		"4.2.1", "1.18.0", "7.4.33", "8.1.12",
	}
)

func randHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}

func randPrivateIP() string {
	return fmt.Sprintf("10.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(254)+1)
}

func randHostname() string {
	a := wordParts[rand.Intn(len(wordParts))]
	b := wordParts[rand.Intn(len(wordParts))]
	n := rand.Intn(90) + 10
	return fmt.Sprintf("%s-%s-%d", a, b, n)
}

func randUsername() string {
	return usernames[rand.Intn(len(usernames))]
}

func randBranch() string {
	return branches[rand.Intn(len(branches))]
}

func randVersion() string {
	return versions[rand.Intn(len(versions))]
}

func randDateStr() string {
	// Random date within the last 2 years.
	now := time.Now()
	delta := time.Duration(rand.Int63n(int64(365 * 2 * 24 * time.Hour)))
	return now.Add(-delta).UTC().Format("2006-01-02T15:04:05Z")
}

func randPort() int {
	// Common internal service ports.
	ports := []int{3000, 3306, 5432, 6379, 8080, 8443, 9000, 9200, 27017}
	return ports[rand.Intn(len(ports))]
}
