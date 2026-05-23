package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "pass -q @- to read the query from stdin, or -q @file to read from a file")
	}

	var (
		query      string
		variables  string
		headers    headerFlag
		headerFile string
		subscription bool
	)

	fs.StringVar(&query, "q", "", "GraphQL query or mutation (use @- for stdin, @file for a file)")
	fs.StringVar(&variables, "v", "{}", "JSON variables object")
	fs.Var(&headers, "H", "HTTP header, e.g. 'Authorization: Bearer token' (repeatable)")
	fs.StringVar(&headerFile, "header-file", "", "JSON file of {\"Header\": \"value\"} pairs")
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

	// Merge headers: file headers first so that -H flags take precedence.
	if headerFile != "" {
		fileHeaders, err := loadHeaderFile(headerFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		headers = append(fileHeaders, headers...)
	}

	resolvedQuery, err := resolveQuery(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if subscription {
		return runSubscription(ctx, endpoint, resolvedQuery, variables, headers)
	}
	return runQuery(ctx, endpoint, resolvedQuery, variables, headers)
}

// resolveQuery returns the query string, reading from stdin or a file when the
// value begins with @.
func resolveQuery(q string) (string, error) {
	if !strings.HasPrefix(q, "@") {
		return q, nil
	}
	src := q[1:]
	var r io.Reader
	if src == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(src)
		if err != nil {
			return "", fmt.Errorf("opening query file: %w", err)
		}
		defer f.Close()
		r = f
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading query: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
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
