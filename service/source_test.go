package service

import (
	"testing"

	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestSourceQueries(t *testing.T) {

	service := getTestSourceService()
	object := service.New()

	it, err := service.ListByMethod(model.SourceMethodPoll)

	assert.Nil(t, err)

	for it.Next(object) {
		spew.Dump(object)
	}

	t.Fail()
}

func TestSourcePolling(t *testing.T) {

	service := getTestSourceService()

	err, contentErrors := service.Poll()

	spew.Dump(err)
	spew.Dump(contentErrors)
	spew.Dump(service.session)
	t.Fail()
}

func getTestSourceService() Source {

	f := getTestFactory()

	service := f.Source()

	{
		source := service.New()
		source.Adapter = model.SourceAdapterRSS
		source.Method = model.SourceMethodPoll
		source.Config = model.SourceConfig{
			"url": "https://appleinsider.com/rss/news",
		}

		service.Save(source, "Creating Test Data")
	}

	{
		source := service.New()
		source.Adapter = model.SourceAdapterRSS
		source.Method = model.SourceMethodPoll
		source.Config = model.SourceConfig{
			"url": "https://daringfireball.net/feeds/main",
		}

		service.Save(source, "Creating Test Data")
	}

	return service
}
