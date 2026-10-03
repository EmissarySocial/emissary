package mastodon

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// mentionsForStream lists the accounts a Stream mentions, as the API reports them. A mention with no
// usable profile address is completed from known(acct), and left out when that knows nothing.
func mentionsForStream(stream *model.Stream, known func(acct string) string) []object.StatusMention {

	tags := model.TagsOfType(stream.Tags, vocab.LinkTypeMention)
	result := make([]object.StatusMention, 0, len(tags))

	parsed, _ := url.Parse(stream.URL)

	for _, tag := range tags {

		if tag.Name == "" {
			continue
		}

		// An unresolved tag carries "" or a "-" placeholder, never a usable address
		href := tag.Href

		if !tag.IsResolved() {
			href = known(tag.Name)
		}

		if href == "" {
			continue
		}

		username, _, _ := strings.Cut(tag.Name, "@")

		result = append(result, object.StatusMention{
			ID:       mentionAccountID(parsed, href),
			Username: username,
			URL:      href,
			Acct:     tag.Name,
		})
	}

	return result
}

// mentionAccountID returns the account ID for a mentioned profile URL: a User's own ID when the
// profile is on this server, otherwise the encoded form used for every other account.
func mentionAccountID(origin *url.URL, profileURL string) string {

	if origin != nil && origin.Host != "" {

		if local, found := strings.CutPrefix(profileURL, origin.Scheme+"://"+origin.Host+"/@"); found {

			if _, err := primitive.ObjectIDFromHex(local); err == nil {
				return local
			}
		}
	}

	return model.EncodeRemoteAccountID(profileURL)
}

// markMentionLinks wraps each mention in status HTML in Mastodon's mention markup, so clients treat it as a
// profile link. Text already inside a link is left alone; content with no mentions is returned as it came.
func markMentionLinks(content string, mentions []object.StatusMention) string {

	if content == "" || len(mentions) == 0 {
		return content
	}

	nodes, err := html.ParseFragment(strings.NewReader(content), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})

	if err != nil {
		return content
	}

	// Top-level nodes come back with no parent, so hold them under one while their text is replaced
	root := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

	for _, node := range nodes {
		root.AppendChild(node)
	}

	if !linkMentions(root, mentions) {
		return content
	}

	var builder strings.Builder

	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&builder, child); err != nil {
			return content
		}
	}

	return builder.String()
}

// linkMentions walks a node tree, replacing mention text with links, and reports whether it replaced any.
func linkMentions(node *html.Node, mentions []object.StatusMention) bool {

	if node.Type == html.ElementNode && node.DataAtom == atom.A {
		return false
	}

	changed := false

	// Collect the children first, because replacing a text node edits the list being walked
	children := []*html.Node{}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		children = append(children, child)
	}

	for _, child := range children {

		if child.Type == html.TextNode {

			if replaceMentionText(child, mentions) {
				changed = true
			}

			continue
		}

		if linkMentions(child, mentions) {
			changed = true
		}
	}

	return changed
}

// replaceMentionText splits one text node around the mentions it contains.
func replaceMentionText(node *html.Node, mentions []object.StatusMention) bool {

	text := node.Data

	var pieces []*html.Node
	position := 0

	// Split the text around each mention, matching without regard to case
	for lower := strings.ToLower(text); position < len(text); {

		start, length, mention := nextMention(text, lower, position, mentions)

		if start < 0 {
			break
		}

		if start > position {
			pieces = append(pieces, &html.Node{Type: html.TextNode, Data: text[position:start]})
		}

		pieces = append(pieces, mentionNode(mention))
		position = start + length
	}

	// Leave the text as it is when it has no mentions
	if len(pieces) == 0 {
		return false
	}

	// Keep any text after the last mention
	if position < len(text) {
		pieces = append(pieces, &html.Node{Type: html.TextNode, Data: text[position:]})
	}

	// Put the pieces in place of the original text
	for _, piece := range pieces {
		node.Parent.InsertBefore(piece, node)
	}

	node.Parent.RemoveChild(node)
	return true
}

// nextMention finds the earliest mention in text at or after position, as a whole "@name".
func nextMention(text string, lower string, position int, mentions []object.StatusMention) (int, int, object.StatusMention) {

	bestStart := -1
	bestLength := 0
	best := object.StatusMention{}

	for _, mention := range mentions {

		for needle, offset := "@"+strings.ToLower(mention.Acct), position; offset <= len(lower)-len(needle); {

			found := strings.Index(lower[offset:], needle)

			if found < 0 {
				break
			}

			start := offset + found

			if end := start + len(needle); mentionBoundary(text, start, end) {

				if bestStart < 0 || start < bestStart || (start == bestStart && len(needle) > bestLength) {
					bestStart, bestLength, best = start, len(needle), mention
				}

				break
			}

			offset = start + 1
		}
	}

	return bestStart, bestLength, best
}

// mentionBoundary confirms a match is a whole mention: not the tail of a longer word or address.
func mentionBoundary(text string, start int, end int) bool {

	if start > 0 {
		if before, _ := utf8.DecodeLastRuneInString(text[:start]); unicode.IsLetter(before) || unicode.IsDigit(before) || before == '_' {
			return false
		}
	}

	if end < len(text) {
		after, _ := utf8.DecodeRuneInString(text[end:])

		if unicode.IsLetter(after) || unicode.IsDigit(after) || after == '_' || after == '-' || after == '@' {
			return false
		}

		// A "." only ends a sentence; one followed by more letters means the domain goes on
		if after == '.' && end+1 < len(text) {
			if next, _ := utf8.DecodeRuneInString(text[end+1:]); unicode.IsLetter(next) || unicode.IsDigit(next) {
				return false
			}
		}
	}

	return true
}

// mentionNode builds Mastodon's markup for one mention: a link around the short "@username" form.
func mentionNode(mention object.StatusMention) *html.Node {

	card := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{{Key: "class", Val: "h-card"}}}

	link := &html.Node{Type: html.ElementNode, DataAtom: atom.A, Data: "a", Attr: []html.Attribute{
		{Key: "href", Val: mention.URL},
		{Key: "class", Val: "u-url mention"},
	}}

	name := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span"}
	name.AppendChild(&html.Node{Type: html.TextNode, Data: mention.Username})

	link.AppendChild(&html.Node{Type: html.TextNode, Data: "@"})
	link.AppendChild(name)
	card.AppendChild(link)

	return card
}

// mentionsForDocument lists the accounts a remote post mentions. A mention's URL is the
// link the post's own content uses for it, because clients match a tapped link against it exactly.
func mentionsForDocument(document streams.Document) []object.StatusMention {

	links := contentLinks(document.Content())
	result := make([]object.StatusMention, 0)

	for tag := range document.Tag().Range() {

		if tag.Type() != vocab.LinkTypeMention {
			continue
		}

		actorURL := tag.Href()
		name := strings.TrimPrefix(tag.Name(), "@")

		if actorURL == "" || name == "" {
			continue
		}

		// Remote mentions often omit the domain, so take it from the actor's own address
		username, _, hasDomain := strings.Cut(name, "@")
		acct := name

		if parsed, err := url.Parse(actorURL); err == nil && !hasDomain {
			acct = username + "@" + parsed.Host
		}

		// Prefer the link written in the content, falling back to the actor URL
		linkURL := actorURL

		if found, ok := links["@"+strings.ToLower(username)]; ok {
			linkURL = found
		}

		result = append(result, object.StatusMention{
			ID:       model.EncodeRemoteAccountID(actorURL),
			Username: username,
			URL:      linkURL,
			Acct:     acct,
		})
	}

	return result
}

// contentLinks maps the lowercased text of each link in HTML content (without any domain
// part) to its href, so a mention can be matched to the link that displays it.
func contentLinks(content string) map[string]string {

	result := make(map[string]string)

	// Parse the content
	root, err := html.Parse(strings.NewReader(content))

	if err != nil {
		return result
	}

	// Visit every link, noting where each one points
	var walk func(node *html.Node)

	walk = func(node *html.Node) {

		if node.Type == html.ElementNode && node.DataAtom == atom.A {

			// Read the link's address
			href := ""

			for _, attr := range node.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
			}

			// Read the link's text, keeping only "@username" from a "@username@domain" display
			text := strings.ToLower(strings.TrimSpace(nodeText(node)))

			if strings.HasPrefix(text, "@") {
				name, _, _ := strings.Cut(text[1:], "@")
				text = "@" + name
			}

			if href != "" && text != "" {
				result[text] = href
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)

	return result
}

// nodeText returns all of the text inside a node.
func nodeText(node *html.Node) string {

	if node.Type == html.TextNode {
		return node.Data
	}

	var builder strings.Builder

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(nodeText(child))
	}

	return builder.String()
}
