# Sherlock

<img alt="AI Generated Sherlock Holmes" src="https://github.com/EmissarySocial/emissary/raw/main/tools/sherlock/meta/logo.jpg" style="width:100%; display:block; margin-bottom:20px;">

Sherlock is an experimental library that inspects a URL for any and all available metadata, in whatever format the web page happens to support.  The goal is to 1) provide a single interface into all of the metadata that might be bundled into a web page, while 2) giving you control over additional services that might need to be called to retrieve that.

### Supported Formats

✅ [Microformats](https://microformats.org)

✅ [Open Graph](https://ogp.me)

### In Progress

🚧 [JSON-LD (Embedded)](https://json-ld.org/)

🚧 [JSON-LD (Linked)](https://json-ld.org/)

🚧 [Twitter Metadata](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)

🚧 [Microdata](https://html.spec.whatwg.org/multipage/microdata.html#microdata)

🚧 [RDFa](https://rdfa.info)

🚧 [oEmbed data provider](https://oembed.com)


### Using Sherlock
```go
// If you only have a URL, then pass it in to .Load()
result, err := sherlock.Load("https://my-url-here")

// If you have already downloaded a file, then pass it to .Parse()
result, err := sherlock.Parse("https://original-url", &bytes.Buffer)

// Individual functions search for data in specific formats
result := sherlock.NewPage()
err := sherlock.ParseMicroformats(*url.URL, io.Reader, &result)

```