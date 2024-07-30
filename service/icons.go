package service

import (
	"io"
)

type Icons struct{}

func (service Icons) Get(name string) string {
	switch name {

	// App Actions and Behaviors
	case "add":
		return service.get("plus-lg")
	case "add-circle":
		return service.get("plus-circle")
	case "add-emoji":
		return service.get("emoji-smile")
	case "album":
		return service.get("cassette")
	case "album-fill":
		return service.get("cassette-fill")
	case "archive":
		return service.get("archive")
	case "archive-fill":
		return service.get("archive-fill")
	case "book":
		return service.get("book")
	case "book-fill":
		return service.get("book-fill")
	case "bookmark":
		return service.get("bookmark")
	case "bookmark-fill":
		return service.get("bookmark-fill")
	case "box":
		return service.get("box")
	case "box-fill":
		return service.get("box-fill")
	case "calendar":
		return service.get("calendar3")
	case "calendar-fill":
		return service.get("calendar3-week-fill")
	case "cancel":
		return service.get("x-lg")
	case "chat":
		return service.get("chat")
	case "chat-fill":
		return service.get("chat-fill")
	case "check-badge":
		return service.get("patch-check")
	case "check-badge-fill":
		return service.get("patch-check-fill")
	case "check-circle":
		return service.get("check-circle")
	case "check-circle-fill":
		return service.get("check-circle-fill")
	case "check-shield":
		return service.get("shield-check")
	case "check-shield-fill":
		return service.get("shield-check-fill")
	case "chevron-left":
		return service.get("chevron-left")
	case "chevron-right":
		return service.get("chevron-right")
	case "circle":
		return service.get("circle")
	case "circle-fill":
		return service.get("circle-fill")
	case "clipboard":
		return service.get("clipboard")
	case "clipboard-fill":
		return service.get("clipboard-fill")
	case "clock":
		return service.get("clock")
	case "clock-fill":
		return service.get("clock-fill")
	case "cloud":
		return service.get("cloud")
	case "cloud-fill":
		return service.get("cloud-fill")
	case "database":
		return service.get("database")
	case "database-fill":
		return service.get("database-fill")
	case "delete":
		return service.get("trash")
	case "delete-fill":
		return service.get("trash-fill")
	case "drag-handle":
		return service.get("grip-vertical")
	case "edit":
		return service.get("pencil-square")
	case "edit-fill":
		return service.get("pencil-square-fill")
	case "email":
		return service.get("envelope")
	case "email-fill":
		return service.get("envelope-fill")
	case "explicit":
		return service.get("explicit")
	case "explicit-fill":
		return service.get("explicit-fill")
	case "file":
		return service.get("file-earmark")
	case "file-fill":
		return service.get("file-earmark-fill")
	case "filter":
		return service.get("filter-circle")
	case "filter-fill":
		return service.get("filter-circle-fill")
	case "flag":
		return service.get("flag")
	case "flag-fill":
		return service.get("flag-fill")
	case "folder":
		return service.get("folder")
	case "folder-fill":
		return service.get("folder-fill")
	case "globe":
		return service.get("globe2")
	case "globe-fill":
		return service.get("globe2")
	case "grip-vertical":
		return service.get("grip-vertical")
	case "grip-horizontal":
		return service.get("grip-horizontal")
	case "hashtag":
		return service.get("hash")
	case "heart":
		return service.get("heart")
	case "heart-fill":
		return service.get("heart-fill")
	case "home":
		return service.get("house")
	case "home-fill":
		return service.get("house-fill")
	case "info":
		return service.get("info-circle")
	case "info-fill":
		return service.get("info-circle-fill")
	case "invisible":
		return service.get("eye-slash")
	case "invisible-fill":
		return service.get("eye-slash-fill")
	case "journal":
		return service.get("journal")
	case "key":
		return service.get("key")
	case "key-fill":
		return service.get("key-fill")
	case "link":
		return service.get("link-45deg")
	case "link-outbound":
		return service.get("box-arrow-up-right")
	case "location":
		return service.get("geo-alt")
	case "location-fill":
		return service.get("geo-alt-fill")
	case "lock":
		return service.get("lock")
	case "lock-fill":
		return service.get("lock-fill")
	case "loading":
		return service.get("arrow-clockwise")
	case "login":
		return service.get("box-arrow-in-right")
	case "megaphone":
		return service.get("megaphone")
	case "megaphone-fill":
		return service.get("megaphone-fill")
	case "mention":
		return service.get("at")
	case "more-horizontal":
		return service.get("three-dots")
	case "more-vertical":
		return service.get("three-dots-vertical")
	case "mute":
		return service.get("mic-mute")
	case "mute-fill":
		return service.get("mic-mute-fill")
	case "newspaper":
		return service.get("newspaper")
	case "none":
		return service.get("ban")
	case "pause":
		return service.get("pause")
	case "pause-fill":
		return service.get("pause-fill")
	case "person":
		return service.get("person")
	case "person-fill":
		return service.get("person-fill")
	case "people":
		return service.get("people")
	case "people-fill":
		return service.get("people-fill")
	case "play":
		return service.get("play")
	case "play-fill":
		return service.get("play-fill")
	case "reply":
		return service.get("reply")
	case "reply-fill":
		return service.get("reply-fill")
	case "repost":
		return service.get("repeat")
	case "rocket":
		return service.get("rocket-takeoff")
	case "rocket-fill":
		return service.get("rocket-takeoff-fill")
	case "rule":
		return service.get("funnel")
	case "rule-fill":
		return service.get("funnel-fill")
	case "save":
		return service.get("check-lg")
	case "search":
		return service.get("search")
	case "settings":
		return service.get("gear")
	case "settings-fill":
		return service.get("gear-fill")
	case "server":
		return service.get("hdd-stack")
	case "server-fill":
		return service.get("hdd-stack-fill")
	case "share":
		return service.get("arrow-up-right-square")
	case "share-fill":
		return service.get("arrow-up-right-square-fill")
	case "shield":
		return service.get("shield")
	case "shield-fill":
		return service.get("shield-fill")
	case "shield-lock":
		return service.get("shield-lock")
	case "skip-backward":
		return service.get("skip-backward")
	case "skip-backward-fill":
		return service.get("skip-backward-fill")
	case "skip-forward":
		return service.get("skip-forward")
	case "skip-forward-fill":
		return service.get("skip-forward-fill")
	case "star":
		return service.get("star")
	case "star-fill":
		return service.get("star-fill")
	case "template":
		return service.get("layout-text-sidebar-reverse")
	case "thumbs-down":
		return service.get("hand-thumbs-down")
	case "thumbs-down-fill":
		return service.get("hand-thumbs-down-fill")
	case "thumbs-up":
		return service.get("hand-thumbs-up")
	case "thumbs-up-fill":
		return service.get("hand-thumbs-up-fill")
	case "unlink":
		return service.get("link-45deg")
	case "upload":
		return service.get("upload")
	case "user":
		return service.get("person-circle")
	case "user-fill":
		return service.get("person-circle-fill")
	case "users":
		return service.get("people")
	case "users-fill":
		return service.get("people-fill")
	case "visible":
		return service.get("eye")
	case "visible-fill":
		return service.get("eye-fill")

		// Layouts
	case "layout-social":
		return service.get("list-ul")
	case "layout-social-fill":
		return service.get("list-ul")

	case "layout-chat":
		return service.get("chat-text")
	case "layout-chat-fill":
		return service.get("chat-text")

	case "layout-newspaper":
		return service.get("postcard")
	case "layout-newspaper-fill":
		return service.get("postcard")

	case "layout-magazine":
		return service.get("view-stacked")
	case "layout-magazine-fill":
		return service.get("view-stacked")

		// Services
	case "activitypub":
		// return service.get("globe2")
		return service.activityPub()
	case "activitypub-fill":
		return service.activityPub()
		// return service.get("globe2")
	case "facebook":
		return service.get("facebook")
	case "fediverse":
		return service.activityPub()
	case "github":
		return service.get("github")
	case "google":
		return service.get("google")
	case "json":
		return service.get("braces")
	case "json-fill":
		return service.get("braces")
	case "instagram":
		return service.get("instagram")
	case "twitter":
		return service.get("twitter")
	case "rss":
		return service.get("rss")
	case "rss-fill":
		return service.get("rss-fill")
	case "rss-cloud":
		return service.get("cloud-arrow-down")
	case "rss-cloud-fill":
		return service.get("cloud-arrow-down-fill")
	case "stripe":
		return service.get("credit-card")
	case "stripe-fill":
		return service.get("credit-card-fill")
	case "websub":
		return service.get("cloud-arrow-down")
	case "websub-fill":
		return service.get("cloud-arrow-down-fill")

	// Content Types
	case "article":
		return service.get("file-text")
	case "article-fill":
		return service.get("file-text-fill")
	case "block":
		return service.get("slash-circle")
	case "block-fill":
		return service.get("slash-circle-fill")
	case "collection":
		return service.get("view-stacked")
	case "forward":
		return service.get("forward")
	case "forward-fill":
		return service.get("forward-fill")
	case "html":
		return service.get("code-slash")
	case "html-fill":
		return service.get("code-slash")
	case "inbox":
		return service.get("inbox")
	case "inbox-fill":
		return service.get("inbox-fill")
	case "markdown":
		return service.get("markdown")
	case "markdown-fill":
		return service.get("markdown-fill")
	case "message":
		return service.get("chat-left-text")
	case "message-fill":
		return service.get("chat-left-text-fill")
	case "outbox":
		return service.get("envelope")
	case "outbox-fill":
		return service.get("envelope-fill")
	case "picture":
		return service.get("image")
	case "picture-fill":
		return service.get("image-fill")
	case "pictures":
		return service.get("images")
	case "shopping-cart":
		return service.get("cart")
	case "shopping-cart-fill":
		return service.get("cart-fill")
	case "video":
		return service.get("camera-video")
	case "video-fill":
		return service.get("camera-video-fill")
	}

	return service.get(name)
}

func (service Icons) get(name string) string {
	return `<i class="bi bi-` + name + `"></i>`
}

func (service Icons) Write(name string, writer io.Writer) {
	// Okay to ignore write error
	// nolint:errcheck
	writer.Write([]byte(service.Get(name)))
}

func (service Icons) activityPub() string {
	return `<svg viewBox="-10 -5 1034 1034" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em;">
<path fill="currentColor"
 d="M539 176q-32 0 -55 22t-25 55t20.5 58t56 27t58.5 -20.5t27 -56t-20.5 -59t-56.5 -26.5h-5zM452 271l-232 118q20 20 25 48l231 -118q-19 -20 -24 -48zM619 298q-13 25 -38 38l183 184q13 -25 39 -38zM477 320l-135 265l40 40l143 -280q-28 -5 -48 -25zM581 336
 q-22 11 -46 10l-8 -1l21 132l56 9zM155 370q-32 0 -55 22.5t-25 55t20.5 58t56.5 27t59 -21t26.5 -56t-21 -58.5t-55.5 -27h-6zM245 438q1 9 1 18q-1 19 -10 35l132 21l26 -50zM470 474l-26 51l311 49q-1 -8 -1 -17q1 -19 10 -36zM842 480q-32 1 -55 23t-24.5 55t21 58
 t56 27t58.5 -20.5t27 -56.5t-20.5 -59t-56.5 -27h-6zM236 493q-13 25 -39 38l210 210l51 -25zM196 531q-21 11 -44 10l-9 -1l40 256q21 -10 45 -9l8 1zM560 553l48 311q21 -10 44 -9l10 1l-46 -294zM755 576l-118 60l8 56l135 -68q-20 -20 -25 -48zM781 625l-119 231
 q28 5 48 25l119 -231q-28 -5 -48 -25zM306 654l-68 134q28 5 48 25l60 -119zM568 671l-281 143q19 20 24 48l265 -135zM513 771l-51 25l106 107q13 -25 39 -38zM222 795q-32 0 -55.5 22.5t-25 55t21 57.5t56 27t58.5 -20.5t27 -56t-20.5 -58.5t-56.5 -27h-5zM311 863
 q2 9 1 18q-1 19 -9 35l256 41q-1 -9 -1 -18q1 -18 10 -35zM646 863q-32 0 -55 22.5t-24.5 55t20.5 58t56 27t59 -21t27 -56t-20.5 -58.5t-56.5 -27h-6z" />
 </svg>`
}
