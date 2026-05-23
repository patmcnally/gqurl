package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
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

	if err := fs.Parse(os.Args[1:]); err != nil {
		return 1
	}

	endpoint := fs.Arg(0)
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
