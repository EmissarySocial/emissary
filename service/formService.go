package service

import (
	"strings"

	"github.com/benpate/ghost/model"
)

type FormService struct {
}

func (service FormService) Render(stream *model.Stream, transitionID string, transition *model.Transition) (string, string) {

	var head strings.Builder
	var foot strings.Builder

	url := "/" + stream.Token + "/forms/" + transitionID

	head.WriteString(`<form method="post" url="`)
	head.WriteString(url)
	head.WriteString(`" hx-post="`)
	head.WriteString(url)
	head.WriteString(`" hx-target="#stream">`)

	foot.WriteString(`<div><input type="submit" class="uk-button uk-button-primary" value="`)
	foot.WriteString(transition.Label)
	foot.WriteString(`"> <a href="/`)
	foot.WriteString(stream.Token)
	foot.WriteString(`" hx-boost="true" class="uk-button uk-button-default">Cancel</a></div>`)
	foot.WriteString(`</form>`)

	return head.String(), foot.String()
}
