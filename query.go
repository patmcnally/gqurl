package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	graphql "github.com/hasura/go-graphql-client"
)

func runQuery(ctx context.Context, endpoint, query, variables string, headers headerFlag, out io.Writer, compact bool, operationName string) int {
	vars, err := parseVariables(variables)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	parsed := parseHeaders(headers)
	client := graphql.NewClient(endpoint, http.DefaultClient).
		WithRequestModifier(func(req *http.Request) {
			for k, vs := range parsed {
				for _, v := range vs {
					req.Header.Set(k, v)
				}
			}
		})

	var opts []graphql.Option
	if operationName != "" {
		opts = append(opts, graphql.OperationName(operationName))
	}

	data, err := client.ExecRaw(ctx, query, vars, opts...)
	// Print data first — partial results arrive alongside errors.
	if len(data) > 0 {
		printJSON(out, data, compact)
	}
	if err != nil {
		printGraphQLError(err)
		return 1
	}
	return 0
}

func runSubscription(ctx context.Context, endpoint, query, variables string, headers headerFlag, out io.Writer, compact bool, operationName string) int {
	vars, err := parseVariables(variables)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	errCh := make(chan error, 1)

	client := graphql.NewSubscriptionClient(endpoint).
		WithWebSocketOptions(graphql.WebsocketOptions{
			HTTPHeader: parseHeaders(headers),
		}).
		OnError(func(sc *graphql.SubscriptionClient, err error) error {
			return err
		})

	var opts []graphql.Option
	if operationName != "" {
		opts = append(opts, graphql.OperationName(operationName))
	}

	_, err = client.Exec(query, vars, func(data []byte, err error) error {
		if err != nil {
			printGraphQLError(err)
			return nil
		}
		printJSON(out, data, compact)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	go func() { errCh <- client.RunWithContext(ctx) }()

	select {
	case <-ctx.Done():
		_ = client.Close()
		return 0
	case err = <-errCh:
		if err != nil {
			printGraphQLError(err)
			return 1
		}
		return 0
	}
}

// printGraphQLError writes a human-readable error to stderr. It understands
// graphql.Errors (structured GraphQL errors) and graphql.NetworkError
// (non-2xx HTTP responses), falling back to a plain message for anything else.
func printGraphQLError(err error) {
	if gqlErrs, ok := errors.AsType[graphql.Errors](err); ok {
		for _, e := range gqlErrs {
			loc := ""
			if len(e.Locations) > 0 {
				loc = fmt.Sprintf(" (line %d, col %d)", e.Locations[0].Line, e.Locations[0].Column)
			}
			fmt.Fprintf(os.Stderr, "error: %s%s\n", e.Message, loc)
		}
		return
	}

	if netErr, ok := errors.AsType[graphql.NetworkError](err); ok {
		fmt.Fprintf(os.Stderr, "error: HTTP %d: %s\n", netErr.StatusCode(), netErr.Body())
		return
	}

	fmt.Fprintln(os.Stderr, "error:", err)
}

func parseVariables(s string) (map[string]any, error) {
	if s == "" || s == "{}" {
		return nil, nil
	}
	var vars map[string]any
	if err := json.Unmarshal([]byte(s), &vars); err != nil {
		return nil, fmt.Errorf("invalid variables JSON: %w", err)
	}
	return vars, nil
}

func parseHeaders(headers headerFlag) http.Header {
	h := make(http.Header)
	for _, raw := range headers {
		key, val, ok := strings.Cut(raw, ":")
		if !ok {
			continue
		}
		h.Set(strings.TrimSpace(key), strings.TrimSpace(val))
	}
	return h
}

func printJSON(w io.Writer, data []byte, compact bool) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		w.Write(data)
		fmt.Fprintln(w)
		return
	}
	enc := json.NewEncoder(w)
	if !compact {
		enc.SetIndent("", "  ")
	}
	_ = enc.Encode(v)
}
