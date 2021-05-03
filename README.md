# Ghost 👻

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/benpate/ghost)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/ghost?style=flat-square)](https://goreportcard.com/report/github.com/benpate/ghost)
[![Build Status](http://img.shields.io/travis/benpate/ghost.svg?style=flat-square)](https://travis-ci.com/benpate/ghost)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/ghost.svg?style=flat-square)](https://codecov.io/gh/benpate/ghost)
![Version](https://img.shields.io/github/v/release/benpate/ghost?include_prereleases&style=flat-square&color=brightgreen)


#### This is a work-in-progress that is *guaranteed to NOT work* for anyone, in any capacity, at any time.  Do not use this code in production, in development, or even in theory, because it's all wrong and may never be corrected, maintained, or supported.

# Why Ghost?
In the early 2000's, social media got many things right: it's easy to publish, easy to subscribe, and share.
But things went off the rails, and our current social media landscape is far from perfect.  Now, social media is synonymous with disinformation, invasive tracking, and lack of control.  

Malicious algorithms with global reach churn human beings into ad revenue.  It didn't have to be this way.  Ghost is a social CMS with a small reach, allowing you to stay connected to the most important people -- the friends and family in your inner circle.

# What is It?

Ghost is a new kind of decentralized, private media server that will connect people instead of driving them apart, and will return power and privacy to users and content creators.

## Decentralized
When completed, this will be a new kind of personal media server, meant to be an open, [federated](https://en.wikipedia.org/wiki/Fediverse) replacement for many of the closed, centralized services that we all use today.  

## Private
Ghost belongs to the users, not the service providers.  There is no tracking built in to ghost, and we will work to keep it that way.  Strong access controls make your content easy to share, and easy to manage.

## Social
It will work with customizable templates that will replicate many of the social media services out there: posts, comments, images, videos, real time communications and more.

## Real-Time
Ghost will support several real-time messaging interfaces, pushing live content to your community instantly.  

# Technology Goals
Ghost must be extremely service-provider-friendly: easy to virtualize, provision, and deploy. To make this easy , it should be self-contained, with as few dependencies as possible.  Here are a few of the interfaces that I'd like to implement:

* [RSS](https://en.wikipedia.org/wiki/RSS) / [JSON Feed](https://jsonfeed.org)
* [ActivityPub](https://activitypub.rocks) / [OpenSocial](https://www.getopensocial.com) / [Diaspora](https://diasporafoundation.org)
* [OAuth](https://www.google.com/url?sa=t&rct=j&q=&esrc=s&source=web&cd=&ved=2ahUKEwjByq6-_K3wAhVeIDQIHdMuCmsQFjAQegQIBBAD&url=https%3A%2F%2Foauth.net%2F&usg=AOvVaw3GDFM0pkIJMe4FATEf5VSd)
* [WebRTC](https://webrtc.org)
* [Server Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events)
* [oEmbed](https://oembed.com)

## Toolkit
| Tool | Info|
|---|---|
| [Go](https://golang.org) | Single file executable server, compiled for every OS and architecture |
| [Mongodb](https://mongodb.org) | Database server (possibly swappable) |
| [htmx](https://htmx.org) & [hyperscript](https://hyperscript.org)  | Interactive HTML content
| [ckEditor](https://ckeditor.com/ckeditor-5/) | Rich content editing
| ??? | Various local and online file storage systems


# Pitch In!
There's a lot to do, and I'd love to have your help.  If you're interested in building the federated web, please get in touch or submit a pull request. 👻

