package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"maragu.dev/is"
)

func TestExtractEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantEndpoint string
		wantRest     []string
	}{
		{
			name:         "http endpoint before flags",
			args:         []string{"http://example.com", "-q", "{}"},
			wantEndpoint: "http://example.com",
			wantRest:     []string{"-q", "{}"},
		},
		{
			name:         "https endpoint after flags",
			args:         []string{"-q", "{}", "https://example.com"},
			wantEndpoint: "https://example.com",
			wantRest:     []string{"-q", "{}"},
		},
		{
			name:         "endpoint between flags",
			args:         []string{"-H", "X: y", "https://example.com", "-q", "{}"},
			wantEndpoint: "https://example.com",
			wantRest:     []string{"-H", "X: y", "-q", "{}"},
		},
		{
			name:         "no endpoint returns empty and passes args through",
			args:         []string{"-q", "{}"},
			wantEndpoint: "",
			wantRest:     []string{"-q", "{}"},
		},
		{
			name:         "empty args",
			args:         []string{},
			wantEndpoint: "",
			wantRest:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, rest := extractEndpoint(tt.args)
			is.Equal(t, tt.wantEndpoint, endpoint)
			is.EqualSlice(t, tt.wantRest, rest)
		})
	}
}

func TestDefaultApolloHeaders(t *testing.T) {
	t.Run("sends apollographql-client-name and version by default", func(t *testing.T) {
		var gotName, gotVer string
		srv := gqlServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotName = r.Header.Get("Apollographql-Client-Name")
			gotVer = r.Header.Get("Apollographql-Client-Version")
			gqlRespond(w, map[string]any{"ok": true}, nil)
		})

		defaults := headerFlag{
			"apollographql-client-name: gqurl",
			"apollographql-client-version: " + version,
		}
		var buf bytes.Buffer
		runQuery(t.Context(), srv.URL, "{ ok }", "{}", defaults, &buf, false, "")

		is.Equal(t, "gqurl", gotName)
		is.Equal(t, version, gotVer)
	})

	t.Run("-H flag overrides the default client name", func(t *testing.T) {
		var gotName string
		srv := gqlServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotName = r.Header.Get("Apollographql-Client-Name")
			gqlRespond(w, map[string]any{"ok": true}, nil)
		})

		// Simulate run()'s merge: defaults first, then user -H (which wins).
		merged := append(
			headerFlag{"apollographql-client-name: gqurl"},
			headerFlag{"apollographql-client-name: my-app"}...,
		)
		var buf bytes.Buffer
		runQuery(t.Context(), srv.URL, "{ ok }", "{}", merged, &buf, false, "")

		is.Equal(t, "my-app", gotName)
	})
}

func TestResolveQuery(t *testing.T) {
	t.Run("returns inline query unchanged", func(t *testing.T) {
		q, err := resolveQuery("{ hero { name } }")
		is.NotError(t, err)
		is.Equal(t, "{ hero { name } }", q)
	})

	t.Run("reads query from a file with @path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "q.graphql")
		os.WriteFile(path, []byte("  { hero { name } }\n"), 0600)

		q, err := resolveQuery("@" + path)
		is.NotError(t, err)
		is.Equal(t, "{ hero { name } }", q) // trimmed
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := resolveQuery("@/tmp/gqurl-no-such-file.graphql")
		is.True(t, err != nil)
	})
}
