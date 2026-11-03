package tabular

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestParse_RefusesAFileOverTheByteLimit(t *testing.T) {

	body := []byte(strings.Repeat("a", 101))

	_, err := Parse(body, FormatCSV, Limits{MaxBytes: 100})
	require.Error(t, err)
	require.True(t, derp.IsBadRequest(err))

	// Exactly at the limit is allowed
	_, err = Parse(body[:100], FormatCSV, Limits{MaxBytes: 100})
	require.NoError(t, err)
}

func TestParse_RefusesTextThatIsNotUTF8(t *testing.T) {

	// Latin-1 "é", as a Windows spreadsheet might export it
	body := []byte("token,label\ncafe,caf\xe9\n")

	for _, format := range []Format{FormatCSV, FormatTSV, FormatJSON, FormatYAML} {
		_, err := Parse(body, format, Limits{})
		require.Error(t, err, format)
		require.True(t, derp.IsBadRequest(err), format)
	}
}

func TestParse_RefusesAnUnknownFormat(t *testing.T) {

	_, err := Parse([]byte("token\na"), FormatUnknown, Limits{})
	require.True(t, derp.IsBadRequest(err))

	_, err = Parse([]byte("token\na"), Format("TOML"), Limits{})
	require.True(t, derp.IsBadRequest(err))
}

func TestParse_StripsTheByteOrderMark(t *testing.T) {

	for _, format := range []Format{FormatCSV, FormatJSON, FormatYAML} {

		body := map[Format]string{
			FormatCSV:  "token,label\nparis,Paris\n",
			FormatJSON: `[{"token":"paris","label":"Paris"}]`,
			FormatYAML: "- token: paris\n  label: Paris\n",
		}[format]

		table, err := Parse([]byte(byteOrderMark+body), format, Limits{})
		require.NoError(t, err, format)
		require.Len(t, table.Rows, 1, format)

		value, ok := table.Rows[0].Get("token")
		require.True(t, ok, format)
		require.Equal(t, "paris", value, format)
	}
}

func TestParse_EmptyFilesHaveNoRows(t *testing.T) {

	for _, format := range []Format{FormatCSV, FormatTSV, FormatJSON, FormatYAML} {
		table, err := Parse([]byte(""), format, Limits{})
		require.NoError(t, err, format)
		require.Equal(t, format, table.Format)
		require.Empty(t, table.Rows, format)
		require.False(t, table.HasProblems(), format)
	}
}

func FuzzParse(f *testing.F) {

	f.Add([]byte("token,label\nparis,Paris\nlyon\n"), uint8(0))
	f.Add([]byte("token\tlabel\nparis\tParis\n"), uint8(1))
	f.Add([]byte(`[{"token":"a","location":{"latitude":1.5}},3,{"a":{"b":1},"a.b":2}]`), uint8(2))
	f.Add([]byte("- token: a\n  tags: [x, y]\n- &r {token: b}\n- *r\n"), uint8(3))
	f.Add([]byte("\"unterminated,quote\n"), uint8(0))
	f.Add([]byte(`{"items":[]}`), uint8(2))
	f.Add([]byte("---\n- a\n---\n- b\n"), uint8(3))

	formats := []Format{FormatCSV, FormatTSV, FormatJSON, FormatYAML}

	limits := Limits{
		MaxBytes:     4096,
		MaxRows:      20,
		MaxColumns:   8,
		MaxCellBytes: 64,
		MaxDepth:     3,
		MaxProblems:  5,
	}

	f.Fuzz(func(t *testing.T, body []byte, which uint8) {

		format := formats[int(which)%len(formats)]
		table, err := Parse(body, format, limits)

		// Error XOR a usable table
		if err != nil {
			require.Empty(t, table.Rows)
			return
		}

		require.Equal(t, format, table.Format)
		require.LessOrEqual(t, len(table.Rows), limits.MaxRows)
		require.LessOrEqual(t, len(table.Problems), limits.MaxProblems+1)

		for _, row := range table.Rows {

			require.Positive(t, row.Number)
			require.LessOrEqual(t, len(row.Cells), limits.MaxColumns)

			seen := make(map[string]bool, len(row.Cells))

			for _, cell := range row.Cells {
				require.NotEmpty(t, cell.Path)
				require.False(t, seen[cell.Path], "path %q repeats", cell.Path)
				seen[cell.Path] = true

				if text, isString := cell.Value.(string); isString {
					require.LessOrEqual(t, len(text), limits.MaxCellBytes)
				}
			}
		}
	})
}
