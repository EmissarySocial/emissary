package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestTextToHTML covers paragraphs, line breaks, escaping, and empty text
func TestTextToHTML(t *testing.T) {

	require.Equal(t, "<p>hello</p>", textToHTML("hello"))
	require.Equal(t, "<p>one<br>two</p>", textToHTML("one\ntwo"))
	require.Equal(t, "<p>first</p><p>second</p>", textToHTML("first\n\nsecond"))
	require.Equal(t, "<p>first</p><p>second</p>", textToHTML("first\r\n\r\n\r\nsecond\n"))
	require.Equal(t, "<p>&lt;b&gt;not bold&lt;/b&gt; &amp; more</p>", textToHTML("<b>not bold</b> & more"))
	require.Equal(t, "", textToHTML("  \n "))
}

// TestContent_TextFormat confirms TEXT content keeps the author's text as its source and renders safe HTML
func TestContent_TextFormat(t *testing.T) {

	service := Content{}
	content := service.New(model.ContentFormatText, "@bob hi\n\n<script>alert(1)</script>")

	require.Equal(t, model.ContentFormatText, content.Format)
	require.Equal(t, "@bob hi\n\n<script>alert(1)</script>", content.Raw, "the source is what the author typed")
	require.Equal(t, "<p>@bob hi</p><p>&lt;script&gt;alert(1)&lt;/script&gt;</p>", content.HTML)
}
