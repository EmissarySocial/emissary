package content

// Encrypted represents encrypted content that is not visible to the server.  It is encrypted with a key that is only
// accessible to clients via the KMS, and will be decrypted on the client systems only.
type Encrypted struct {
	Content string
}

// HTML returns the HTML representation of this content type.  It is required to implement the Content interface
func (encrypted *Encrypted) HTML() string {
	return `<encrypted-content keyId="">` + encrypted.Content + `</encrypted-content>`
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (encrypted *Encrypted) WebComponents(accumulator map[string]bool) {

	accumulator["/components/content-encrypted.js"] = true
	return
}
