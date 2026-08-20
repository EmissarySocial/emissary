package ashash

import "github.com/benpate/hannibal/streams"

// testInnerClient is a stub streams.Client that serves a fixed set of documents to the tests in this package
type testInnerClient struct{}

// SetRootClient implements the streams.Client interface. The stub ignores the root client.
func (client testInnerClient) SetRootClient(rootClient streams.Client) {}

// Load implements the streams.Client interface, and returns one of this stub's canned documents
func (client testInnerClient) Load(url string, options ...any) (streams.Document, error) {

	switch url {
	case "http://example.com/with-hash":
		return streams.NewDocument(map[string]any{
			"id":      "http://example.com/without-hash",
			"name":    "With Hash",
			"summary": "It's my hash and I can cry if I want to",
			"collection": map[string]any{
				"id":      "http://example.com/with-hash#hash",
				"name":    "Here's the Hash",
				"summary": "Done somebody gots a hash, now.",
			},
		}), nil

	case "http://example.com/without-hash":
		return streams.NewDocument(map[string]any{
			"id":      "http://example.com/without-hash",
			"name":    "Without Hash",
			"summary": "Ain't nobody got no hash",
		}), nil

	}
	return streams.NilDocument(), nil
}

// Save implements the streams.Client interface. The stub discards all writes.
func (client testInnerClient) Save(document streams.Document) error {
	return nil
}

// Delete implements the streams.Client interface. The stub discards all deletes.
func (client testInnerClient) Delete(documentID string) error {
	return nil
}
