package content

import (
	"net/url"
	"path"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"golang.org/x/net/html"
)

// AttachmentRef is one link in a rendered document that names a file to copy into Emissary
type AttachmentRef struct {
	URL  string // Absolute address of the remote file, resolved against the source
	Kind string // image, video, audio, or document, as the file extension names it
}

// FindAttachments returns every link in a rendered document that names a file in an attachments
// folder on the source's own host, in order of first appearance and without duplicates
func FindAttachments(document string, sourceURL string) []AttachmentRef {

	base, err := url.Parse(sourceURL)

	// An unreadable source address has nothing to resolve against, so nothing is imported
	if err != nil {
		return nil
	}

	for result, seen, tokenizer := make([]AttachmentRef, 0), make(map[string]bool), html.NewTokenizer(strings.NewReader(document)); ; {
		switch tokenizer.Next() {

		case html.ErrorToken:
			return result

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()

			for _, attribute := range token.Attr {

				ref, isAttachment := linkedAttachment(base, token.Data, attribute)

				if !isAttachment || seen[ref.URL] {
					continue
				}

				seen[ref.URL] = true
				result = append(result, ref)
			}
		}
	}
}

// RewriteAttachments replaces every attachment link in a rendered document with its local
// address.  An image link to a video or audio file becomes a player.  Every other byte of the
// document is written back as it was.
func RewriteAttachments(document string, sourceURL string, localURLs map[string]string) string {

	base, err := url.Parse(sourceURL)

	if err != nil {
		return document
	}

	var result strings.Builder
	result.Grow(len(document))

	for tokenizer := html.NewTokenizer(strings.NewReader(document)); ; {
		tokenType := tokenizer.Next()

		if tokenType == html.ErrorToken {
			return result.String()
		}

		// RULE: Raw bytes are copied, never re-rendered.  Rendering a parsed document normalizes
		// quoting and entities everywhere, which would make every sync rewrite the whole page.
		raw := string(tokenizer.Raw())

		if (tokenType != html.StartTagToken) && (tokenType != html.SelfClosingTagToken) {
			result.WriteString(raw)
			continue
		}

		token := tokenizer.Token()
		rewritten, changed := rewriteTag(base, token, localURLs)

		if !changed {
			result.WriteString(raw)
			continue
		}

		result.WriteString(rewritten)
	}
}

/******************************************
 * Helper Functions
 ******************************************/

// rewriteTag returns a tag with its attachment link replaced by the local address, and FALSE
// when the tag links to no imported file
func rewriteTag(base *url.URL, token html.Token, localURLs map[string]string) (string, bool) {

	for index, attribute := range token.Attr {

		ref, isAttachment := linkedAttachment(base, token.Data, attribute)

		if !isAttachment {
			continue
		}

		localURL, isImported := localURLs[ref.URL]

		if !isImported {
			continue
		}

		// The sanitizer removes <video> and <audio>, so the author writes an image link and the
		// player is built here, from an address this server issued
		if token.Data == "img" {
			switch ref.Kind {
			case model.AttachmentMediaTypeVideo, model.AttachmentMediaTypeAudio:
				return mediaPlayer(ref.Kind, localURL, attributeValue(token, "alt")), true
			}
		}

		token.Attr[index].Val = localURL
		return token.String(), true
	}

	return "", false
}

// mediaPlayer returns a <video> or <audio> element, with the alt text as its fallback content
func mediaPlayer(kind string, localURL string, fallback string) string {

	tag := "video"

	if kind == model.AttachmentMediaTypeAudio {
		tag = "audio"
	}

	return "<" + tag + ` controls src="` + html.EscapeString(localURL) + `">` + html.EscapeString(fallback) + "</" + tag + ">"
}

// attributeValue returns the value of a tag's named attribute, or "" when it has none
func attributeValue(token html.Token, name string) string {

	for _, attribute := range token.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}

	return ""
}

// linkedAttachment returns the file an attribute links to, and FALSE when the attribute is not
// a link, or links to anything other than an attachment
func linkedAttachment(base *url.URL, tagName string, attribute html.Attribute) (AttachmentRef, bool) {

	// RULE: Only the two links that survive the sanitizer are read
	switch {
	case (tagName == "img") && (attribute.Key == "src"):
	case (tagName == "a") && (attribute.Key == "href"):
	default:
		return AttachmentRef{}, false
	}

	return resolveAttachment(base, attribute.Val)
}

// resolveAttachment resolves a link against its source, and returns it when it names a file in
// an attachments folder on the source's own host
func resolveAttachment(base *url.URL, value string) (AttachmentRef, bool) {

	reference, err := url.Parse(strings.TrimSpace(value))

	if err != nil {
		return AttachmentRef{}, false
	}

	resolved := base.ResolveReference(reference)
	resolved.Fragment = ""
	resolved.RawFragment = ""

	// RULE: Only the source's own host.  A link anywhere else is the author pointing at somebody
	// else's file, which stays a link.
	if (resolved.Scheme != base.Scheme) || !strings.EqualFold(resolved.Host, base.Host) {
		return AttachmentRef{}, false
	}

	if !inAttachmentFolder(resolved.Path) {
		return AttachmentRef{}, false
	}

	kind := attachmentKindByPath(resolved.Path)

	if kind == "" {
		return AttachmentRef{}, false
	}

	// RULE: An address too long to store stays a link, rather than failing the whole sync
	address := resolved.String()

	if len(address) > maxURLLength {
		return AttachmentRef{}, false
	}

	return AttachmentRef{URL: address, Kind: kind}, true
}

// inAttachmentFolder returns TRUE if a path names a file inside a folder called "attachments",
// at any depth
func inAttachmentFolder(value string) bool {

	segments := strings.Split(value, "/")

	// The last segment is the filename, which is never the folder
	for _, segment := range segments[:len(segments)-1] {
		if segment == attachmentFolder {
			return true
		}
	}

	return false
}

// AttachmentKind returns the kind of file that an address names by its extension, or "" when
// the extension is not one that may be imported
func AttachmentKind(address string) string {

	parsed, err := url.Parse(address)

	if err != nil {
		return ""
	}

	// RULE: Read the PATH, never the whole address, so a query string cannot hide the extension
	return attachmentKindByPath(parsed.Path)
}

// attachmentKindByPath returns the kind of file that a path names by its extension, or "" when
// the extension is not one that may be imported
func attachmentKindByPath(value string) string {

	switch strings.ToLower(path.Ext(value)) {

	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return model.AttachmentMediaTypeImage

	case ".mp4", ".m4v", ".webm", ".mov":
		return model.AttachmentMediaTypeVideo

	case ".mp3", ".m4a", ".aac", ".ogg", ".oga", ".opus", ".wav", ".flac":
		return model.AttachmentMediaTypeAudio

	case ".pdf":
		return model.AttachmentMediaTypeDocument
	}

	return ""
}

// MaxAttachmentBytes returns the largest file of a kind that will be downloaded
func MaxAttachmentBytes(kind string) int64 {

	switch kind {

	case model.AttachmentMediaTypeVideo, model.AttachmentMediaTypeAudio:
		return maxMediaBytes
	}

	return maxImageBytes
}
