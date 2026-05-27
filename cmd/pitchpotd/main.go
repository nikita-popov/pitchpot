// pitchpotd is the pitchpot tarpit daemon.
//
// It listens on a local address (typically proxied to by nginx) and for every
// incoming HTTP request it:
//   - Captures all request metadata into a universal Event
//   - Matches the path against a corpus pack
//   - Slowly drip-feeds a plausible but completely bogus response
//   - Writes a full JSONL event log and a compact ban-log
//
// Usage:
//
//	pitchpotd --listen 127.0.0.1:9999 --pack /etc/pitchpot/packs/default
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nikita-popov/pitchpot/internal/corpus"
	"github.com/nikita-popov/pitchpot/internal/logging"
	protohttp "github.com/nikita-popov/pitchpot/internal/proto/http"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:9999", "Address to listen on")
	packDir := flag.String("pack", "", "Path to corpus pack directory (required)")
	sensor := flag.String("sensor", hostname(), "Sensor label written to events")
	jsonlPath := flag.String("jsonl-log", "/var/log/pitchpot/events.jsonl", "Full event log path")
	banlogPath := flag.String("ban-log", "/var/log/pitchpot/ban.log", "Compact ban-log path")
	chunkSize := flag.Int("chunk-size", 64, "Bytes per drip tick")
	tickMS := flag.Int("tick-ms", 2000, "Milliseconds between drip ticks")
	maxDurSec := flag.Int("max-duration", 300, "Max tarpit duration seconds (0=unlimited)")
	flag.Parse()

	if *packDir == "" {
		fmt.Fprintln(os.Stderr, "error: --pack is required")
		os.Exit(1)
	}

	// Load corpus pack.
	pack, err := corpus.Load(*packDir)
	if err != nil {
		log.Fatalf("corpus: %v", err)
	}
	log.Printf("loaded corpus pack %q profile=%s entries=%d",
		*packDir, pack.Manifest.Profile, len(pack.Manifest.Entries))

	// Set up log writers.
	jsonlW, err := logging.NewJSONLWriter(*jsonlPath)
	if err != nil {
		log.Fatalf("jsonl log: %v", err)
	}
	banW, err := logging.NewBanLogWriter(*banlogPath)
	if err != nil {
		log.Fatalf("ban log: %v", err)
	}
	writer := logging.NewMulti(jsonlW, banW)
	defer writer.Close()

	// Build HTTP handler.
	cfg := protohttp.HandlerConfig{
		Sensor:              *sensor,
		ChunkSize:           *chunkSize,
		TickInterval:        time.Duration(*tickMS) * time.Millisecond,
		MaxDuration:         time.Duration(*maxDurSec) * time.Second,
		MaxBodyExcerptBytes: 512,
	}
	handler := protohttp.New(cfg, pack, writer)

	srv := &http.Server{
		Addr:        *listen,
		Handler:     handler,
		ReadTimeout: 10 * time.Second,
		// WriteTimeout intentionally omitted — drip responses are long.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("pitchpotd listening on %s", *listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	srv.Close()
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "pitchpot"
	}
	return h
}
