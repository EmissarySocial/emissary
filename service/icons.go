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
	case "add-square":
		return service.get("plus-square")
	case "add-emoji":
		return service.get("emoji-smile")
	case "album":
		return service.get("cassette")
	case "album-fill":
		return service.get("cassette-fill")
	case "alert":
		return service.get("exclamation-triangle")
	case "alert-fill":
		return service.get("exclamation-triangle-fill")
	case "announce":
		return service.get("arrow-up-right-square")
	case "announce-fill":
		return service.get("arrow-up-right-square-fill")
	case "archive":
		return service.get("archive")
	case "archive-fill":
		return service.get("archive-fill")
	case "at":
		return service.get("at")
	case "at-fill":
		return service.get("at")
	case "bell":
		return service.get("bell")
	case "bell-fill":
		return service.get("bell-fill")
	case "braces":
		return service.get("braces")
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
	case "card":
		return service.get("postcard")
	case "card-fill":
		return service.get("postcard-fill")
	case "cassette":
		return service.get("cassette")
	case "cassette-fill":
		return service.get("cassette-fill")
	case "chat":
		return service.get("chat")
	case "chat-fill":
		return service.get("chat-fill")
	case "chat-square":
		return service.get("chat-square-dots")
	case "chat-square-fill":
		return service.get("chat-square-dots-fill")
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
	case "credit-card":
		return service.get("credit-card")
	case "credit-card-fill":
		return service.get("credit-card-fill")
	case "database":
		return service.get("database")
	case "database-fill":
		return service.get("database-fill")
	case "delete":
		return service.get("x-lg")
	case "delete-fill":
		return service.get("x-lg")
	case "dislike":
		return service.get("thumbs-down")
	case "dislike-fill":
		return service.get("thumbs-down-fill")
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
	case "focus":
		return service.get("crosshair")
	case "focus-fill":
		return service.get("crosshair2")
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
	case "help":
		return service.get("question-circle")
	case "help-fill":
		return service.get("question-circle-fill")
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
	case "like":
		return service.get("heart")
	case "like-fill":
		return service.get("heart-fill")
	case "link":
		return service.get("link-45deg")
	case "link-outbound":
		return service.get("box-arrow-up-right")
	case "list":
		return service.get("list")
	case "list-fill":
		return service.get("list")
	case "list-nested":
		return service.get("list-nested")
	case "list-nested-fill":
		return service.get("list-nested")
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
	case "menu":
		return service.get("list")
	case "more-horizontal":
		return service.get("three-dots")
	case "more-vertical":
		return service.get("three-dots-vertical")
	case "music":
		return service.get("music-note-beamed")
	case "mute":
		return service.get("mic-mute")
	case "mute-fill":
		return service.get("mic-mute-fill")
	case "new-window":
		return service.get("arrow-up-right-square")
	case "newspaper":
		return service.get("newspaper")
	case "none":
		return service.get("ban")
	case "note":
		return service.get("sticky")
	case "note-fill":
		return service.get("sticky-fill")
	case "notes":
		return service.get("stickies")
	case "notes-fill":
		return service.get("stickies-fill")
	case "pause":
		return service.get("pause")
	case "pause-fill":
		return service.get("pause-fill")
	case "person":
		return service.get("person")
	case "person-fill":
		return service.get("person-fill")
	case "person-card":
		return service.get("person-vcard")
	case "person-card-fill":
		return service.get("person-vcard-fill")
	case "people":
		return service.get("people")
	case "people-fill":
		return service.get("people-fill")
	case "play":
		return service.get("play")
	case "play-fill":
		return service.get("play-fill")
	case "reload":
		return service.get("arrow-repeat")
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
	case "search-fill":
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
	case "timeline":
		return service.get("clock")
	case "timeline-fill":
		return service.get("clock-fill")
	case "unlink":
		return service.get("link-45deg")
	case "upload":
		return service.get("cloud-arrow-up")
	case "upload-fill":
		return service.get("cloud-arrow-up-fill")
	case "user":
		return service.get("person-circle")
	case "user-fill":
		return service.get("person-circle")
	case "users":
		return service.get("people")
	case "users-fill":
		return service.get("people-fill")
	case "visible":
		return service.get("eye")
	case "visible-fill":
		return service.get("eye-fill")
	case "volume-up":
		return service.get("volume-up")
	case "volume-up-fill":
		return service.get("volume-up-fill")
	case "volume-down":
		return service.get("volume-down")
	case "volume-down-fill":
		return service.get("volume-down-fill")
	case "volume-mute":
		return service.get("volume-mute")
	case "volume-mute-fill":
		return service.get("volume-mute-fill")

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
		return service.activityPub()
	case "activitypub-fill":
		return service.activityPub()
	case "facebook":
		return service.get("facebook")
	case "fediverse":
		return service.fediverse()
	case "fediverse-fill":
		return service.fediverse()
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
	case "remove-square":
		return service.get("dash-square")
	case "rss":
		return service.get("rss")
	case "rss-fill":
		return service.get("rss-fill")
	case "rss-cloud":
		return service.get("cloud-arrow-down")
	case "rss-cloud-fill":
		return service.get("cloud-arrow-down-fill")
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
	case "code":
		return service.get("code-slash")
	case "code-fill":
		return service.get("code-slash")
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
	case "webhooks":
		return service.webhooks()
	case "video":
		return service.get("camera-video")
	case "video-fill":
		return service.get("camera-video-fill")

	// License Types
	case "copyright":
		return service.copyright()

	case "cc-by":
		return service.ccCC() + " " + service.ccBy()

	case "cc-by-sa":
		return service.ccCC() + " " + service.ccBy() + " " + service.ccSa()

	case "cc-by-nc":
		return service.ccCC() + " " + service.ccBy() + " " + service.ccNc()

	case "cc-by-nc-sa":
		return service.ccCC() + " " + service.ccBy() + " " + service.ccNc() + " " + service.ccSa()

	case "cc-by-nd":
		return service.ccCC() + " " + service.ccBy() + " " + service.ccNd()

	case "cc-by-nc-nd":
		return service.ccCC() + " " + service.ccBy() + " " + service.ccNc() + " " + service.ccNd()

	case "cc0":
		return service.cc0()

	case "public-domain":
		return service.publicDomain()

	// Services
	case "paypal", "PAYPAL":
		return service.get("paypal")

	case "stripe", "STRIPE":
		return service.get("stripe")

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

func (service Icons) copyright() string {
	return "©"
}

func (service Icons) ccCC() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.441-3.5c8.951,0,16.572,3.125,22.857,9.372c3.008,3.009,5.295,6.448,6.857,10.314
c1.561,3.867,2.344,7.971,2.344,12.314c0,4.381-0.773,8.486-2.314,12.313c-1.543,3.828-3.82,7.21-6.828,10.143
c-3.123,3.085-6.666,5.448-10.629,7.086c-3.961,1.638-8.057,2.457-12.285,2.457s-8.276-0.808-12.143-2.429
c-3.866-1.618-7.333-3.961-10.4-7.027c-3.067-3.066-5.4-6.524-7-10.372S5.5,32.767,5.5,28.5c0-4.229,0.809-8.295,2.428-12.2
c1.619-3.905,3.972-7.4,7.057-10.486C21.08-0.394,28.565-3.5,37.441-3.5z M37.557,2.272c-7.314,0-13.467,2.553-18.458,7.657
c-2.515,2.553-4.448,5.419-5.8,8.6c-1.354,3.181-2.029,6.505-2.029,9.972c0,3.429,0.675,6.734,2.029,9.913
c1.353,3.183,3.285,6.021,5.8,8.516c2.514,2.496,5.351,4.399,8.515,5.715c3.161,1.314,6.476,1.971,9.943,1.971
c3.428,0,6.75-0.665,9.973-1.999c3.219-1.335,6.121-3.257,8.713-5.771c4.99-4.876,7.484-10.99,7.484-18.344
c0-3.543-0.648-6.895-1.943-10.057c-1.293-3.162-3.18-5.98-5.654-8.458C50.984,4.844,44.795,2.272,37.557,2.272z M37.156,23.187
l-4.287,2.229c-0.458-0.951-1.019-1.619-1.685-2c-0.667-0.38-1.286-0.571-1.858-0.571c-2.856,0-4.286,1.885-4.286,5.657
c0,1.714,0.362,3.084,1.085,4.113c0.724,1.029,1.791,1.544,3.201,1.544c1.867,0,3.181-0.915,3.944-2.743l3.942,2
c-0.838,1.563-2,2.791-3.486,3.686c-1.484,0.896-3.123,1.343-4.914,1.343c-2.857,0-5.163-0.875-6.915-2.629
c-1.752-1.752-2.628-4.19-2.628-7.313c0-3.048,0.886-5.466,2.657-7.257c1.771-1.79,4.009-2.686,6.715-2.686
C32.604,18.558,35.441,20.101,37.156,23.187z M55.613,23.187l-4.229,2.229c-0.457-0.951-1.02-1.619-1.686-2
c-0.668-0.38-1.307-0.571-1.914-0.571c-2.857,0-4.287,1.885-4.287,5.657c0,1.714,0.363,3.084,1.086,4.113
c0.723,1.029,1.789,1.544,3.201,1.544c1.865,0,3.18-0.915,3.941-2.743l4,2c-0.875,1.563-2.057,2.791-3.541,3.686
c-1.486,0.896-3.105,1.343-4.857,1.343c-2.896,0-5.209-0.875-6.941-2.629c-1.736-1.752-2.602-4.19-2.602-7.313
c0-3.048,0.885-5.466,2.658-7.257c1.77-1.79,4.008-2.686,6.713-2.686C51.117,18.558,53.938,20.101,55.613,23.187z"/>
</svg>
`
}

func (service Icons) ccBy() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.443-3.5c8.988,0,16.57,3.085,22.742,9.257C66.393,11.967,69.5,19.548,69.5,28.5c0,8.991-3.049,16.476-9.145,22.456
C53.879,57.319,46.242,60.5,37.443,60.5c-8.649,0-16.153-3.144-22.514-9.43C8.644,44.784,5.5,37.262,5.5,28.5
c0-8.761,3.144-16.342,9.429-22.742C21.101-0.415,28.604-3.5,37.443-3.5z M37.557,2.272c-7.276,0-13.428,2.553-18.457,7.657
c-5.22,5.334-7.829,11.525-7.829,18.572c0,7.086,2.59,13.22,7.77,18.398c5.181,5.182,11.352,7.771,18.514,7.771
c7.123,0,13.334-2.607,18.629-7.828c5.029-4.838,7.543-10.952,7.543-18.343c0-7.276-2.553-13.465-7.656-18.571
C50.967,4.824,44.795,2.272,37.557,2.272z M46.129,20.557v13.085h-3.656v15.542h-9.944V33.643h-3.656V20.557
c0-0.572,0.2-1.057,0.599-1.457c0.401-0.399,0.887-0.6,1.457-0.6h13.144c0.533,0,1.01,0.2,1.428,0.6
C45.918,19.5,46.129,19.986,46.129,20.557z M33.042,12.329c0-3.008,1.485-4.514,4.458-4.514s4.457,1.504,4.457,4.514
c0,2.971-1.486,4.457-4.457,4.457S33.042,15.3,33.042,12.329z"/>
</svg>
`
}

func (service Icons) ccSa() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.443-3.5c8.951,0,16.531,3.105,22.742,9.315C66.393,11.987,69.5,19.548,69.5,28.5c0,8.954-3.049,16.457-9.145,22.514
C53.918,57.338,46.279,60.5,37.443,60.5c-8.649,0-16.153-3.143-22.514-9.429C8.644,44.786,5.5,37.264,5.5,28.501
c0-8.723,3.144-16.285,9.429-22.685C21.138-0.395,28.643-3.5,37.443-3.5z M37.557,2.272c-7.276,0-13.428,2.572-18.457,7.715
c-5.22,5.296-7.829,11.467-7.829,18.513c0,7.125,2.59,13.257,7.77,18.4c5.181,5.182,11.352,7.771,18.514,7.771
c7.123,0,13.334-2.609,18.629-7.828c5.029-4.876,7.543-10.99,7.543-18.343c0-7.313-2.553-13.485-7.656-18.513
C51.004,4.842,44.832,2.272,37.557,2.272z M23.271,23.985c0.609-3.924,2.189-6.962,4.742-9.114
c2.552-2.152,5.656-3.228,9.314-3.228c5.027,0,9.029,1.62,12,4.856c2.971,3.238,4.457,7.391,4.457,12.457
c0,4.915-1.543,9-4.627,12.256c-3.088,3.256-7.086,4.886-12.002,4.886c-3.619,0-6.743-1.085-9.371-3.257
c-2.629-2.172-4.209-5.257-4.743-9.257H31.1c0.19,3.886,2.533,5.829,7.029,5.829c2.246,0,4.057-0.972,5.428-2.914
c1.373-1.942,2.059-4.534,2.059-7.771c0-3.391-0.629-5.971-1.885-7.743c-1.258-1.771-3.066-2.657-5.43-2.657
c-4.268,0-6.667,1.885-7.2,5.656h2.343l-6.342,6.343l-6.343-6.343L23.271,23.985L23.271,23.985z"/>
</svg>
`
}

func (service Icons) ccNc() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.442-3.5c8.99,0,16.571,3.085,22.743,9.256C66.393,11.928,69.5,19.509,69.5,28.5c0,8.992-3.048,16.476-9.145,22.458
C53.88,57.32,46.241,60.5,37.442,60.5c-8.686,0-16.19-3.162-22.513-9.485C8.644,44.728,5.5,37.225,5.5,28.5
c0-8.762,3.144-16.343,9.429-22.743C21.1-0.414,28.604-3.5,37.442-3.5z M12.7,19.872c-0.952,2.628-1.429,5.505-1.429,8.629
c0,7.086,2.59,13.22,7.77,18.4c5.219,5.144,11.391,7.715,18.514,7.715c7.201,0,13.409-2.608,18.63-7.829
c1.867-1.79,3.332-3.657,4.398-5.602l-12.056-5.371c-0.421,2.02-1.439,3.667-3.057,4.942c-1.622,1.276-3.535,2.011-5.744,2.2
v4.915h-3.714v-4.915c-3.543-0.036-6.782-1.312-9.714-3.827l4.4-4.457c2.094,1.942,4.476,2.913,7.143,2.913
c1.104,0,2.048-0.246,2.83-0.743c0.78-0.494,1.172-1.312,1.172-2.457c0-0.801-0.287-1.448-0.858-1.943l-3.085-1.315l-3.771-1.715
l-5.086-2.229L12.7,19.872z M37.557,2.214c-7.276,0-13.428,2.571-18.457,7.714c-1.258,1.258-2.439,2.686-3.543,4.287L27.786,19.7
c0.533-1.676,1.542-3.019,3.029-4.028c1.484-1.009,3.218-1.571,5.2-1.686V9.071h3.715v4.915c2.934,0.153,5.6,1.143,8,2.971
l-4.172,4.286c-1.793-1.257-3.619-1.885-5.486-1.885c-0.991,0-1.876,0.191-2.656,0.571c-0.781,0.381-1.172,1.029-1.172,1.943
c0,0.267,0.095,0.533,0.285,0.8l4.057,1.83l2.8,1.257l5.144,2.285l16.397,7.314c0.535-2.248,0.801-4.533,0.801-6.857
c0-7.353-2.552-13.543-7.656-18.573C51.005,4.785,44.831,2.214,37.557,2.214z"/>
</svg>
`
}

func (service Icons) ccNd() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"  version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.443-3.5c8.951,0,16.531,3.105,22.742,9.315C66.393,11.987,69.5,19.548,69.5,28.5c0,8.954-3.049,16.457-9.145,22.514
C53.918,57.338,46.279,60.5,37.443,60.5c-8.649,0-16.153-3.143-22.514-9.43C8.644,44.786,5.5,37.264,5.5,28.501
c0-8.723,3.144-16.285,9.429-22.685C21.138-0.395,28.643-3.5,37.443-3.5z M37.557,2.272c-7.276,0-13.428,2.572-18.457,7.715
c-5.22,5.296-7.829,11.467-7.829,18.513c0,7.125,2.59,13.257,7.77,18.4c5.181,5.182,11.352,7.771,18.514,7.771
c7.123,0,13.334-2.608,18.629-7.828c5.029-4.876,7.543-10.989,7.543-18.343c0-7.313-2.553-13.485-7.656-18.513
C51.004,4.842,44.832,2.272,37.557,2.272z M49.615,20.956v5.486H26.358v-5.486H49.615z M49.615,31.243v5.483H26.358v-5.483H49.615
z"/>
</svg>
`
}

func (service Icons) cc0() string {
	return `<svg viewBox="-0.5 0.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"  version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="31.325" cy="32.873" r="30.096"/>
<path fill="currentColor" id="text2809_1_" d="M31.5,14.08c-10.565,0-13.222,9.969-13.222,18.42c0,8.452,2.656,18.42,13.222,18.42
c10.564,0,13.221-9.968,13.221-18.42C44.721,24.049,42.064,14.08,31.5,14.08z M31.5,21.026c0.429,0,0.82,0.066,1.188,0.157
c0.761,0.656,1.133,1.561,0.403,2.823l-7.036,12.93c-0.216-1.636-0.247-3.24-0.247-4.437C25.808,28.777,26.066,21.026,31.5,21.026z
M36.766,26.987c0.373,1.984,0.426,4.056,0.426,5.513c0,3.723-0.258,11.475-5.69,11.475c-0.428,0-0.822-0.045-1.188-0.136
c-0.07-0.021-0.134-0.043-0.202-0.067c-0.112-0.032-0.23-0.068-0.336-0.11c-1.21-0.515-1.972-1.446-0.874-3.093L36.766,26.987z"/>
<path id="path2815_1_" d="M31.433,0.5c-8.877,0-16.359,3.09-22.454,9.3c-3.087,3.087-5.443,6.607-7.082,10.532
C0.297,24.219-0.5,28.271-0.5,32.5c0,4.268,0.797,8.32,2.397,12.168c1.6,3.85,3.921,7.312,6.969,10.396
c3.085,3.049,6.549,5.399,10.398,7.037c3.886,1.602,7.939,2.398,12.169,2.398c4.229,0,8.34-0.826,12.303-2.465
c3.962-1.639,7.496-3.994,10.621-7.081c3.011-2.933,5.289-6.297,6.812-10.106C62.73,41,63.5,36.883,63.5,32.5
c0-4.343-0.77-8.454-2.33-12.303c-1.562-3.885-3.848-7.32-6.857-10.33C48.025,3.619,40.385,0.5,31.433,0.5z M31.567,6.259
c7.238,0,13.412,2.566,18.554,7.709c2.477,2.477,4.375,5.31,5.67,8.471c1.296,3.162,1.949,6.518,1.949,10.061
c0,7.354-2.516,13.454-7.506,18.33c-2.592,2.516-5.502,4.447-8.74,5.781c-3.2,1.334-6.498,1.994-9.927,1.994
c-3.468,0-6.788-0.653-9.949-1.948c-3.163-1.334-6.001-3.238-8.516-5.716c-2.515-2.514-4.455-5.353-5.826-8.516
c-1.333-3.199-2.017-6.498-2.017-9.927c0-3.467,0.684-6.787,2.017-9.949c1.371-3.2,3.312-6.074,5.826-8.628
C18.092,8.818,24.252,6.259,31.567,6.259z"/>
</svg>
`
}

func (service Icons) publicDomain() string {
	return `<svg viewBox="5.5 -3.5 64 64" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em; vertical-align:text-bottom;">
<circle stroke="currentColor" fill-opacity="0" cx="37.47" cy="28.736" r="29.471"/>
<path fill="currentColor" d="M37.443-3.5c8.988,0,16.58,3.096,22.77,9.286C66.404,11.976,69.5,19.547,69.5,28.5c0,8.954-3.049,16.437-9.145,22.456
C53.918,57.319,46.279,60.5,37.443,60.5c-8.687,0-16.182-3.144-22.486-9.43C8.651,44.784,5.5,37.262,5.5,28.5
c0-8.761,3.144-16.342,9.429-22.742C21.101-0.415,28.604-3.5,37.443-3.5z M37.529,2.272c-7.257,0-13.401,2.553-18.428,7.657
c-5.22,5.296-7.829,11.486-7.829,18.572s2.59,13.22,7.771,18.398c5.181,5.182,11.352,7.771,18.514,7.771
c7.162,0,13.371-2.607,18.629-7.828c5.029-4.877,7.543-10.991,7.543-18.343c0-7.314-2.553-13.504-7.656-18.571
C50.967,4.824,44.785,2.272,37.529,2.272z M22.471,37.186V19.472h8.8c4.342,0,6.514,1.999,6.514,6
c0,0.686-0.105,1.342-0.314,1.972c-0.209,0.629-0.572,1.256-1.086,1.886c-0.514,0.629-1.285,1.143-2.314,1.543
c-1.028,0.399-2.247,0.6-3.656,0.6h-3.486v5.714H22.471z M26.871,22.785v5.372h3.771c0.914,0,1.6-0.258,2.058-0.772
c0.458-0.513,0.687-1.152,0.687-1.915c0-1.79-0.953-2.686-2.858-2.686h-3.657V22.785z M38.984,37.186V19.472h6.859
c2.818,0,5.027,0.724,6.629,2.171c1.598,1.448,2.398,3.677,2.398,6.686c0,3.01-0.801,5.24-2.398,6.686
c-1.602,1.447-3.811,2.171-6.629,2.171H38.984z M43.387,23.186v10.287h2.57c1.562,0,2.695-0.466,3.4-1.401
c0.705-0.933,1.057-2.179,1.057-3.742c0-1.562-0.352-2.809-1.057-3.743c-0.705-0.933-1.857-1.399-3.457-1.399L43.387,23.186
L43.387,23.186z"/>
</svg>
`
}

func (service Icons) fediverse() string {
	return `<svg viewBox="-10 -5 1034 1034" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em;">
<path fill="currentColor"
d="M539 176q-32 0 -55 22t-25 55t20.5 58t56 27t58.5 -20.5t27 -56t-20.5 -59t-56.5 -26.5h-5zM452 271l-232 118q20 20 25 48l231 -118q-19 -20 -24 -48zM619 298q-13 25 -38 38l183 184q13 -25 39 -38zM477 320l-135 265l40 40l143 -280q-28 -5 -48 -25zM581 336
q-22 11 -46 10l-8 -1l21 132l56 9zM155 370q-32 0 -55 22.5t-25 55t20.5 58t56.5 27t59 -21t26.5 -56t-21 -58.5t-55.5 -27h-6zM245 438q1 9 1 18q-1 19 -10 35l132 21l26 -50zM470 474l-26 51l311 49q-1 -8 -1 -17q1 -19 10 -36zM842 480q-32 1 -55 23t-24.5 55t21 58
t56 27t58.5 -20.5t27 -56.5t-20.5 -59t-56.5 -27h-6zM236 493q-13 25 -39 38l210 210l51 -25zM196 531q-21 11 -44 10l-9 -1l40 256q21 -10 45 -9l8 1zM560 553l48 311q21 -10 44 -9l10 1l-46 -294zM755 576l-118 60l8 56l135 -68q-20 -20 -25 -48zM781 625l-119 231
q28 5 48 25l119 -231q-28 -5 -48 -25zM306 654l-68 134q28 5 48 25l60 -119zM568 671l-281 143q19 20 24 48l265 -135zM513 771l-51 25l106 107q13 -25 39 -38zM222 795q-32 0 -55.5 22.5t-25 55t21 57.5t56 27t58.5 -20.5t27 -56t-20.5 -58.5t-56.5 -27h-5zM311 863
q2 9 1 18q-1 19 -9 35l256 41q-1 -9 -1 -18q1 -18 10 -35zM646 863q-32 0 -55 22.5t-24.5 55t20.5 58t56 27t59 -21t27 -56t-20.5 -58.5t-56.5 -27h-6z" />
</svg>`
}

func (service Icons) activityPub() string {
	return `<svg viewBox="0 0 1034 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xml:space="preserve" xmlns:serif="http://www.serif.com/" style="height:1em;">
<g transform="matrix(1,0,0,1,17,-128)">
<path fill="currentColor" d="M457,341L25,590L25,690L370,491L370,890L457,939L457,341ZM543,341L543,441L889,640L543,840L543,939L975,690L975,590L543,341ZM543,541L543,740L716,640L543,541ZM284,640L111,740L284,840L284,640Z"/>
</g>
</svg>
`
}

func (service Icons) webhooks() string {
	return `<svg viewBox="-10 -5 1034 1034" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" style="height:1em;">
<path fill="currentColor"
d="M482 226h-1l-10 2q-33 4 -64.5 18.5t-55.5 38.5q-41 37 -57 91q-9 30 -8 63t12 63q17 45 52 78l13 12l-83 135q-26 -1 -45 7q-30 13 -45 40q-7 15 -9 31t2 32q8 30 33 48q15 10 33 14.5t36 2t34.5 -12.5t27.5 -25q12 -17 14.5 -39t-5.5 -41q-1 -5 -7 -14l-3 -6l118 -192
q6 -9 8 -14l-10 -3q-9 -2 -13 -4q-23 -10 -41.5 -27.5t-28.5 -39.5q-17 -36 -9 -75q4 -23 17 -43t31 -34q37 -27 82 -27q27 -1 52.5 9.5t44.5 30.5q17 16 26.5 38.5t10.5 45.5q0 17 -6 42l70 19l8 1q14 -43 7 -86q-4 -33 -19.5 -63.5t-39.5 -53.5q-42 -42 -103 -56
q-6 -2 -18 -4l-14 -2h-37zM500 350q-17 0 -34 7t-30.5 20.5t-19.5 31.5q-8 20 -4 44q3 18 14 34t28 25q24 15 56 13q3 4 5 8l112 191q3 6 6 9q27 -26 58.5 -35.5t65 -3.5t58.5 26q32 25 43.5 61.5t0.5 73.5q-8 28 -28.5 50t-48.5 33q-31 13 -66.5 8.5t-63.5 -24.5
q-4 -3 -13 -10l-5 -6q-4 3 -11 10l-47 46q23 23 52 38.5t61 21.5l22 4h39l28 -5q64 -13 110 -60q22 -22 36.5 -50.5t19.5 -59.5q5 -36 -2 -71.5t-25 -64.5t-44 -51t-57 -35q-34 -14 -70.5 -16t-71.5 7l-17 5l-81 -137q13 -19 16 -37q5 -32 -13 -60q-16 -25 -44 -35
q-17 -6 -35 -6zM218 614q-58 13 -100 53q-47 44 -61 105l-4 24v37l2 11q2 13 4 20q7 31 24.5 59t42.5 49q50 41 115 49q38 4 76 -4.5t70 -28.5q53 -34 78 -91q7 -17 14 -45q6 -1 18 0l125 2q14 0 20 1q11 20 25 31t31.5 16t35.5 4q28 -3 50 -20q27 -21 32 -54
q2 -17 -1.5 -33t-13.5 -30q-16 -22 -41 -32q-17 -7 -35.5 -6.5t-35.5 7.5q-28 12 -43 37l-3 6q-14 0 -42 -1l-113 -1q-15 -1 -43 -1l-50 -1l3 17q8 43 -13 81q-14 27 -40 45t-57 22q-35 6 -70 -7.5t-57 -42.5q-28 -35 -27 -79q1 -37 23 -69q13 -19 32 -32t41 -19l9 -3z" />
</svg>`
}
