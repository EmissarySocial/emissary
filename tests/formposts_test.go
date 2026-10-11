package tests

import (
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// update rewrites every golden file from the current behavior: go test ./tests -update
var update = flag.Bool("update", false, "rewrite the golden files in testdata from current behavior")

// goldenFile is the pinned behavior of one template group
type goldenFile struct {
	Cases     map[string]goldenCase `json:"cases"`
	Inventory map[string]string     `json:"inventory"`
}

// goldenCase is the pinned behavior of one step under every scenario
type goldenCase struct {
	Fields    []string          `json:"fields"`
	Scenarios map[string]result `json:"scenarios"`
}

// TestFormPosts pins what every form POST in every template does to the object it edits,
// against the golden file for its template group
func TestFormPosts(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			actual := pinGroup(loaded)
			filename := filepath.Join("testdata", group.Name+".golden.json")

			if *update {

				// RULE: never pin behavior that changes from one run to the next
				require.Equal(t, actual, pinGroup(loaded), "behavior is not deterministic, so it cannot be pinned")

				encoded, err := json.MarshalIndent(actual, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filename, append(encoded, '\n'), 0o600))
				return
			}

			encoded, err := os.ReadFile(filename) // #nosec G304 -- the name is built from a fixed group list
			require.NoError(t, err, "missing golden file; run go test ./tests -update")

			expected := goldenFile{}
			require.NoError(t, json.Unmarshal(encoded, &expected))

			// Every step that is pinned or skipped is still where the golden file says it is
			require.Equal(t, sortedKeys(expected.Cases), sortedKeys(actual.Cases), "the set of form-posting steps changed")
			require.Equal(t, expected.Inventory, actual.Inventory, "the set of unpinned POST-reading steps changed")

			// Then each step behaves exactly as pinned
			for _, name := range sortedKeys(expected.Cases) {
				t.Run(name, func(t *testing.T) {
					require.Equal(t, expected.Cases[name], actual.Cases[name])
				})
			}
		})
	}
}

// pinScenario runs one scenario.  A step that ranges over Go maps runs once for every
// order in mapOrders; see AGENTS.md, "Some results depend on map order".
func pinScenario(postCase postCase, values url.Values) result {

	if postCase.Orders == nil {
		return postCase.run(values)
	}

	var first result
	var pinned result
	varied := false

	for index, orders := range mapOrders(postCase.Orders(values)) {

		next := postCase.runOrdered(values, orders)

		if index == 0 {
			first, pinned = next, next
			continue
		}

		varied = varied || !reflect.DeepEqual(first, next)
		pinned.Error = sameOrVaries(pinned.Error, next.Error)
		pinned.Root = sameOrVaries(pinned.Root, next.Root)
		pinned.Panic = sameOrVaries(pinned.Panic, next.Panic)
	}

	if !varied {
		return first
	}

	// Keep only the error text that every order agreed on
	return result{
		Error:          pinned.Error,
		Root:           pinned.Root,
		Panic:          pinned.Panic,
		OrderDependent: true,
	}
}

// mapOrders returns every combination of orders to visit a step's maps in: each map's keys
// in every rotation, forwards and backwards, so that every key is visited first once
func mapOrders(groups [][]string) [][][]string {

	result := [][][]string{{}}

	for _, keys := range groups {

		next := make([][][]string, 0)

		for _, prefix := range result {
			for _, order := range rotations(keys) {
				combination := append(append([][]string{}, prefix...), order)
				next = append(next, combination)
			}
		}

		result = next
	}

	return result
}

// rotations returns every rotation of a list, forwards and backwards, without duplicates
func rotations(keys []string) [][]string {

	if len(keys) == 0 {
		return [][]string{{}}
	}

	result := make([][]string, 0, 2*len(keys))
	seen := map[string]bool{}

	reversed := make([]string, len(keys))
	for index, key := range keys {
		reversed[len(keys)-1-index] = key
	}

	for _, list := range [][]string{keys, reversed} {
		for start := range list {
			rotation := append(append([]string{}, list[start:]...), list[:start]...)
			if signature := strings.Join(rotation, "\x00"); !seen[signature] {
				seen[signature] = true
				result = append(result, rotation)
			}
		}
	}

	return result
}

// sameOrVaries keeps a value only while every run agrees on it
func sameOrVaries(pinned string, next string) string {
	if pinned == next {
		return pinned
	}
	return "varies with map order"
}

// pinGroup runs every scenario against every postCase in a group
func pinGroup(group *loadedGroup) goldenFile {

	collected := collectGroup(group)

	pinnedGroup := goldenFile{
		Cases:     make(map[string]goldenCase, len(collected.Cases)),
		Inventory: collected.Inventory,
	}

	for _, postCase := range collected.Cases {

		pinned := goldenCase{
			Fields:    postCase.sortedFields(),
			Scenarios: make(map[string]result, len(scenarios)),
		}

		for _, scenario := range scenarios {
			pinned.Scenarios[scenario.Name] = pinScenario(postCase, scenario.Values(postCase.Fields))
		}

		pinnedGroup.Cases[postCase.Name] = pinned
	}

	return pinnedGroup
}
