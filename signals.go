package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseSignal parses the given string as a signal. If the signal is not found,
// an error is returned.
func ParseSignal(s string) (os.Signal, error) {
	sig, ok := SignalLookup[strings.ToUpper(s)]
	if !ok {
		valid := make([]string, 0, len(SignalLookup))
		for k, _ := range SignalLookup {
			valid = append(valid, k)
		}
		sort.Strings(valid)
		return nil, fmt.Errorf("invalid signal %q - valid signals are %q", sig, valid)
	}
	return sig, nil
}
