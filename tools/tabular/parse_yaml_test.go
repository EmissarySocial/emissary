package tabular

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestParseYAML_FlattensAndKeepsOrder(t *testing.T) {

	body := `# Cities
- token: paris
  location:
    longitude: 2.35
    latitude: 48.85
  label: Paris
- token: lyon
  rank: 2
  featured: yes
  published: true
  summary: ~
  tags: [a, b]
`

	table, err := Parse([]byte(body), FormatYAML, Limits{})
	require.NoError(t, err)
	require.Equal(t, FormatYAML, table.Format)
	require.False(t, table.HasProblems())

	require.Equal(t, []Row{
		{Number: 1, Line: 2, Cells: []Cell{
			{"token", "paris"},
			{"location.longitude", 2.35},
			{"location.latitude", 48.85},
			{"label", "Paris"},
		}},
		{Number: 2, Line: 7, Cells: []Cell{
			{"token", "lyon"},
			{"rank", 2},
			{"featured", "yes"},
			{"published", true},
			{"summary", nil},
			{"tags", []any{"a", "b"}},
		}},
	}, table.Rows)
}

func TestParseYAML_DocumentMarker(t *testing.T) {

	table, err := Parse([]byte("---\n- token: a\n"), FormatYAML, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Rows, 1)
}

func TestParseYAML_AliasesAreRefused(t *testing.T) {

	body := "- &first\n  token: a\n- *first\n- token: c\n  copy: *first\n- token: d\n  <<: *first\n"

	table, err := Parse([]byte(body), FormatYAML, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Rows, 1, "only the row that defines the anchor survives")

	require.Len(t, table.Problems, 3)
	for _, problem := range table.Problems {
		require.Equal(t, "YAML aliases are not supported", problem.Message)
	}
}

func TestParseYAML_RowProblems(t *testing.T) {

	test := func(body string, message string) {
		t.Helper()

		table, err := Parse([]byte(body), FormatYAML, Limits{MaxDepth: 2})
		require.NoError(t, err, body)
		require.Empty(t, table.Rows, body)
		require.Len(t, table.Problems, 1, body)
		require.Equal(t, message, table.Problems[0].Message, body)
	}

	test("- just text\n", "Entry must be an object")
	test("- [1, 2]\n", "Entry must be an object")
	test("- ? [a, b]\n  : c\n", "Keys must be plain text")
	test("- a: {b: {c: 1}}\n", "Value is nested more than 2 levels deep")
	test("- a: [[[[[1]]]]]\n", "Value is nested more than 2 levels deep")

	// A scalar that contradicts its own tag cannot be decoded.  The library's wording is its own.
	table, err := Parse([]byte("- a: !!int notanumber\n"), FormatYAML, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Problems, 1)
	require.Contains(t, table.Problems[0].Message, "Value cannot be read: ")
}

func TestParseYAML_FileErrors(t *testing.T) {

	test := func(body string) {
		t.Helper()
		_, err := Parse([]byte(body), FormatYAML, Limits{})
		require.Error(t, err, body)
		require.True(t, derp.IsBadRequest(err), body)
	}

	test("items:\n  - token: a\n")
	test("just text\n")
	test("- token: a\n---\n- token: b\n")
	test("- token: [unclosed\n")
	test("\t- bad indent: x\n  - y")
}

func TestParseYAML_NothingButComments(t *testing.T) {

	for _, body := range []string{"# nothing here\n", "---\n", "--- # empty\n...\n"} {
		table, err := Parse([]byte(body), FormatYAML, Limits{})
		require.NoError(t, err, body)
		require.Empty(t, table.Rows, body)
	}
}

func TestParseYAML_TooManyRows(t *testing.T) {

	_, err := Parse([]byte("- a: 1\n- a: 2\n- a: 3\n"), FormatYAML, Limits{MaxRows: 2})
	require.True(t, derp.IsBadRequest(err))
}

func TestParseYAML_StopsAtTheProblemLimit(t *testing.T) {

	table, err := Parse([]byte("- 1\n- 2\n- 3\n- 4\n"), FormatYAML, Limits{MaxProblems: 2})
	require.NoError(t, err)
	require.Len(t, table.Problems, 3)
	require.Equal(t, "Too many problems; stopped reading here", table.Problems[2].Message)
}
