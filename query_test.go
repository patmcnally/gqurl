package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"maragu.dev/is"
)

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
		wantKey string
		wantVal any
	}{
		{name: "empty string returns nil map", input: "", wantNil: true},
		{name: "empty braces returns nil map", input: "{}", wantNil: true},
		{name: "valid JSON object returns map", input: `{"id": "42"}`, wantKey: "id", wantVal: "42"},
		// nested map omitted from table — tested separately below
		{name: "invalid JSON returns error", input: `{bad json}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars, err := parseVariables(tt.input)
			if tt.wantErr {
				is.True(t, err != nil)
				return
			}
			is.NotError(t, err)
			if tt.wantNil {
				is.True(t, vars == nil)
				return
			}
			is.Equal(t, tt.wantVal, vars[tt.wantKey])
		})
	}

	t.Run("nested JSON object is preserved as map", func(t *testing.T) {
		vars, err := parseVariables(`{"filter": {"active": true}}`)
		is.NotError(t, err)
		nested, ok := vars["filter"].(map[string]any)
		is.True(t, ok)
		is.Equal(t, true, nested["active"])
	})
}

func TestParseHeaders(t *testing.T) {
	t.Run("parses a single header", func(t *testing.T) {
		h := parseHeaders(headerFlag{"Authorization: Bearer token"})
		is.Equal(t, "Bearer token", h.Get("Authorization"))
	})

	t.Run("parses multiple headers", func(t *testing.T) {
		h := parseHeaders(headerFlag{
			"Authorization: Bearer token",
			"X-Custom: value",
		})
		is.Equal(t, "Bearer token", h.Get("Authorization"))
		is.Equal(t, "value", h.Get("X-Custom"))
	})

	t.Run("preserves colons in the value", func(t *testing.T) {
		h := parseHeaders(headerFlag{"X-Time: 12:00:00"})
		is.Equal(t, "12:00:00", h.Get("X-Time"))
	})

	t.Run("skips entries without a colon", func(t *testing.T) {
		h := parseHeaders(headerFlag{"no-colon-here"})
		is.Equal(t, 0, len(h))
	})

	t.Run("later entry wins for duplicate key", func(t *testing.T) {
		h := parseHeaders(headerFlag{
			"X-Foo: first",
			"X-Foo: second",
		})
		is.Equal(t, "second", h.Get("X-Foo"))
	})
}

func TestPrintJSON(t *testing.T) {
	t.Run("pretty-prints valid JSON", func(t *testing.T) {
		var buf bytes.Buffer
		printJSON(&buf, []byte(`{"name":"Pat","age":42}`))
		is.True(t, strings.Contains(buf.String(), "\n"))
		is.True(t, strings.Contains(buf.String(), `"name": "Pat"`))
	})

	t.Run("passes through invalid JSON as-is", func(t *testing.T) {
		var buf bytes.Buffer
		printJSON(&buf, []byte(`not json`))
		is.True(t, strings.Contains(buf.String(), "not json"))
	})
}

func TestRunQuery(t *testing.T) {
	t.Run("returns 0 and prints data on success", func(t *testing.T) {
		srv := gqlServer(t, func(w http.ResponseWriter, _ *http.Request) {
			gqlRespond(w, map[string]any{"hello": "world"}, nil)
		})

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "{ hello }", "{}", nil, &buf)
		is.Equal(t, 0, code)
		is.True(t, strings.Contains(buf.String(), `"hello": "world"`))
	})

	t.Run("returns 1 and prints nothing on GraphQL error", func(t *testing.T) {
		srv := gqlServer(t, func(w http.ResponseWriter, _ *http.Request) {
			gqlRespond(w, nil, []map[string]any{{"message": "field not found"}})
		})

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "{ bogus }", "{}", nil, &buf)
		is.Equal(t, 1, code)
		is.Equal(t, "", buf.String())
	})

	t.Run("prints partial data alongside errors", func(t *testing.T) {
		srv := gqlServer(t, func(w http.ResponseWriter, _ *http.Request) {
			gqlRespond(w,
				map[string]any{"partial": "result"},
				[]map[string]any{{"message": "some field failed"}},
			)
		})

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "{ partial bogus }", "{}", nil, &buf)
		is.Equal(t, 1, code)
		is.True(t, strings.Contains(buf.String(), `"partial": "result"`))
	})

	t.Run("sends variables to the server", func(t *testing.T) {
		var gotVars map[string]any
		srv := gqlServer(t, func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Variables map[string]any `json:"variables"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			gotVars = body.Variables
			gqlRespond(w, map[string]any{"ok": true}, nil)
		})

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "query Q($id: ID!) { node(id: $id) { id } }", `{"id":"99"}`, nil, &buf)
		is.Equal(t, 0, code)
		is.Equal(t, "99", gotVars["id"])
	})

	t.Run("sends custom headers to the server", func(t *testing.T) {
		var gotAuth string
		srv := gqlServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gqlRespond(w, map[string]any{"ok": true}, nil)
		})

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "{ ok }", "{}", headerFlag{"Authorization: Bearer tok"}, &buf)
		is.Equal(t, 0, code)
		is.Equal(t, "Bearer tok", gotAuth)
	})

	t.Run("returns 1 on non-2xx HTTP response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		var buf bytes.Buffer
		code := runQuery(t.Context(), srv.URL, "{ me { id } }", "{}", nil, &buf)
		is.Equal(t, 1, code)
	})
}

// gqlServer starts a test HTTP server and registers cleanup.
func gqlServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// gqlRespond writes a minimal GraphQL response. Pass nil data or nil errors to omit them.
func gqlRespond(w http.ResponseWriter, data any, errs []map[string]any) {
	body := map[string]any{}
	if data != nil {
		body["data"] = data
	}
	if len(errs) > 0 {
		body["errors"] = errs
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}
