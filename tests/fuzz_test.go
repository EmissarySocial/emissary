package tests

import (
	"net/url"
	"testing"
	"text/template"
)

// FuzzFormPosts posts arbitrary text into every field of every pinned step, in every
// template group, and fails on any panic.  A rejected value is fine; a crash is not.
func FuzzFormPosts(f *testing.F) {

	cases := make([]postCase, 0)

	for _, group := range templateGroups {
		if group.missingLocation() == "" {
			cases = append(cases, collectGroup(loadGroup(f, group)).Cases...)
		}
	}

	if len(cases) == 0 {
		f.Skip("no template groups are available")
	}

	// Seed with the interesting values from the pinning scenarios
	for _, seed := range []string{"", "Sample", "true", "7", "3.5", "2026-10-02", "#336699", "0123456789abcdef01234567", "https://example.com/", "javascript:alert(1)", "<script>alert(1)</script>", "-99999999999999999999", "NaN", "\u202e\x00", "a.b.c", "0", "-1"} {
		f.Add(uint16(0), seed, false)
		f.Add(uint16(len(cases)/2), seed, true)
	}

	f.Fuzz(func(t *testing.T, index uint16, value string, repeat bool) {

		postCase := cases[int(index)%len(cases)]
		values := url.Values{}

		for _, field := range postCase.Fields {
			values[field.Path] = []string{value}
			if repeat {
				values[field.Path] = append(values[field.Path], value)
			}
		}

		if output := postCase.run(values); output.Panic != "" {
			t.Fatalf("%s panicked on %q: %s", postCase.Name, value, output.Panic)
		}
	})
}

// FuzzSetData sends arbitrary text through every set-data step, as a query value, a posted
// value, and the rendered text of every "values" template, and fails on any panic
func FuzzSetData(f *testing.F) {

	uses := make([]setDataUse, 0)

	for _, group := range templateGroups {
		if group.missingLocation() == "" {
			uses = append(uses, collectGroup(loadGroup(f, group)).SetData...)
		}
	}

	if len(uses) == 0 {
		f.Skip("no template groups are available")
	}

	for _, seed := range []string{"", "Sample", "true", "0", "-1", "3.5", "NONE", "{{.Label}}", "<script>alert(1)</script>", "javascript:alert(1)", "\u202e\x00", "a,b,c"} {
		f.Add(uint16(0), seed, false)
		f.Add(uint16(len(uses)/2), seed, true)
	}

	f.Fuzz(func(t *testing.T, index uint16, value string, post bool) {

		use := uses[int(index)%len(uses)]
		method := "GET"
		if post {
			method = "POST"
		}

		scenario := setDataScenario{
			Name:     "fuzz",
			Request:  eachField(func(postField) []string { return []string{value} }),
			Rendered: func(postField, *template.Template) string { return value },
		}

		postCase := use.postCase(method, scenario)

		if output := postCase.run(scenario.Request(postCase.Fields)); output.Panic != "" {
			t.Fatalf("%s %s panicked on %q: %s", method, use.Name, value, output.Panic)
		}
	})
}
