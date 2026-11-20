package mastodon

import (
	"testing"

	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// TestFirstCardLink confirms the card comes from the first ordinary web link, as Mastodon picks it
func TestFirstCardLink(t *testing.T) {

	status := func(content string) *object.Status {
		return &object.Status{
			URL:     "https://remote.example/@alice/1",
			URI:     "https://remote.example/users/alice/statuses/1",
			Content: content,
		}
	}

	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"plain link", `<p>Read <a href="https://news.example/story">this</a></p>`, "https://news.example/story"},
		{"first of two", `<p><a href="https://one.example/">one</a> <a href="https://two.example/">two</a></p>`, "https://one.example/"},
		{"Mastodon's split link text", `<p><a href="https://news.example/story" rel="nofollow noopener"><span class="invisible">https://</span><span>news.example/story</span></a></p>`, "https://news.example/story"},
		{"mention skipped", `<p><a href="https://remote.example/@bob" class="u-url mention">@<span>bob</span></a> <a href="https://news.example/">news</a></p>`, "https://news.example/"},
		{"hashtag skipped", `<p><a href="https://remote.example/tags/go" class="mention hashtag" rel="tag">#<span>go</span></a></p>`, ""},
		{"unmarked hashtag skipped", `<p><a href="https://remote.example/tags/go">#go</a></p>`, ""},
		{"rel tag skipped", `<p><a href="https://remote.example/tags/go" rel="tag">go</a></p>`, ""},
		{"link to the post itself", `<p><a href="https://remote.example/@alice/1">permalink</a></p>`, ""},
		{"not a web link", `<p><a href="mailto:alice@remote.example">mail</a> <a href="javascript:alert(1)">x</a></p>`, ""},
		{"relative link", `<p><a href="/about">about</a></p>`, ""},
		{"no links", `<p>Just words</p>`, ""},
		{"empty", ``, ""},
	}

	for _, test := range cases {
		require.Equal(t, test.want, firstCardLink(status(test.content)), test.name)
	}
}
