// pitchpot-configurator reads an nginx site config, determines which honeypot
// locations are absent, and generates an nginx include file that proxies those
// locations to pitchpotd.
//
// Usage:
//
//	pitchpot-configurator generate \
//	  --nginx-conf /etc/nginx/sites-enabled/mysite.conf \
//	  --tarpit-addr 127.0.0.1:9999 \
//	  --output /etc/pitchpot/honeypot.conf
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nikita-popov/pitchpot/internal/nginx"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: pitchpot-configurator <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  generate   Generate nginx honeypot location include")
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	nginxConf := fs.String("nginx-conf", "", "Path to nginx site config file (required)")
	tarpitAddr := fs.String("tarpit-addr", "127.0.0.1:9999", "Address of pitchpotd (host:port)")
	output := fs.String("output", "", "Output path for generated include file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *nginxConf == "" {
		fmt.Fprintln(os.Stderr, "error: --nginx-conf is required")
		os.Exit(1)
	}

	locations, err := nginx.ParseLocations(*nginxConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	include := nginx.GenerateInclude(locations, *tarpitAddr)

	if *output == "" {
		fmt.Print(include)
		return
	}

	if err := os.WriteFile(*output, []byte(include), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write output: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Written to %s\n", *output)
}
