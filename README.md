<img src="https://emissary.social/63a8916bccc34c36f1f55e4d/attachments/648a403a01d6b7eb886a3b3c" width="40%">

[![Go Reference](https://pkg.go.dev/badge/github.com/EmissarySocial/emissary.svg)](https://pkg.go.dev/github.com/EmissarySocial/emissary)
[![Build Status](https://img.shields.io/github/actions/workflow/status/EmissarySocial/emissary/go.yml?branch=main)](https://github.com/EmissarySocial/emissary/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/EmissarySocial/emissary?style=flat-square)](https://goreportcard.com/report/github.com/EmissarySocial/emissary)
[![Codecov](https://img.shields.io/codecov/c/github/EmissarySocial/emissary.svg?style=flat-square)](https://codecov.io/gh/EmissarySocial/emissary)


Emissary is an open-source project to make a different kind of distributed social medium that is friendly, safe, and welcoming to everyone.

Emissary will give each person a customizable, private space [in the Fediverse](#enter-the-fediverse) where they can create, share, and collaborate with the groups of people who matter most to them, both big and small.

## Why Emissary?

### Human-centric Design

Above all, Emissary is built to be easy enough for regular people to use.  End-users deserve a simple, streamlined way to find, post, share, and respond to everything on the web.

So, Grandma should be able to open her iPad, look through the latest baby photos that her kids have posted, and reply with a thought, a smiley, or a picture from her garden.  

### Designer/Developer Nirvana

While Emissary works "out of the box" every application template can be customized to fit your specific needs.

For designers and developers, Emissary is a **low-code environment** that offers a rich array of pre-built tools like state machines, validators, and draft versioning, and more.  Just wire together the pre-tested steps you need to make your templates shine.

And, there's no central database, so anyone can post a new feature and anyone can import customized templates straight into their Emissary instance.  

### Safely Customizable

But wide-ranging customization leads to [the WordPress pitfall](#the-wordpress-pitfall).  To avoid this, customized templates in Emissary are not written in a general-purpose language like PHP.  

Instead, Emissary templates wire together a wide range of well-tested actions, such as "display this object using this HTML template" or "save a version of this object to the database."  

Each template is isolated from others, so bugs in one template won't bleed out into the rest of your site.  This should prevent the incompatibility, feature bloat, and bugginess that have plagued the WordPress plugin ecosystem.

Most importantly, because of Emissary's low-code architecture, you'll be able to trust that new features will work alongside existing ones without the hassle.

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

Emissary is intended to be as easy to run and as scalable as possible.  Therefore, it is built on solid open-source foundations.

* [Go](https://go.dev)
* [Mongodb](https://mongodb.org)
* [HTMX](https://htmx.org) / [Hyperscript](https://hyperscript.org)
* That's it.  I'll worry about a cute acronym later.

A list of other open-source libraries can be found in the [Go module file](https://github.com/EmissarySocial/emissary/blob/main/go.mod).

## The Elephant in the <s>Room</s> Internet

It's not really a secret anymore.  At its core, our current social media platforms are breaking the Internet and breaking society at large.  Online services that survive on advertising revenue foster extremism in exchange for "engagement" in the same way that news networks turn to click-bait and rage-based "commentary" instead of actual journalism.  

Unfortunately, moderation is a band-aid at best, and more often an excuse to further fan the flames of hateful rhetoric.  Even the most well-intentioned moderation (from humans or AIs) can only cover up this problem instead of addressing its root cause, which is the misaligned goals of advertising revenue (engagement) vs. quality of life and social discourse.  There must be a better way.

<a id="enter-the-fediverse"></a>
### Enter the Fediverse

This, of course, is not new.  [The Fediverse](https://fediverse.party) and [IndieWeb](https://indieweb.org) are filled with fantastic and robust open-source solutions to this problem.

But they haven't taken off.

The real reasons will be debated for generations.  Emissary is built on the theory that (like much open-source software) existing distributed social media apps are aimed too narrowly at techies and not at the non-techie majority of people.  The technical barriers to entry are too high for average people and the network lock-in effects of existing social media are enough to keep the majority of people in place.

<a id="identity-and-trust"></a>
### Identity and Trust (OpSec)

Of course, bad actors will find ways to abuse any system we create.  So the system needs to at least level the playing field against trolls, disinformation warriors, and the like.  Emissary will address this by 1) not rewarding viral and "engaging" content, and 2) by making blocks more powerful than likes, allowing your trusted friends to help prevent the spread of hateful content.

Of course, closed groups will still develop their own information silos.  There's no way around that.  But blocks can act like a vaccine to innoculate the news feeds of people on the extremist fringes, helping to prevent them from falling into (or pull people out of) extremist ideologies.

<a id="all-about-the-money"></a>
### All About The Money

There's one more thorn in the side of open source social media, which is that there's no easy way to make money on the Fediverse that doesn't fall back on advertising, ultimately falling back into the same trap as centralized social media.  

Fortunately, a few closed-source services have already figured this out.  Wildly successful companies like Discord and Dropbox use "freemium" pricing to make their services available to everyone for free, then rake in huge profits from the subset of people who will pay more for extra features and capacity.

So, any solution to the distributed social-media dilemma must make space for somebody (probably hosting and design companies) to earn enough money to convince their shareholders to enter and continue in the business.

<a id="the-wordpress-pitfall"></a>
### The WordPress Pitfall

This, too, is not new.  WordPress is a hugely successful piece of software that powers something like 37% of all websites on the Internet.  A phenomenal achievement.

But there is so much complexity baked into the WordPress ecosystem that launching a website requires a professional designer at best and a pact with a Lovecraftian deity at worst.

A real solution should be as easy to turn on as signing up for Facebook or Twitter, and as easy to customize as picking a feature set from a restaurant menu.

**For Site Owners**: Any feature or template can be added to an Emissary instance simply by adding its Git address into the configuration tool.  When you pick something from the worldwide menu of Emissary features, it's instantly added to all of your sites.

**For Designers**: Emissary is a low-code environment, where template designers can wire together pre-built actions to create custom templates with sophisticated behaviors.

<a id="redemption-arc"></a>
### Redemption Arc

This is all a lot, to say that my best take on social media is to make a solution that:

1. is free and open-source for everyone to use
1. plugs into existing federated APIs in an easy, human-centric way
1. is free from surveillance capitalism and all its horrors
1. supports freemium business models where honest people can earn honest money.
1. strikes a balance between customization and stability

<a id="everyone-welcome"></a>
### Everyone Welcome

While this is as far as I've gotten, I'm certain I've missed something.  I welcome your thoughts, ideas, feedback, criticisms, and mockery if it will help create a more realistic and workable solution to "The Elephant in the Internet."  

Please try it out, get in touch, file a suggestion, report bugs "@" me, block me, whatever.  Just get involved and help make a difference.
