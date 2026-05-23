package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	graphql "github.com/hasura/go-graphql-client"
)

func runQuery(ctx context.Context, endpoint, query, variables string, headers headerFlag) int {
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

	data, err := client.ExecRaw(ctx, query, vars)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	printJSON(data)
	return 0
}

func runSubscription(ctx context.Context, endpoint, query, variables string, headers headerFlag) int {
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

	_, err = client.Exec(query, vars, func(data []byte, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, "subscription error:", err)
			return nil
		}
		printJSON(data)
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
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		return 0
	}
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

func printJSON(data []byte) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		os.Stdout.Write(data)
		fmt.Println()
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
