package tabular

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestParseDelimited_CSV(t *testing.T) {

	body := "token,label,location.latitude\nparis,Paris,48.8566\nlyon,\"Lyon, France\",45.764\n"

	table, err := Parse([]byte(body), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Equal(t, FormatCSV, table.Format)
	require.False(t, table.HasProblems())

	require.Equal(t, []Row{
		{Number: 1, Line: 2, Cells: []Cell{{"token", "paris"}, {"label", "Paris"}, {"location.latitude", "48.8566"}}},
		{Number: 2, Line: 3, Cells: []Cell{{"token", "lyon"}, {"label", "Lyon, France"}, {"location.latitude", "45.764"}}},
	}, table.Rows)
}

func TestParseDelimited_TSV(t *testing.T) {

	body := "token\tlabel\nparis\tParis, France\n"

	table, err := Parse([]byte(body), FormatTSV, Limits{})
	require.NoError(t, err)
	require.Equal(t, FormatTSV, table.Format)
	require.Equal(t, []Cell{{"token", "paris"}, {"label", "Paris, France"}}, table.Rows[0].Cells)
}

func TestParseDelimited_HeadingsAreTrimmed(t *testing.T) {

	table, err := Parse([]byte(" token , label \na,b\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Equal(t, []Cell{{"token", "a"}, {"label", "b"}}, table.Rows[0].Cells)
}

func TestParseDelimited_ValuesAreKeptAsWritten(t *testing.T) {

	table, err := Parse([]byte("token,label,summary\na, spaced ,\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Equal(t, []Cell{{"token", "a"}, {"label", " spaced "}, {"summary", ""}}, table.Rows[0].Cells)
}

func TestParseDelimited_LinesFollowQuotedNewlines(t *testing.T) {

	body := "token,summary\na,\"one\ntwo\"\nb,three\n"

	table, err := Parse([]byte(body), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Rows, 2)
	require.Equal(t, 2, table.Rows[0].Line)
	require.Equal(t, "one\ntwo", table.Rows[0].Cells[1].Value)
	require.Equal(t, 4, table.Rows[1].Line)
	require.Equal(t, 2, table.Rows[1].Number)
}

func TestParseDelimited_BlankLinesAreSkipped(t *testing.T) {

	table, err := Parse([]byte("\n\ntoken\n\na\n\nb\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Rows, 2)
	require.Equal(t, 5, table.Rows[0].Line)
	require.Equal(t, 7, table.Rows[1].Line)
}

func TestParseDelimited_HeaderOnly(t *testing.T) {

	table, err := Parse([]byte("token,label\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Empty(t, table.Rows)
	require.False(t, table.HasProblems())
}

func TestParseDelimited_RaggedRowsAreProblems(t *testing.T) {

	body := "token,label\nparis,Paris\nlyon\nnice,Nice,extra\nmetz,Metz\n"

	table, err := Parse([]byte(body), FormatCSV, Limits{})
	require.NoError(t, err)

	// The good rows survive, keeping their own numbers
	require.Len(t, table.Rows, 2)
	require.Equal(t, 1, table.Rows[0].Number)
	require.Equal(t, 4, table.Rows[1].Number)

	require.Equal(t, []Problem{
		{Row: 2, Line: 3, Message: "Row has 1 cells, but the header has 2"},
		{Row: 3, Line: 4, Message: "Row has 3 cells, but the header has 2"},
	}, table.Problems)
}

func TestParseDelimited_BadHeadings(t *testing.T) {

	table, err := Parse([]byte("token,,label,token\na,b,c,d\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Empty(t, table.Rows, "rows mean nothing without good headings")

	require.Equal(t, []Problem{
		{Line: 1, Column: "", Message: "Heading 2 is empty"},
		{Line: 1, Column: "token", Message: "Heading appears more than once"},
	}, table.Problems)
}

func TestParseDelimited_HeadingProblemsNameTheirLine(t *testing.T) {

	table, err := Parse([]byte("\n\ntoken,\n"), FormatCSV, Limits{})
	require.NoError(t, err)
	require.Len(t, table.Problems, 1)
	require.Equal(t, 3, table.Problems[0].Line)
}

func TestParseDelimited_LongCellIsAProblem(t *testing.T) {

	body := "token,summary\na," + strings.Repeat("x", 11) + "\nb,short\n"

	table, err := Parse([]byte(body), FormatCSV, Limits{MaxCellBytes: 10})
	require.NoError(t, err)
	require.Len(t, table.Rows, 1)
	require.Equal(t, []Problem{{Row: 1, Line: 2, Column: "summary", Message: "Value is longer than 10 bytes"}}, table.Problems)
}

func TestParseDelimited_TooManyRows(t *testing.T) {

	_, err := Parse([]byte("token\na\nb\nc\n"), FormatCSV, Limits{MaxRows: 2})
	require.True(t, derp.IsBadRequest(err))

	table, err := Parse([]byte("token\na\nb\n"), FormatCSV, Limits{MaxRows: 2})
	require.NoError(t, err)
	require.Len(t, table.Rows, 2)
}

func TestParseDelimited_TooManyColumns(t *testing.T) {

	_, err := Parse([]byte("a,b,c\n1,2,3\n"), FormatCSV, Limits{MaxColumns: 2})
	require.True(t, derp.IsBadRequest(err))
}

func TestParseDelimited_SyntaxErrorsEndTheFile(t *testing.T) {

	// A bare quote inside an unquoted field
	_, err := Parse([]byte("token,label\na,b\"c\n"), FormatCSV, Limits{})
	require.True(t, derp.IsBadRequest(err))

	// An unterminated quote in the header
	_, err = Parse([]byte("\"token,label\n"), FormatCSV, Limits{})
	require.True(t, derp.IsBadRequest(err))
}

func TestParseDelimited_StopsAtTheProblemLimit(t *testing.T) {

	body := "token,label\n" + strings.Repeat("x\n", 10)

	table, err := Parse([]byte(body), FormatCSV, Limits{MaxProblems: 3})
	require.NoError(t, err)
	require.Len(t, table.Problems, 4)
	require.Equal(t, "Too many problems; stopped reading here", table.Problems[3].Message)
	require.Equal(t, 3, table.Problems[3].Row)
}

func TestParseDelimited_StopsAtTheProblemLimitForLongCells(t *testing.T) {

	body := "token\n" + strings.Repeat("xxxx\n", 5)

	table, err := Parse([]byte(body), FormatCSV, Limits{MaxCellBytes: 3, MaxProblems: 2})
	require.NoError(t, err)
	require.Empty(t, table.Rows)
	require.Len(t, table.Problems, 3)
}

func TestParseDelimited_HeadingProblemsStopAtTheLimit(t *testing.T) {

	// Found by FuzzParse: six empty headings once produced eight problems against a limit of five
	table, err := Parse([]byte(",,,,,\n"), FormatCSV, Limits{MaxProblems: 5})
	require.NoError(t, err)
	require.Len(t, table.Problems, 6)
	require.Equal(t, "Too many problems; stopped reading here", table.Problems[5].Message)
}
