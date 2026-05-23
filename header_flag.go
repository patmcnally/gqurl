package main

import "strings"

// headerFlag implements flag.Value for repeated -H flags.
type headerFlag []string

func (h *headerFlag) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlag) Set(v string) error {
	*h = append(*h, v)
	return nil
}
