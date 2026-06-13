# HANDOFF — gqurl

## Status
**Idle** — repo cloned to Forge sandbox on 2026-06-13. No active task.

## What it is
A `curl`-like CLI for GraphQL. Send queries, mutations, and subscriptions from the terminal. Supports variables, custom headers, header files with env var expansion, file/stdin input, and WebSocket subscriptions. Output is pretty-printed JSON from the `data` field; errors go to stderr with location info.

## Sandbox
- **Path:** `/home/pat/sandbox/gqurl` on `forge.patmcnally.com`
- **Repo:** `github.com/patmcnally/gqurl`

## Stack
Go. Single binary. No external runtime dependencies.

## Quick start
```bash
make install          # build and install to $GOPATH/bin
make test             # run tests
go run . <endpoint> -q '{ hero { name } }'
```

## For the agent picking this up
1. Read this file
2. Read the README for full usage and examples
3. Check `kbase.patmcnally.com` (`kiwi_search "gqurl"`) for any prior context
4. Ask the operator what they want to work on

## Last operator note
*(no notes yet — leave instructions here before handing off to an agent)*
