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
		query         string
		variables     string
		headers       headerFlag
		headerFile    string
		outputFile    string
		operationName string
		compact       bool
		introspect    bool
		subscription  bool
	)

	fs.StringVar(&query, "q", "", "GraphQL query or mutation (use @- for stdin, @file for a file)")
	fs.StringVar(&variables, "v", "{}", "JSON variables object")
	fs.Var(&headers, "H", "HTTP header, e.g. 'Authorization: Bearer token' (repeatable)")
	fs.StringVar(&headerFile, "header-file", "", "JSON file of {\"Header\": \"value\"} pairs (values support $ENV expansion)")
	fs.StringVar(&outputFile, "o", "", "Write JSON output to file instead of stdout")
	fs.StringVar(&operationName, "n", "", "Operation name for multi-operation documents")
	fs.BoolVar(&compact, "c", false, "Compact output (no pretty-printing)")
	fs.BoolVar(&introspect, "introspect", false, "Fetch and print the schema via introspection")
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

	if !introspect && query == "" {
		fmt.Fprintln(os.Stderr, "error: -q query is required")
		return 1
	}

	// Merge headers lowest-to-highest precedence: defaults < file < -H flags.
	defaults := headerFlag{
		"apollographql-client-name: gqurl",
		"apollographql-client-version: " + version,
	}
	headers = append(defaults, headers...)

	if headerFile != "" {
		fileHeaders, err := loadHeaderFile(headerFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		headers = append(fileHeaders, headers...)
	}

	out, err := openOutput(outputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer out.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if introspect {
		return runIntrospect(ctx, endpoint, headers, out, compact)
	}

	resolvedQuery, err := resolveQuery(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if subscription {
		return runSubscription(ctx, endpoint, resolvedQuery, variables, headers, out, compact, operationName)
	}
	return runQuery(ctx, endpoint, resolvedQuery, variables, headers, out, compact, operationName)
}

// openOutput returns a write-closer for the given path, or a no-op-close
// wrapper around os.Stdout when path is empty.
func openOutput(path string) (interface {
	io.Writer
	Close() error
}, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("opening output file: %w", err)
	}
	return f, nil
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

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
