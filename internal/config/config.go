// Package config defines shared configuration structures for pitchpotd.
package config

// ServerConfig is the top-level daemon configuration (loaded from TOML).
type ServerConfig struct {
	Listen   string      `toml:"listen"`    // e.g. "127.0.0.1:9999"
	Sensor   string      `toml:"sensor"`    // sensor/host label in events
	Corpus   CorpusCfg   `toml:"corpus"`
	Log      LogCfg      `toml:"log"`
	HTTP     HTTPCfg     `toml:"http"`
}

// CorpusCfg points to a generated corpus pack directory.
type CorpusCfg struct {
	PackDir string `toml:"pack_dir"`
}

// LogCfg controls log output paths.
type LogCfg struct {
	JSONLPath  string `toml:"jsonl_path"`   // full event log
	BanLogPath string `toml:"banlog_path"`  // compact ban-log for fail2ban/crowdsec
}

// HTTPCfg controls HTTP tarpit behaviour.
type HTTPCfg struct {
	// TarpitProfile selects the default drip profile: "annoy", "trap", "endless".
	TarpitProfile string `toml:"tarpit_profile"`

	// ChunkSize is the number of bytes sent per drip tick.
	ChunkSize int `toml:"chunk_size"`

	// TickMS is the delay between drip ticks in milliseconds.
	TickMS int `toml:"tick_ms"`

	// MaxDurationSec caps total response duration (0 = unlimited).
	MaxDurationSec int `toml:"max_duration_sec"`

	// MaxBodyExcerptBytes is how many bytes of POST body to capture in the log.
	MaxBodyExcerptBytes int `toml:"max_body_excerpt_bytes"`
}
