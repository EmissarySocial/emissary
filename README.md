<img src="https://emissary.dev/.templates/emissary_dev_homepage/resources/Emissary-Wordmark-Black-600.png" width="40%">

[![Go Reference](https://pkg.go.dev/badge/github.com/EmissarySocial/emissary.svg)](https://pkg.go.dev/github.com/EmissarySocial/emissary)
[![Build Status](https://img.shields.io/github/actions/workflow/status/EmissarySocial/emissary/go.yml?branch=main)](https://github.com/EmissarySocial/emissary/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/EmissarySocial/emissary?style=flat-square)](https://goreportcard.com/report/github.com/EmissarySocial/emissary)
[![Codecov](https://img.shields.io/codecov/c/github/EmissarySocial/emissary.svg?style=flat-square)](https://codecov.io/gh/EmissarySocial/emissary)


Emissary is **the Social Web Toolkit** -- a standalone Fediverse server designed for end users, app creators, and hosting admins — that gives everyone powerful new ways to join the social web.

## Why Emissary?

### Trustworthy Custom Applications

As a developer, Emissary empowers you to build custom, social applications in a simple, declarative, **low-code environment**.  Using only HTML templates and a JSON config file, you can create full-featured social apps that are easy to deploy and easy to maintain.

This is done by building action pipelines out of simple, composable steps, like: "show an edit form", "create a thumbnail", and "save the object".  Pipelines work alongside Emissary's built-in state machines and access permissions to form robust and secure applications that you and your end-users can trust.

Distribute your applications via Git and .zip files, comprising one or more  Each template is isolated from others, so bugs in one template won't bleed out into the rest of your site.  This should prevent the incompatibility, feature bloat, and bugginess that have plagued other plugin ecosystems.

### Multi-Network

Emissary use the [sherlock library](https://github.com/benpate/sherlock) to bridge across different federated protocols.  This includes ActivityPub, RSS+WebSub, and IndieWeb.  More are coming, to be added into the core system.  This means that applications you build on Emissary interact with the entire social web, and will grow as Emissary grows.

### Baked-In DevOps

Anyone should be able to stand up their own Emissary server.  But few people will.  Grandma probably won't.  For Emissary to be successful, hosting companies must be able to offer new Emissary accounts at the click of a button.

This means that Emissary must be an exemplary citizen in any DevOps workflow.  It should be as easy for an individual hobbyist to turn on a test server on a big hosting provider as it is for that big hosting provider to offer hundreds of thousands (or millions) of accounts to the general public at scale.

### High Performance

Emissary is built to be fast on any hardware.  Lightweight, cacheable templates work with the latest web performance techniques from [htmx.org](https://htmx.org) for a web application that always loads quickly and runs smoothly.

## Get Started

To get started, visit the [Emissary Developer Website](https://emissary.dev).  This resource is growing every day, and includes [a quickstart guide](https://emissary.dev/installation) along with detailed documentation on [how to configure Emissary](https://emissary.dev/configuring) for your own environment.

## Feature Checklist

There's a lot of work to do.  Check out the [project status page](https://emissary.dev/status) and the [Emissary kanban](https://trello.com/b/Ir9dDTdu/emissary-dev) for a peek at where we are right now.

## Tech Stack

Emissary is intended to be as easy to run and as scalable as possible.  It runs with a minimal set of dependencies, so you should be able to [install and run an Emissary server](https://emissary.dev/installation) in between lunch and tea time.

* [Go](https://go.dev)
* [Mongodb](https://mongodb.org)
* [HTMX](https://htmx.org) / [Hyperscript](https://hyperscript.org)
* That's it.  I'll worry about a cute acronym later.

Emissary also relies on a stack of custom libraries that make it go:

* [Hannibal](https://github.com/benpate/hannibal) - A robust, idiomatic ActivityPub interfaces in Go
* [Sherlock](https://github.com/benpate/sherlock) - Inspect data in ActivityPub/RSS/MicroFormats and more
* [Toot](https://github.com/benpate/toot) - Mastodon Server API
* [Rosetta](https://github.com/benpate/rosetta) - Data mapping and manipulations: schemas, conversions, etc 

A complete list can be found in the [Go module file](https://github.com/EmissarySocial/emissary/blob/main/go.mod).

### Everyone Welcome

I welcome your thoughts, ideas, feedback, criticisms, and mockery if it will help create a more realistic and workable way for people to use the Internet *as originally intended*.

Please try it out, get in touch, file a suggestion, report bugs "@" me, block me, whatever.  Just get involved and help make a difference.
