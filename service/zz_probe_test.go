package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

func TestProbeBisect(t *testing.T) {
	const themeDir = "../_embed/templates/theme-default"
	definition, _ := os.ReadFile(themeDir + "/theme.hjson")
	theme := model.NewTheme("default", nil)
	require.NoError(t, hjson.Unmarshal(definition, &theme))
	engine := goeditorjs.NewHTMLEngine()
	engine.RegisterBlockHandlers(&goeditorjs.HeaderHandler{}, &goeditorjs.ParagraphHandler{})
	contentService := NewContent(engine)
	themeService := NewTheme(nil, &contentService, nil)
	themeService.setStartupContent(&theme, os.DirFS(themeDir+"/content"))

	ss := (&Stream{}).Schema()

	// grab the "about" markdown content (production-identical rendering)
	var c model.Content
	for _, data := range theme.StartupStreams {
		if data.GetString("token") == "about" {
			c = data["content"].(model.Content)
		}
	}
	t.Logf("about raw=%q", c.Raw[:min(120,len(c.Raw))])
	t.Logf("about html=%q", c.HTML[:min(200,len(c.HTML))])

	try := func(name string, content model.Content) {
		s := model.NewStream()
		s.TemplateID = "article-markdown"
		s.Token = "about"
		s.Content = content
		if _, err := ss.Validate(&s); err != nil {
			t.Logf("%-24s -> ERR: %s", name, derp.RootCause(err))
		} else {
			t.Logf("%-24s -> OK", name)
		}
	}

	try("full", c)
	try("format-only", model.Content{Format: c.Format})
	try("format+raw", model.Content{Format: c.Format, Raw: c.Raw})
	try("format+html", model.Content{Format: c.Format, HTML: c.HTML})
	try("empty", model.Content{})
	try("format=MARKDOWN", model.Content{Format: "MARKDOWN"})
	try("raw-only(noformat)", model.Content{Raw: c.Raw})
	try("html-only(noformat)", model.Content{HTML: c.HTML})
}
func min(a,b int) int { if a<b {return a}; return b }
