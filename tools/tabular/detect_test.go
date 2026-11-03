package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetect_ContentType(t *testing.T) {

	test := func(contentType string, expected Format) {
		t.Helper()
		require.Equal(t, expected, Detect(contentType, "", nil), "contentType %q", contentType)
	}

	test("text/csv", FormatCSV)
	test("text/csv; charset=utf-8", FormatCSV)
	test("TEXT/CSV", FormatCSV)
	test("application/csv", FormatCSV)
	test("text/tab-separated-values", FormatTSV)
	test("application/json", FormatJSON)
	test("text/json", FormatJSON)
	test("application/yaml", FormatYAML)
	test("application/x-yaml", FormatYAML)
	test("text/yaml", FormatYAML)
	test("text/x-yaml", FormatYAML)

	// Vague, unknown, or broken types fall through, and there is nothing else to go on
	test("text/plain", FormatUnknown)
	test("application/octet-stream", FormatUnknown)
	test("application/geo+json", FormatUnknown)
	test("", FormatUnknown)
	test(";;;", FormatUnknown)
}

func TestDetect_Name(t *testing.T) {

	test := func(name string, expected Format) {
		t.Helper()
		require.Equal(t, expected, Detect("text/plain", name, nil), "name %q", name)
	}

	test("cities.csv", FormatCSV)
	test("CITIES.CSV", FormatCSV)
	test("cities.tsv", FormatTSV)
	test("cities.tab", FormatTSV)
	test("cities.json", FormatJSON)
	test("cities.yaml", FormatYAML)
	test("cities.yml", FormatYAML)
	test("https://example.com/data/cities.csv?token=abc#top", FormatCSV)
	test("https://example.com/data/cities.yml", FormatYAML)

	// No extension, or one that names nothing, falls through
	test("https://example.com/export?format=csv", FormatUnknown)
	test("cities.txt", FormatUnknown)
	test("cities", FormatUnknown)
	test("", FormatUnknown)
	test("%zz.csv", FormatCSV)
}

func TestDetect_Content(t *testing.T) {

	test := func(body string, expected Format) {
		t.Helper()
		require.Equal(t, expected, Detect("", "", []byte(body)), "body %q", body)
	}

	test("[{\"token\":\"a\"}]", FormatJSON)
	test("  \n\t[]", FormatJSON)
	test("{\"items\":[]}", FormatJSON)
	test("- token: a\n", FormatYAML)
	test("---\n- token: a\n", FormatYAML)
	test("# cities\n\n- token: a\n", FormatYAML)
	test("token,label\na,b\n", FormatCSV)
	test("token\tlabel\na\tb\n", FormatTSV)
	test("token\n", FormatCSV)
	test("a,b\tc\n", FormatCSV)
	test("a\tb\tc,d\n", FormatTSV)
	test(byteOrderMark+"token,label\n", FormatCSV)
	test(byteOrderMark+"[]", FormatJSON)
	test("\r\n[]", FormatJSON)

	// Nothing meaningful at all
	test("", FormatUnknown)
	test("   \n\n\t ", FormatUnknown)
	test("# only a comment\n", FormatUnknown)
	test(byteOrderMark, FormatUnknown)
}

func TestDetect_Precedence(t *testing.T) {

	// A specific Content-Type beats the name, and the name beats the content
	require.Equal(t, FormatCSV, Detect("text/csv", "cities.json", []byte("[]")))
	require.Equal(t, FormatJSON, Detect("text/plain", "cities.json", []byte("token,label")))
	require.Equal(t, FormatTSV, Detect("text/plain", "cities", []byte("token\tlabel")))
}

func FuzzDetect(f *testing.F) {

	f.Add("text/csv", "a.csv", []byte("token,label"))
	f.Add("", "", []byte("[]"))
	f.Add("text/plain", "https://x/y.yml?z", []byte("- a: b"))
	f.Add("", "", []byte(byteOrderMark+"\n# c\n\t,"))

	known := map[Format]bool{FormatUnknown: true, FormatCSV: true, FormatTSV: true, FormatJSON: true, FormatYAML: true}

	f.Fuzz(func(t *testing.T, contentType string, name string, body []byte) {
		require.True(t, known[Detect(contentType, name, body)])
	})
}
