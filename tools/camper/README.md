# Camper 🏕️

<img src="https://github.com/EmissarySocial/emissary/raw/main/tools/camper/meta/logo.webp" style="width:100%; display:block; margin-bottom:20px;"  alt="Watercolor painting titled: A Tent in the Rockies (1916) by John Singer Sargent (American, 1856-1925)">

## Activity Intents for the New Social Web

Camper helps you to implement [FEP-3b86: Activity Intents](https://w3id.org/fep/3b86), which publishes the actions that a user can take from their home server and provides a consistent API for calling these intents from an external web page.

## Looking Up Intents

Camper makes it easy to look up the Activity Intent capabilities of a user's home server.  If the target user's server publishes its capabilities via the WebFinger standard, then Camper will use these links directly.

Otherwise, if the user's home server does not publish any activity intent links, then Camper will try to make a best guess based on the kind of Fediverse software they're using (provided by [NodeInfo 2.0](https://github.com/jhass/nodeinfo/blob/main/PROTOCOL.md)) and the template strings published by [Wladimir Palant](https://palant.info/2023/10/19/implementing-a-share-on-mastodon-button-for-a-blog/).

All of this logic is wrapped in a single system call so that your code stays clean.

``` go
// Create a new Camper client
client := camper.New( /* functional options here */ )

// Look up the template for this intent and username.
// If none is found, then urlTemplate will be empty.
urlTemplate := client.GetTemplate("create", "@username@server.social")

if urlTemplate != "" {
    // You're all clear, kid!
}
```

## Configuration Options

Each camper client can be configured using functional options.  You can apply these options to camper on initialization via functional options, or after the client has been created, using the `.With()` method

### `WithClient(*http.Client)`
This option allows you to specify a custom HTTP client to use for all camper calls.
For instance, you may want to use an HTTP client that caches responses from frequently
visited services.

``` go
// Create a custom HTTP client
customClient := &http.NewClient{}

// Apply the custom client to camper upon initialization
client := camper.New(camper.WithClient(customClient))

// Or apply afterwards using `With`
client.With(camper.NewClient(customClient))

// continue using the camper service...
```

## Transaction Types

Camper also includes predefined structs for each of the Activity Types defined in the [Activity Intents Proposal](https://w3id.org/fep/3b86).  You can use them in your code like this:

```go

// Import all of the data for a `Create` intent (from URL or Form data)
var txn camper.CreateIntent

if err := echo.Bind(&txn); err != nil {
    // handle errors...
}

// continue processing the Create intent
```

## Constants

Camper also includes constant values for each of the Activity Intents defined in the [Activity Intents Proposal](https://w3id.org/fep/3b86)


## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making it better, send in a pull request.  We're all in this together! 🏕️
