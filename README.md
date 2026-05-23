# gqurl

A `curl`-like CLI for GraphQL — send queries, mutations, and subscriptions from the terminal.

```
gqurl <endpoint> -q '{ hero { name } }'
```

---

## Install

```bash
go install github.com/patmcnally/gqurl@latest
```

Or build from source:

```bash
git clone https://github.com/patmcnally/gqurl
cd gqurl
make install
```

---

## Usage

```
gqurl <endpoint> [flags]

  -q string         GraphQL query or mutation (required)
  -v string         JSON variables object (default: {})
  -H value          HTTP header — repeatable
  -header-file      JSON file of {"Header": "value"} pairs
  -o string         Write output to file instead of stdout
  -subscription     Run query as a WebSocket subscription
```

Flags may appear before or after the endpoint.

---

## Examples

**Query**

```bash
gqurl https://api.example.com/graphql \
  -q '{ me { id name email } }'
```

**Query with variables**

```bash
gqurl https://api.example.com/graphql \
  -q 'query User($id: ID!) { user(id: $id) { name } }' \
  -v '{"id": "42"}'
```

**Mutation**

```bash
gqurl https://api.example.com/graphql \
  -q 'mutation CreatePost($title: String!) { createPost(title: $title) { id } }' \
  -v '{"title": "Hello world"}'
```

**Pass a header**

```bash
gqurl https://api.example.com/graphql \
  -H 'Authorization: Bearer mytoken' \
  -q '{ me { id } }'
```

**Read query from stdin** (`@-`) **or a file** (`@path`)

```bash
# stdin
cat query.graphql | gqurl https://api.example.com/graphql -q @-

# file
gqurl https://api.example.com/graphql -q @query.graphql
```

**Write output to a file**

```bash
gqurl https://api.example.com/graphql \
  -q '{ products { id name price } }' \
  -o products.json
```

**Subscription** (streams one JSON object per event, exits on Ctrl-C)

```bash
gqurl https://api.example.com/graphql \
  -q 'subscription { messageAdded { id body } }' \
  --subscription
```

---

## Auth via header file

Keep secrets out of shell history by loading headers from a JSON file:

```json
{
  "Authorization": "Bearer $API_TOKEN",
  "X-Tenant-ID": "${TENANT_ID}"
}
```

Values support `$VAR` / `${VAR}` expansion from the environment at runtime.

```bash
export API_TOKEN=$(cat ~/.secrets/api-token)

gqurl https://api.example.com/graphql \
  -header-file ~/.config/gqurl/headers.json \
  -q '{ me { id } }'
```

Individual `-H` flags take precedence over the file for the same header name.

---

## Output

Responses are pretty-printed JSON from the GraphQL `data` field:

```json
{
  "me": {
    "id": "42",
    "name": "Pat"
  }
}
```

GraphQL errors print to stderr with location info; the exit code is 1:

```
error: Cannot query field "bogus" on type "Query". (line 1, col 3)
```

Partial results (data + errors) print data to stdout and errors to stderr.

---

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | GraphQL error, network error, or bad arguments |
