package service

import (
	"strings"

	"github.com/benpate/ghost/model"
)

type FormService struct {
}

func (service FormService) Render(stream *model.Stream, transition *model.Transition) (string, string) {

	var head strings.Builder
	var foot strings.Builder

	url := "/" + stream.Token + "/forms/" + transition.ID

	head.WriteString(`<form method="post" encoding="" url="`)
	head.WriteString(url)
	head.WriteString(`" hx-post="`)
	head.WriteString(url)
	head.WriteString(`" hx-target="#stream">`)

	foot.WriteString(`<div><input type="submit" value="`)
	foot.WriteString(transition.Label)
	foot.WriteString(`"> <a href="/`)
	foot.WriteString(stream.Token)
	foot.WriteString(`" hx-boost="true">Cancel</a></div>`)
	foot.WriteString(`</form>`)

	return head.String(), foot.String()
}
