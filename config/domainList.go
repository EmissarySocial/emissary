package config

import (
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

type DomainList []Domain

/**************************
 * Path Interface
 **************************/

func (domainList DomainList) GetPath(name string) (interface{}, bool) {

	if name == "" {
		return domainList, true
	}

	head, tail := path.Split(name)
	index, err := path.Index(head, len(domainList))

	if err != nil {
		return nil, false
	}

	return domainList[index].GetPath(tail)
}

func (domainList *DomainList) SetPath(name string, value interface{}) error {

	head, tail := path.Split(name)
	index, err := path.Index(head, len(*domainList))

	if err != nil {
		return derp.Wrap(err, "whisper.config.DomainList.GetPath", "Invalid index", name)
	}

	return (*domainList)[index].SetPath(tail, value)
}
