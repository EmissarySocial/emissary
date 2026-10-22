package mastodon

import (
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/benpate/toot/object"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// markHashtagLinks rewrites the anchors in status HTML whose text is one of the status's
// hashtags ("#name") to that tag's URL, with Mastodon's hashtag markup
// (class="mention hashtag" rel="tag").
//
// RULE: anchors are matched by their text, not their old href, since Emissary's own hashtag
// URLs have varied across versions and Templates.
func markHashtagLinks(content string, tags []object.StatusTag) string {

	if content == "" || len(tags) == 0 {
		return content
	}

	tagURLs := make(map[string]string, len(tags))

	for _, tag := range tags {
		tagURLs[strings.ToLower(tag.Name)] = tag.URL
	}

	nodes, err := html.ParseFragment(strings.NewReader(content), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})

	if err != nil {
		return content
	}

	changed := false

	for _, node := range nodes {
		if markHashtagAnchors(node, tagURLs) {
			changed = true
		}
	}

	if !changed {
		return content
	}

	var builder strings.Builder

	for _, node := range nodes {
		if err := html.Render(&builder, node); err != nil {
			return content
		}
	}

	return builder.String()
}

// apiHashtagURL returns the URL Mastodon clients expect for a hashtag: "<origin>/tags/<name>".
func apiHashtagURL(origin string, name string) string {
	return strings.TrimSuffix(origin, "/") + "/tags/" + url.PathEscape(name)
}

// apiHashtags converts a status's hashtags to Mastodon-shaped URLs. A tag that already has
// the right shape (any Mastodon server's own) is left alone.
func apiHashtags(tags []object.StatusTag) []object.StatusTag {

	result := make([]object.StatusTag, 0, len(tags))

	for _, tag := range tags {

		parsed, err := url.Parse(tag.URL)

		if err == nil && parsed.Host != "" && !(strings.EqualFold(path.Base(parsed.Path), tag.Name) && slices.Contains(strings.Split(parsed.Path, "/"), "tags")) {
			tag.URL = apiHashtagURL(parsed.Scheme+"://"+parsed.Host, tag.Name)
		}

		result = append(result, tag)
	}

	return result
}

// markHashtagAnchors walks a node tree, fixing each hashtag anchor. It returns TRUE if it changed any.
func markHashtagAnchors(node *html.Node, tagURLs map[string]string) bool {

	changed := false

	if node.Type == html.ElementNode && node.DataAtom == atom.A {

		text := strings.TrimSpace(anchorText(node))

		if name, isHashtag := strings.CutPrefix(text, "#"); isHashtag {

			if replacement, found := tagURLs[strings.ToLower(name)]; found && replacement != "" {

				href, _ := anchorAttribute(node, "href")
				class, _ := anchorAttribute(node, "class")

				if href != replacement || !slices.Contains(strings.Fields(class), "hashtag") {
					setAnchorAttribute(node, "href", replacement)
					setAnchorAttribute(node, "class", "mention hashtag")
					setAnchorAttribute(node, "rel", "tag")
					changed = true
				}
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if markHashtagAnchors(child, tagURLs) {
			changed = true
		}
	}

	return changed
}

// anchorText returns the visible text inside a node.
func anchorText(node *html.Node) string {

	if node.Type == html.TextNode {
		return node.Data
	}

	var builder strings.Builder

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(anchorText(child))
	}

	return builder.String()
}

// anchorAttribute returns the value of a node's named attribute.
func anchorAttribute(node *html.Node, name string) (string, bool) {

	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val, true
		}
	}

	return "", false
}

// setAnchorAttribute sets a node's named attribute, replacing any existing value.
func setAnchorAttribute(node *html.Node, name string, value string) {

	for index := range node.Attr {
		if node.Attr[index].Key == name {
			node.Attr[index].Val = value
			return
		}
	}

	node.Attr = append(node.Attr, html.Attribute{Key: name, Val: value})
}
