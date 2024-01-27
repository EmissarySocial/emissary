package ashash

import "github.com/benpate/hannibal/streams"

type testInnerClient struct{}

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
