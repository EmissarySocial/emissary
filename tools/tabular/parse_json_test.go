package tabular

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestParseJSON_FlattensAndKeepsOrder(t *testing.T) {

	body := `[
	{"token": "paris", "location": {"longitude": 2.35, "latitude": 48.85}, "label": "Paris"},
	{"token": "lyon", "rank": 2, "featured": true, "summary": null, "tags": ["a", "b"]}
]`

	table, err := Parse([]byte(body), FormatJSON, Limits{})
	require.NoError(t, err)
	require.Equal(t, FormatJSON, table.Format)
	require.False(t, table.HasProblems())

	require.Equal(t, []Row{
		{Number: 1, Line: 2, Cells: []Cell{
			{"token", "paris"},
			{"location.longitude", 2.35},
			{"location.latitude", 48.85},
			{"label", "Paris"},
		}},
		{Number: 2, Line: 3, Cells: []Cell{
			{"token", "lyon"},
			{"rank", int64(2)},
			{"featured", true},
			{"summary", nil},
			{"tags", []any{"a", "b"}},
		}},
	}, table.Rows)
}

func TestParseJSON_Numbers(t *testing.T) {

	body := `[{"int": 9007199254740993, "float": 1.5, "exp": 1e3, "huge": 1e400}]`

	table, err := Parse([]byte(body), FormatJSON, Limits{})
	require.NoError(t, err)

	cells := table.Rows[0].Cells
	require.Equal(t, int64(9007199254740993), cells[0].Value, "integers keep every digit")
	require.Equal(t, 1.5, cells[1].Value)
	require.Equal(t, float64(1000), cells[2].Value)
	require.Equal(t, "1e400", cells[3].Value, "a number too large for a float64 keeps its digits")
}

func TestParseJSON_ObjectsInsideArraysBecomeMaps(t *testing.T) {

	table, err := Parse([]byte(`[{"token":"a","links":[{"href":"x","rel":"y"}]}]`), FormatJSON, Limits{})
	require.NoError(t, err)

	value, ok := table.Rows[0].Get("links")
	require.True(t, ok)
	require.Equal(t, []any{map[string]any{"href": "x", "rel": "y"}}, value)
}

func TestParseJSON_EntriesThatAreNotObjects(t *testing.T) {

	table, err := Parse([]byte("[\n{\"token\":\"a\"},\n3,\n\"text\",\n[1]\n]"), FormatJSON, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Rows, 1)

	require.Equal(t, []Problem{
		{Row: 2, Line: 3, Message: "Entry must be an object"},
		{Row: 3, Line: 4, Message: "Entry must be an object"},
		{Row: 4, Line: 5, Message: "Entry must be an object"},
	}, table.Problems)
}

func TestParseJSON_RowProblems(t *testing.T) {

	test := func(entry string, column string, message string) {
		t.Helper()

		table, err := Parse([]byte("["+entry+"]"), FormatJSON, Limits{MaxColumns: 3, MaxCellBytes: 5, MaxDepth: 2})
		require.NoError(t, err, entry)
		require.Empty(t, table.Rows, entry)
		require.Len(t, table.Problems, 1, entry)
		require.Equal(t, column, table.Problems[0].Column, entry)
		require.Equal(t, message, table.Problems[0].Message, entry)
	}

	test(`{"a":{"b":1},"a.b":2}`, "a.b", "Path appears more than once")
	test(`{"a":1,"a":2}`, "a", "Path appears more than once")
	test(`{"":1}`, "", "Key is empty")
	test(`{"a":{"":1}}`, "a.", "Key is empty")
	test(`{"a":1,"b":2,"c":3,"d":4}`, "d", "Row has more than 3 columns")
	test(`{"a":"123456"}`, "a", "Value is longer than 5 bytes")
	test(`{"a":["123456"]}`, "a", "Value is longer than 5 bytes")
	test(`{"a":{"b":{"c":1}}}`, "a.b", "Value is nested more than 2 levels deep")
	test(`{"a":[[1]]}`, "a", "Value is nested more than 2 levels deep")
	test(`{"a":[{"b":1}]}`, "a", "Value is nested more than 2 levels deep")
}

func TestParseJSON_LongValueInsideAnArrayOfObjects(t *testing.T) {

	table, err := Parse([]byte(`[{"a":[{"b":"123456"}]}]`), FormatJSON, Limits{MaxCellBytes: 5, MaxDepth: 3})
	require.NoError(t, err)
	require.Equal(t, []Problem{{Row: 1, Line: 1, Column: "a", Message: "Value is longer than 5 bytes"}}, table.Problems)
}

func TestParseJSON_WithinLimitsIsAccepted(t *testing.T) {

	table, err := Parse([]byte(`[{"a":{"b":[1]},"c":"12345"}]`), FormatJSON, Limits{MaxColumns: 2, MaxCellBytes: 5, MaxDepth: 3})
	require.NoError(t, err)
	require.False(t, table.HasProblems())
	require.Equal(t, []Cell{{"a.b", []any{int64(1)}}, {"c", "12345"}}, table.Rows[0].Cells)
}

func TestParseJSON_EmptyArray(t *testing.T) {

	table, err := Parse([]byte(" [ ] "), FormatJSON, Limits{})
	require.NoError(t, err)
	require.Empty(t, table.Rows)
}

func TestParseJSON_FileErrors(t *testing.T) {

	test := func(body string) {
		t.Helper()
		_, err := Parse([]byte(body), FormatJSON, Limits{})
		require.Error(t, err, body)
		require.True(t, derp.IsBadRequest(err), body)
	}

	test(`{"items":[]}`)
	test(`"text"`)
	test(`3`)
	test(`[{"a":1}] extra`)
	test(`[{"a":1}][]`)
	test(`[{"a":1},`)
	test(`[{"a":}]`)
	test(`[{"a":1]`)
	test(`]`)
	test(`[{"a":1}`)
	test(`[{1:2}]`)
	test(`[{"a":[1,]}]`)
	test(`[{"a":[1}]`)
	test(`[{"a":1}}`)
}

func TestParseJSON_SyntaxErrorsNameTheLine(t *testing.T) {

	_, err := Parse([]byte("[\n{\"a\":1},\n{\"b\":}\n]"), FormatJSON, Limits{})
	require.Error(t, err)
	require.Contains(t, derp.Message(err)+" "+strings.Join(errorDetails(err), " "), "line 3")
}

func TestParseJSON_TooManyRows(t *testing.T) {

	_, err := Parse([]byte(`[{"a":1},{"a":2},{"a":3}]`), FormatJSON, Limits{MaxRows: 2})
	require.True(t, derp.IsBadRequest(err))
}

func TestParseJSON_StopsAtTheProblemLimit(t *testing.T) {

	table, err := Parse([]byte(`[1,2,3,4,5]`), FormatJSON, Limits{MaxProblems: 2})
	require.NoError(t, err)
	require.Len(t, table.Problems, 3)
	require.Equal(t, "Too many problems; stopped reading here", table.Problems[2].Message)
}

// errorDetails returns the details a derp error carries, as text
func errorDetails(err error) []string {

	result := make([]string, 0)

	if derpError, isDerpError := err.(derp.Error); isDerpError {
		for _, detail := range derpError.Details {
			if text, isText := detail.(string); isText {
				result = append(result, text)
			}
		}
	}

	return result
}
