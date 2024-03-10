package striputm

import "net/url"

var knownCodes = KnownCodes()

// https://orionfeedback.org/d/4375-remove-trackers-from-copied-urls
// https://urlclean.com
func StripFromURL(href *url.URL) {

	qs := href.Query()
	for _, code := range knownCodes {
		qs.Del(code)
	}
	href.RawQuery = qs.Encode()
}

// https://github.com/jparise/chrome-utm-stripper
// https://www.bleepingcomputer.com/news/security/new-firefox-privacy-feature-strips-urls-of-tracking-parameters/
// https://docs.clearurls.xyz/1.26.1/
func KnownCodes() []string {
	return []string{
		"fbclid",           // Facebook
		"mc_eid",           // Facebook
		"ad_id",            // Facebook
		"adset_id",         // Facebook
		"campaign_id",      // Facebook
		"ad_name",          // Facebook
		"adset_name",       // Facebook
		"campaign_name",    // Facebook
		"placement",        // Facebook
		"site_source_name", // Facebook

		"gclid",           // Google
		"utm_source",      // Google - https://en.wikipedia.org/wiki/UTM_parameters
		"utm_medium",      // Google
		"utm_term",        // Google
		"utm_campaign",    // Google
		"utm_content",     // Google
		"utm_cid",         // Google
		"utm_reader",      // Google
		"utm_referrer",    // Google
		"utm_name",        // Google
		"utm_social",      // Google
		"utm_social-type", // Google
		"stm_source",      // Google (Alt?) - https://en.wikipedia.org/wiki/UTM_parameters
		"stm_medium",      // Google (Alt?)
		"stm_term",        // Google (Alt?)
		"stm_campaign",    // Google (Alt?)
		"stm_content",     // Google (Alt?)
		"stm_cid",         // Google (Alt?)
		"stm_reader",      // Google (Alt?)
		"stm_referrer",    // Google (Alt?)
		"stm_name",        // Google (Alt?)
		"stm_social",      // Google (Alt?)
		"stm_social-type", // Google (Alt?)

		" __s",           // Drip
		"_hsenc",         // HubSpot
		"_hsmi",          // HubSpot
		"igshid",         // Instagram
		"utm_klaviyo_id", // Klaviyo
		"mc_cid",         // MailChimp
		"mc_eid",         // MainChimp
		"mkt_tok",        // Marketo
		"cvid",           // MSN/Bing
		"oicd",           // MSN/Bing
		"oly_anon_id",    // Olytics
		"oly_enc_id",     // Olytics
		"otc",            // Olytics
		"vero_id",        // Vero
		"wickedid",       // Wicked Reports
		"soc_src",        // Yahoo
		"soc_trk",        // Yahoo
		"_openstat",      // Yandex
		"yclid",          // Yandex
		"ICID",           // Other?
		"rb_clickid",     // Other?
	}
}
