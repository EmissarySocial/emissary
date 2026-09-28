package build

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/content"
)

// StreamSourceFiles summarizes the files that a Stream's remote source has copied in, for the
// settings screen
type StreamSourceFiles struct {
	Ready       int      // Files that are stored and served
	Pending     int      // Files still being downloaded, or waiting for a retry
	Failed      int      // Files that could not be copied, and will be tried again on the next sync
	FailedNames []string // Filenames of the failed files, in document order
	AtLimit     bool     // TRUE when the source links to as many files as may be copied
}

// NewStreamSourceFiles summarizes a Stream's imported attachments
func NewStreamSourceFiles(attachments []model.Attachment) StreamSourceFiles {

	result := StreamSourceFiles{
		FailedNames: make([]string, 0),
		AtLimit:     len(attachments) >= content.MaxAttachments,
	}

	for _, attachment := range attachments {

		switch attachment.Status {

		case model.AttachmentStatusReady:
			result.Ready++

		case model.AttachmentStatusFailed:
			result.Failed++
			result.FailedNames = append(result.FailedNames, attachment.Original)

		default:
			result.Pending++
		}
	}

	return result
}

// Total returns the number of files the source links to, up to the limit
func (files StreamSourceFiles) Total() int {
	return files.Ready + files.Pending + files.Failed
}
