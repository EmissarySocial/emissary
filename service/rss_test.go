package service

import (
	"testing"

	"github.com/benpate/data/expression"
	"github.com/davecgh/go-spew/spew"
)

func TestRss(t *testing.T) {

	streamService := getTestStreamService()
	factory := streamService.factory

	rss := factory.RSS()

	feed, err := rss.Feed(expression.All())

	spew.Dump(err)
	spew.Dump(feed)

	t.Fail()
}
