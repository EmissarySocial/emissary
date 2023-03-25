package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// Block Actors by email address

func TestBlockFilter_ActorBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "actor@domain.com",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "actor@domain.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Actor(block, document))
}

func TestBlockFilter_ActorMute(t *testing.T) {

	block := model.Block{
		Trigger:  "actor@domain.com",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "someone@else.com",
		}, {
			EmailAddress: "actor@domain.com",
		}, {
			EmailAddress: "another@someone-else.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorMute, filter_Actor(block, document))
}

func TestBlockFilter_ActorAllow(t *testing.T) {

	block := model.Block{
		Trigger:  "nobody@domain.com",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "someone@else.com",
		}, {
			EmailAddress: "not-the-actor@youre-looking-for.com",
		}, {
			EmailAddress: "another@someone-else.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorAllow, filter_Actor(block, document))
}

// Block Actors by URL

func TestBlockFilter_ActorURLBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "https://domain.com/@actor",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			ProfileURL:   "https://domain.com/@actor",
			EmailAddress: "actor@domain.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Actor(block, document))
}

func TestBlockFilter_ActorURLMute(t *testing.T) {

	block := model.Block{
		Trigger:  "https://domain.com/@actor",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "someone@else.com",
		}, {
			EmailAddress: "https://domain.com/@actor",
		}, {
			EmailAddress: "another@someone-else.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorMute, filter_Actor(block, document))
}

func TestBlockFilter_ActorURLAllow(t *testing.T) {

	block := model.Block{
		Trigger:  "https://domain.com/@actor",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "someone@else.com",
			ProfileURL:   "http://someone-else.com/@someone-else",
		}, {
			EmailAddress: "not-the-actor@youre-looking-for.com",
			ProfileURL:   "https://domain.com/@not-the-actor-for-whom-youre-looking",
		}, {
			EmailAddress: "another@someone-else.com",
			ProfileURL:   "https://domain.com/@another",
		}},
	}

	require.Equal(t, model.BlockBehaviorAllow, filter_Actor(block, document))
}

// Block Content by Domain

func TestBlockFilter_DomainBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "domain.com",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		URL: "https://domain.com/some/path",
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Domain(block, document))
}

func TestBlockFilter_EmailDomainMute(t *testing.T) {

	block := model.Block{
		Trigger:  "domain.com",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "user@domain.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorMute, filter_Domain(block, document))
}

func TestBlockFilter_EmailDomainBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "domain.com",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			EmailAddress: "user@domain.com",
		}},
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Domain(block, document))
}

func TestBlockFilter_ProfileDomainMute(t *testing.T) {

	block := model.Block{
		Trigger:  "domain.com",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		AttributedTo: []model.PersonLink{{
			ProfileURL: "http://domain.com/@username",
		}},
	}

	require.Equal(t, model.BlockBehaviorMute, filter_Domain(block, document))
}

func TestBlockFilter_ContentLabelMute(t *testing.T) {

	block := model.Block{
		Trigger:  "#LMAO",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		Label: "This is super #LMAO",
	}

	require.Equal(t, model.BlockBehaviorMute, filter_Content(block, document, ""))
}

func TestBlockFilter_ContentSummaryBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "#YOLO",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		Label:   "This is super #LMAO",
		Summary: "You only live once #YOLO",
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Content(block, document, ""))
}

func TestBlockFilter_ContentHTMLBlock(t *testing.T) {

	block := model.Block{
		Trigger:  "#HODL",
		Behavior: model.BlockBehaviorBlock,
	}

	document := model.DocumentLink{
		Label:   "This is super #LMAO",
		Summary: "You only live once #YOLO",
	}

	require.Equal(t, model.BlockBehaviorBlock, filter_Content(block, document, "The all-important #HODL is back."))
}

func TestBlockFilter_ContentAllow(t *testing.T) {

	block := model.Block{
		Trigger:  "#MISSING",
		Behavior: model.BlockBehaviorMute,
	}

	document := model.DocumentLink{
		Label:   "This is super #LMAO",
		Summary: "You only live once #YOLO",
	}

	require.Equal(t, model.BlockBehaviorAllow, filter_Content(block, document, "The all-important #HODL is back."))
}
