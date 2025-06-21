package nodeinfo

import (
	"slices"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/slice"
)

type Client struct {
	options []remote.Option
}

func NewClient(options ...remote.Option) Client {
	result := Client{
		options: make([]remote.Option, 0),
	}

	result.With(options...)

	return result
}

func (client *Client) With(options ...remote.Option) *Client {
	client.options = append(client.options, options...)
	return client
}

func (client *Client) Load(server string) (NodeInfo, error) {

	const location = "nodeinfo.Client.Load"

	wellknown := NewWellknown()

	txn := remote.Get(server + "/.well-known/nodeinfo").
		With(client.options...).
		Result(&wellknown)

	if err := txn.Send(); err != nil {
		return NewNodeInfo(), derp.Wrap(err, location, "Error retrieving /.well-known/nodeinfo")
	}

	// Limit to only "nodeinfo" links
	wellknown.Links = slice.Filter(wellknown.Links, func(link Link) bool {
		return strings.HasPrefix(link.Rel, RelationPrefix)
	})

	// Sort links by version DESCENDING
	slices.SortFunc(wellknown.Links, func(a, b Link) int {
		return compare.String(a.Rel, b.Rel)
	})

	// Try each link in order (highest version to lowest)
	for _, link := range wellknown.Links {
		if link.Href != "" {

			result := NewNodeInfo()
			txn := remote.Get(link.Href).
				With(client.options...).
				Result(&result)

			if err := txn.Send(); err != nil {
				derp.Report(derp.Wrap(err, location, "Error retrieving nodeinfo document"))
			} else {
				return result, nil
			}
		}
	}

	// If no valid links were found, return an error
	return NewNodeInfo(), derp.NotFoundError(location, "No valid nodeinfo links found")
}
