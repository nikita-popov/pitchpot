# Pitchpot

> ⚠️ **Work in progress.** The project is under active development. APIs, config formats, and corpus schema may change without notice.

A deception honeypot framework for nginx environments.
Slowly poisons automated scanners, crawlers, and LLM-based agents
with plausible-looking but completely useless content.

## Components

- `pitchpot-configurator` — reads nginx config, generates honeypot location includes and corpus packs
- `pitchpotd` — tarpit server, receives proxied requests, logs events, streams deception responses

## Design Principles

- UNIX/KISS: each component does one thing
- Modular protocol support (HTTP first, others later)
- Universal event log format (JSONL) for all protocols
- Corpus is compiled once; runtime only serves and mutates
- Ollama integration is optional and only used at corpus-generation time

## Quick Start

```sh
# Generate nginx includes and corpus pack for a site
pitchpot-configurator generate \
  --nginx-conf /etc/nginx/sites-enabled/mysite.conf \
  --profile wordpress \
  --output /etc/pitchpot/packs/mysite

# Run the tarpit server
pitchpotd --config /etc/pitchpot/pitchpotd.toml
```

## Log Format

All events are written as JSONL. See `internal/event/event.go` for the schema.
A compact ban-log is also written for fail2ban/CrowdSec.

## Directory Layout

```
cmd/
  pitchpot-configurator/   configurator CLI
  pitchpotd/               tarpit daemon
internal/
  config/                  shared config structs
  corpus/                  corpus pack loader and mutator
  event/                   universal event schema
  logging/                 JSONL + banlog writers
  nginx/                   nginx config parser (location extractor)
  proto/
    http/                  HTTP protocol handler
      profiles/            per-target response profiles
configs/
  pitchpotd.toml.example   example daemon config
examples/
  nginx/                   example nginx include output
```
