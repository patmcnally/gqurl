package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// headerFlag implements flag.Value for repeated -H flags.
type headerFlag []string

func (h *headerFlag) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlag) Set(v string) error {
	*h = append(*h, v)
	return nil
}

// loadHeaderFile reads a JSON object of {"Header-Name": "value"} pairs and
// returns them as a headerFlag. The file may live at any path; use @- for stdin.
func loadHeaderFile(path string) (headerFlag, error) {
	var src []byte
	var err error
	if path == "-" {
		src, err = os.ReadFile("/dev/stdin")
	} else {
		src, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading header file: %w", err)
	}

	var obj map[string]string
	if err := json.Unmarshal(src, &obj); err != nil {
		return nil, fmt.Errorf("parsing header file (expected JSON object): %w", err)
	}

	out := make(headerFlag, 0, len(obj))
	for k, v := range obj {
		out = append(out, k+": "+v)
	}
	return out, nil
}
