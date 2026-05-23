package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"maragu.dev/is"
)

func TestLoadHeaderFile(t *testing.T) {
	t.Run("loads key-value pairs from a JSON file", func(t *testing.T) {
		path := writeJSON(t, map[string]string{
			"Authorization": "Bearer token",
			"X-Custom":      "value",
		})

		headers, err := loadHeaderFile(path)
		is.NotError(t, err)
		is.Equal(t, 2, len(headers))

		parsed := parseHeaders(headers)
		is.Equal(t, "Bearer token", parsed.Get("Authorization"))
		is.Equal(t, "value", parsed.Get("X-Custom"))
	})

	t.Run("expands $VAR placeholders from the environment", func(t *testing.T) {
		t.Setenv("GQURL_TEST_TOKEN", "secret-abc")

		path := writeJSON(t, map[string]string{
			"Authorization": "Bearer $GQURL_TEST_TOKEN",
		})

		headers, err := loadHeaderFile(path)
		is.NotError(t, err)

		parsed := parseHeaders(headers)
		is.Equal(t, "Bearer secret-abc", parsed.Get("Authorization"))
	})

	t.Run("expands ${VAR} placeholders from the environment", func(t *testing.T) {
		t.Setenv("GQURL_TEST_TOKEN", "secret-xyz")

		path := writeJSON(t, map[string]string{
			"Authorization": "Bearer ${GQURL_TEST_TOKEN}",
		})

		headers, err := loadHeaderFile(path)
		is.NotError(t, err)

		parsed := parseHeaders(headers)
		is.Equal(t, "Bearer secret-xyz", parsed.Get("Authorization"))
	})

	t.Run("unset variable expands to empty string", func(t *testing.T) {
		t.Setenv("GQURL_UNSET_VAR", "")

		path := writeJSON(t, map[string]string{
			"Authorization": "Bearer $GQURL_UNSET_VAR",
		})

		headers, err := loadHeaderFile(path)
		is.NotError(t, err)

		// The expanded value is "Bearer " but parseHeaders trims whitespace,
		// so the final header value is "Bearer".
		parsed := parseHeaders(headers)
		is.Equal(t, "Bearer", parsed.Get("Authorization"))
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		os.WriteFile(path, []byte(`{not valid json}`), 0600)

		_, err := loadHeaderFile(path)
		is.True(t, err != nil)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := loadHeaderFile("/tmp/gqurl-does-not-exist.json")
		is.True(t, err != nil)
	})
}

func writeJSON(t *testing.T, v map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "headers.json")
	b, err := json.Marshal(v)
	is.NotError(t, err)
	is.NotError(t, os.WriteFile(path, b, 0600))
	return path
}
