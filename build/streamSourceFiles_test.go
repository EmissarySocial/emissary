package build

import (
	"strconv"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestNewStreamSourceFiles counts files by status, and names the failed ones in order
func TestNewStreamSourceFiles(t *testing.T) {

	attachments := []model.Attachment{
		{Original: "a.png", Status: model.AttachmentStatusReady},
		{Original: "b.png", Status: model.AttachmentStatusFailed},
		{Original: "c.mp4", Status: model.AttachmentStatusWorking},
		{Original: "d.pdf", Status: model.AttachmentStatusFailed},
		{Original: "e.png", Status: ""},
	}

	files := NewStreamSourceFiles(attachments)

	require.Equal(t, 1, files.Ready)
	require.Equal(t, 2, files.Pending, "anything not READY or FAILED is still on its way")
	require.Equal(t, 2, files.Failed)
	require.Equal(t, []string{"b.png", "d.pdf"}, files.FailedNames)
	require.Equal(t, 5, files.Total())
	require.False(t, files.AtLimit)
}

// TestNewStreamSourceFiles_Empty describes a source that links to nothing
func TestNewStreamSourceFiles_Empty(t *testing.T) {

	files := NewStreamSourceFiles(nil)

	require.Zero(t, files.Total())
	require.NotNil(t, files.FailedNames, "a template ranges over it")
}

// TestNewStreamSourceFiles_AtLimit says so when the source links to as many files as may be copied
func TestNewStreamSourceFiles_AtLimit(t *testing.T) {

	attachments := make([]model.Attachment, 0, 50)

	for index := range 50 {
		attachments = append(attachments, model.Attachment{Original: strconv.Itoa(index) + ".png", Status: model.AttachmentStatusReady})
	}

	require.True(t, NewStreamSourceFiles(attachments).AtLimit)
	require.False(t, NewStreamSourceFiles(attachments[:49]).AtLimit)
}
