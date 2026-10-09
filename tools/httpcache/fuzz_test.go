package httpcache

import (
	"net/url"
	"testing"
)

/******************************************
 * Decoder robustness
 *
 * A cache record is bytes that came from a remote server and were
 * then stored. These targets assert only that reading one back
 * never panics: a corrupt entry must read as a MISS.
 ******************************************/

// FuzzGetMetadata confirms that a corrupt metadata record never panics. The record is a
// URL-encoded query string, so invalid escapes are the interesting case.
func FuzzGetMetadata(f *testing.F) {

	f.Add("")
	f.Add("ETag=%22abc%22&Vary=Accept")
	f.Add("%zz")
	f.Add("Vary=" + url.QueryEscape("Accept, Accept-Language, Accept-Encoding"))
	f.Add("&&&===")

	f.Fuzz(func(t *testing.T, record string) {

		cache, adapter := newTestCache()
		adapter["https://example.com/"+metadataMarker] = record

		// A corrupt record must be a miss, never a panic and never a partial value
		if metadata, ok := cache.getMetadata("https://example.com/"); !ok {
			if metadata != nil {
				t.Error("a miss must return nil metadata")
			}
		}
	})
}

// FuzzGetResponse confirms that a corrupt stored response never panics. http.ReadResponse
// is a full HTTP parser, fed here with arbitrary bytes.
func FuzzGetResponse(f *testing.F) {

	f.Add("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nhi")
	f.Add("HTTP/1.1 200 OK\r\n\r\n")
	f.Add("")
	f.Add("not an http response")
	f.Add("HTTP/1.1 200 OK\r\nContent-Length: 999999\r\n\r\nshort")
	f.Add("HTTP/1.1 \r\n\r\n")

	f.Fuzz(func(t *testing.T, record string) {

		cache, adapter := newTestCache()
		request := newTestRequest(t, "https://example.com/page")

		adapter["https://example.com/page"+metadataMarker] = ""
		adapter["https://example.com/page"+headSeparator+cache.getVariesValues(request, url.Values{})] = record

		if response, ok := cache.getResponse(request); ok && response == nil {
			t.Error("a hit must return a non-nil response")
		}
	})
}

// FuzzGetVariesValues confirms that an arbitrary Vary header never panics while building the
// cache-key fingerprint, and that the result is always a parseable query string.
func FuzzGetVariesValues(f *testing.F) {

	f.Add("Accept", "text/html")
	f.Add("", "")
	f.Add("Accept, Accept-Language", "en-US")
	f.Add(",,,", "x")
	f.Add("Accept;q=0.9", "\x00\x01")

	f.Fuzz(func(t *testing.T, vary string, headerValue string) {

		cache, _ := newTestCache()

		request := newTestRequest(t, "https://example.com/")
		request.Header.Set("Accept", headerValue)

		fingerprint := cache.getVariesValues(request, url.Values{"Vary": []string{vary}})

		// The fingerprint becomes part of a cache key, so it must always round-trip
		if _, err := url.ParseQuery(fingerprint); err != nil {
			t.Errorf("fingerprint %q is not a parseable query string: %v", fingerprint, err)
		}
	})
}
