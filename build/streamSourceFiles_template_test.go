package build

import (
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// silentIcons is an icon.Provider that renders nothing
type silentIcons struct{}

// Get implements the icon.Provider interface
func (silentIcons) Get(string) string { return "" }

// Write implements the icon.Provider interface
func (silentIcons) Write(string, io.Writer) {}

// statusCardBuilder stands in for a Stream builder, offering what article-remote's edit-status.html reads
type statusCardBuilder struct {
	files StreamSourceFiles
}

// StreamID returns a fixed Stream ID
func (builder statusCardBuilder) StreamID() string {
	return "62f8f1d4d7f1c8e6b4a1c2d3"
}

// ContentFormat returns the format of a synchronized Markdown file
func (builder statusCardBuilder) ContentFormat() string {
	return model.ContentFormatMarkdown
}

// StreamSource returns a record that has synchronized once
func (builder statusCardBuilder) StreamSource() (model.StreamSource, error) {

	result := model.StreamSource{}
	result.URL = "https://raw.example.com/docs/guide.md"
	result.Status = model.StreamSourceStatusSuccess
	result.LastSynced = 1700000000

	return result, nil
}

// StreamSourceFiles returns the summary the test chose
func (builder statusCardBuilder) StreamSourceFiles() (StreamSourceFiles, error) {
	return builder.files, nil
}

// renderStatusCard renders article-remote's edit-status.html, with the real funcMap, against a builder
func renderStatusCard(t *testing.T, builder statusCardBuilder) string {

	t.Helper()

	statusTemplate, err := template.New("edit-status.html").
		Funcs(emissarytemplates.FuncMap(silentIcons{})).
		ParseFiles("../_embed/templates/stream-article-remote/edit-status.html")

	require.NoError(t, err)

	var buffer strings.Builder
	require.NoError(t, statusTemplate.Execute(&buffer, builder))

	return strings.Join(strings.Fields(buffer.String()), " ")
}

// TestArticleRemoteStatusCard_FilesRow confirms that the status card renders the file summary,
// names the files that failed, and says when the limit was reached
func TestArticleRemoteStatusCard_FilesRow(t *testing.T) {

	files := NewStreamSourceFiles([]model.Attachment{
		{Original: "flow.png", Status: model.AttachmentStatusReady},
		{Original: "demo.mp4", Status: model.AttachmentStatusWorking},
		{Original: "missing.png", Status: model.AttachmentStatusFailed},
		{Original: "wrong.pdf", Status: model.AttachmentStatusFailed},
	})

	page := renderStatusCard(t, statusCardBuilder{files: files})

	require.Contains(t, page, ">Files<")
	require.Contains(t, page, "1 of 4 copied, 1 in progress")
	require.Contains(t, page, "Could not copy: missing.png, wrong.pdf")
	require.NotContains(t, page, "Only the first 50")

	files.AtLimit = true
	require.Contains(t, renderStatusCard(t, statusCardBuilder{files: files}), "Only the first 50 linked files are copied.")
}

// TestArticleRemoteStatusCard_NoFilesNoRow confirms that a source linking to nothing shows no Files row
func TestArticleRemoteStatusCard_NoFilesNoRow(t *testing.T) {

	page := renderStatusCard(t, statusCardBuilder{files: NewStreamSourceFiles(nil)})

	require.Contains(t, page, "Synchronized", "the card rendered")
	require.NotContains(t, page, ">Files<")
}
