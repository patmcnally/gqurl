package main

import (
	"context"
	"fmt"
)

func runQuery(ctx context.Context, endpoint, query, variables string, headers headerFlag) int {
	// TODO: implement with hasura/go-graphql-client
	fmt.Printf("query %s %q vars=%s headers=%v\n", endpoint, query, variables, headers)
	return 0
}

func runSubscription(ctx context.Context, endpoint, query, variables string, headers headerFlag) int {
	// TODO: implement with hasura/go-graphql-client (websocket transport)
	fmt.Printf("subscription %s %q vars=%s headers=%v\n", endpoint, query, variables, headers)
	<-ctx.Done()
	return 0
}
