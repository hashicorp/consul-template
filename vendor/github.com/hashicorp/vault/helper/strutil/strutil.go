package strutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// StrListContains looks for a string in a list of strings.
func StrListContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

// StrListSubset checks if a given list is a subset
// of another set
func StrListSubset(super, sub []string) bool {
	for _, item := range sub {
		if !StrListContains(super, item) {
			return false
		}
	}
	return true
}

// Parses a comma separated list of strings into a slice of strings.
// The return slice will be sorted and will not contain duplicate or
// empty items. The values will be converted to lower case.
func ParseDedupAndSortStrings(input string, sep string) []string {
	input = strings.TrimSpace(input)
	parsed := []string{}
	if input == "" {
		// Don't return nil
		return parsed
	}
	return RemoveDuplicates(strings.Split(input, sep))
}

// Parses a comma separated list of `<key>=<value>` tuples into a
// map[string]string.
func ParseKeyValues(input string, out map[string]string, sep string) error {
	if out == nil {
		return fmt.Errorf("'out is nil")
	}

	keyValues := ParseDedupAndSortStrings(input, sep)
	if len(keyValues) == 0 {
		return nil
	}

	for _, keyValue := range keyValues {
		shards := strings.Split(keyValue, "=")
		key := strings.TrimSpace(shards[0])
		value := strings.TrimSpace(shards[1])
		if key == "" || value == "" {
			return fmt.Errorf("invalid <key,value> pair: key:'%s' value:'%s'", key, value)
		}
		out[key] = value
	}
	return nil
}

// Parses arbitrary <key,value> tuples. The input can be one of
// the following:
// * JSON string
// * Base64 encoded JSON string
// * Comma separated list of `<key>=<value>` pairs
// * Base64 encoded string containing comma separated list of
//   `<key>=<value>` pairs
//
// Input will be parsed into the output paramater, which should
// be a non-nil map[string]string.
func ParseArbitraryKeyValues(input string, out map[string]string, sep string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	if out == nil {
		return fmt.Errorf("'out' is nil")
	}

	// Try to base64 decode the input. If successful, consider the decoded
	// value as input.
	inputBytes, err := base64.StdEncoding.DecodeString(input)
	if err == nil {
		input = string(inputBytes)
	}

	// Try to JSON unmarshal the input. If successful, consider that the
	// metadata was supplied as JSON input.
	err = json.Unmarshal([]byte(input), &out)
	if err != nil {
		// If JSON unmarshalling fails, consider that the input was
		// supplied as a comma separated string of 'key=value' pairs.
		if err = ParseKeyValues(input, out, sep); err != nil {
			return fmt.Errorf("failed to parse the input: %v", err)
		}
	}

	// Validate the parsed input
	for key, value := range out {
		if key != "" && value == "" {
			return fmt.Errorf("invalid value for key '%s'", key)
		}
	}

	return nil
}

// Parses a `sep`-separated list of strings into a
// []string.
//
// The output will always be a valid slice but may be of length zero.
func ParseStringSlice(input string, sep string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	splitStr := strings.Split(input, sep)
	ret := make([]string, len(splitStr))
	for i, val := range splitStr {
		ret[i] = val
	}

	return ret
}

// Parses arbitrary string slice. The input can be one of
// the following:
// * JSON string
// * Base64 encoded JSON string
// * `sep` separated list of values
// * Base64-encoded string containting a `sep` separated list of values
//
// Note that the separator is ignored if the input is found to already be in a
// structured format (e.g., JSON)
//
// The output will always be a valid slice but may be of length zero.
func ParseArbitraryStringSlice(input string, sep string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	// Try to base64 decode the input. If successful, consider the decoded
	// value as input.
	inputBytes, err := base64.StdEncoding.DecodeString(input)
	if err == nil {
		input = string(inputBytes)
	}

	ret := []string{}

	// Try to JSON unmarshal the input. If successful, consider that the
	// metadata was supplied as JSON input.
	err = json.Unmarshal([]byte(input), &ret)
	if err != nil {
		// If JSON unmarshalling fails, consider that the input was
		// supplied as a separated string of values.
		return ParseStringSlice(input, sep)
	}

	if ret == nil {
		return []string{}
	}

	return ret
}

// Removes duplicate and empty elements from a slice of strings.
// This also converts the items in the slice to lower case and
// returns a sorted slice.
func RemoveDuplicates(items []string) []string {
	itemsMap := map[string]bool{}
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		itemsMap[item] = true
	}
	items = []string{}
	for item, _ := range itemsMap {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

// EquivalentSlices checks whether the given string sets are equivalent, as in,
// they contain the same values.
func EquivalentSlices(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	// First we'll build maps to ensure unique values
	mapA := map[string]bool{}
	mapB := map[string]bool{}
	for _, keyA := range a {
		mapA[keyA] = true
	}
	for _, keyB := range b {
		mapB[keyB] = true
	}

	// Now we'll build our checking slices
	var sortedA, sortedB []string
	for keyA, _ := range mapA {
		sortedA = append(sortedA, keyA)
	}
	for keyB, _ := range mapB {
		sortedB = append(sortedB, keyB)
	}
	sort.Strings(sortedA)
	sort.Strings(sortedB)

	// Finally, compare
	if len(sortedA) != len(sortedB) {
		return false
	}

	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}

	return true
}

// StrListDelete removes the first occurance of the given item from the slice
// of strings if the item exists.
func StrListDelete(s []string, d string) []string {
	if s == nil {
		return s
	}

	for index, element := range s {
		if element == d {
			return append(s[:index], s[index+1:]...)
		}
	}

	return s
}
