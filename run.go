package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

func run() int {
	fs := flag.NewFlagSet("gqurl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gqurl <endpoint> [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "flags:")
		fs.PrintDefaults()
	}

	var (
		query        string
		variables    string
		headers      headerFlag
		subscription bool
	)

	fs.StringVar(&query, "q", "", "GraphQL query or mutation")
	fs.StringVar(&variables, "v", "{}", "JSON variables object")
	fs.Var(&headers, "H", "HTTP header, e.g. 'Authorization: Bearer token' (repeatable)")
	fs.BoolVar(&subscription, "subscription", false, "Run query as a subscription")

	// Pull the endpoint out before flag parsing so flags may appear in any position.
	endpoint, flagArgs := extractEndpoint(os.Args[1:])

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if endpoint == "" {
		fs.Usage()
		return 1
	}

	if query == "" {
		fmt.Fprintln(os.Stderr, "error: -q query is required")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if subscription {
		return runSubscription(ctx, endpoint, query, variables, headers)
	}
	return runQuery(ctx, endpoint, query, variables, headers)
}

// extractEndpoint separates the first http/https URL from the arg list so that
// flags may appear before or after the endpoint.
func extractEndpoint(args []string) (endpoint string, rest []string) {
	for i, a := range args {
		if strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
			return a, append(args[:i:i], args[i+1:]...)
		}
	}
	return "", args
}
