package service

/*
func TestAtomLinks(t *testing.T) {

	var body bytes.Buffer

	body.WriteString(`<?xml version="1.0" encoding="UTF-8" ?>
	<?xml-stylesheet type="text/xsl" href="https://websub.rocks/assets/atom.xsl" ?>
	<feed xmlns="http://www.w3.org/2005/Atom">
	  <title>WebSub Rocks! Atom Feed Discovery</title>
	  <link href="https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo" rel="self" type="application/atom+xml" />
	  <link href="https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo/hub" rel="hub" />
	  <id>https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo</id>
	  <publishUrl>https://websub.rocks/subscriber/102/lw2ssiXKSWWlqvc92Wdo/publish</publishUrl>
	  <updated>2022-12-29T01:03:08+00:00</updated>

	  <subtitle>This Atom feed has a stylesheet that will make it look like the websub.rocks site. If you are seeing this message, your browser doesn't support XSLT. To add a new post to this feed, follow this link https://websub.rocks/subscriber/102/lw2ssiXKSWWlqvc92Wdo/publish</subtitle>


	  <entry>
		<id>https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-0</id>
		<title></title>
		<published>2022-12-29T01:03:08+00:00</published>
		<content type="html"><![CDATA[The more you care, the stronger you can be.]]></content>
		<link rel="alternate" type="text/html" href="https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-0" />
		<author>
		  <name>Jim Rohn</name>
		</author>
	  </entry>

	  <entry>
		<id>https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-1</id>
		<title></title>
		<published>2022-12-29T01:03:08+00:00</published>
		<content type="html"><![CDATA[The key to growth is the introduction of higher dimensions of consciousness into our awareness.]]></content>
		<link rel="alternate" type="text/html" href="https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-1" />
		<author>
		  <name>Lao Tzu </name>
		</author>
	  </entry>


	  <entry>
		<id>https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-2</id>
		<title></title>
		<published>2022-12-29T01:03:08+00:00</published>
		<content type="html"><![CDATA[However many holy words you read, however many you speak, what good will they do you if you do not act on upon them?]]></content>
		<link rel="alternate" type="text/html" href="https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo#quote-2" />
		<author>
		  <name>Buddha </name>
		</author>
	  </entry>

	  </feed>`)

	result := discoverLinks_RSS(nil, &body)

	require.Equal(t, 2, len(result))

	require.Equal(t, "self", result[0].RelationType)
	require.Equal(t, "https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo", result[0].Href)
	require.Equal(t, "application/atom+xml", result[0].MediaType)

	require.Equal(t, "hub", result[1].RelationType)
	require.Equal(t, "https://websub.rocks/blog/102/lw2ssiXKSWWlqvc92Wdo/hub", result[1].Href)
}

func TestRSSLinks(t *testing.T) {
	var body bytes.Buffer

	body.WriteString(`<?xml version="1.0" encoding="UTF-8" ?>
	<?xml-stylesheet type="text/xsl" href="https://websub.rocks/assets/rss.xsl" ?>
	<rss version="2.0"
	  xmlns:content="http://purl.org/rss/1.0/modules/content/"
	  xmlns:atom="http://www.w3.org/2005/Atom"
	  >
	<channel>
	  <title>WebSub Rocks! RSS Feed Discovery</title>
	  <atom:link href="https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu" rel="self" type="application/rss+xml" />
	  <atom:link href="https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu/hub" rel="hub" />
	  <link>https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu</link>
	  <publishUrl>https://websub.rocks/subscriber/103/JXKfevIPFu6PFRdErTIu/publish</publishUrl>
	  <lastBuildDate>Thu, 29 Dec 2022 07:03:14 +0000</lastBuildDate>
	  <language>en-US</language>

	  <description>This RSS feed has a stylesheet that will make it look like the websub.rocks site. If you are seeing this message, your browser doesn't support XSLT. To add a new post to this feed, follow this link https://websub.rocks/subscriber/103/JXKfevIPFu6PFRdErTIu/publish</description>


		<item>
		  <title></title>
		  <pubDate>Thu, 29 Dec 2022 07:03:14 +0000</pubDate>
		  <guid isPermaLink="true">https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-0</guid>
		  <description><![CDATA[Designing something is like having a baby. Asking me to try another design once I&#8217;ve birthed something amazing is like asking me to put the baby back in the womb and try again. That never works out for anyone.]]></description>
		  <link>https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-0</link>
		  <author>Chad McMillan</author>
		</item>


		<item>
		  <title></title>
		  <pubDate>Thu, 29 Dec 2022 07:03:14 +0000</pubDate>
		  <guid isPermaLink="true">https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-1</guid>
		  <description><![CDATA[As if a device can function if it has no style. As if a device can be called stylish that does not function superbly&#8230; yes, beauty matters. Boy, does it matter. It is not surface, it is not an extra, it is the thing itself.]]></description>
		  <link>https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-1</link>
		  <author>Stephen Fry</author>
		</item>


		<item>
		  <title></title>
		  <pubDate>Thu, 29 Dec 2022 07:03:14 +0000</pubDate>
		  <guid isPermaLink="true">https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-2</guid>
		  <description><![CDATA[Your task is not to seek love, but merely to seek and find all the barriers within yourself that you have built against it.]]></description>
		  <link>https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu#quote-2</link>
		  <author>Rumi</author>
		</item>

	  </channel>
	</rss>`)

	result := discoverLinks_RSS(nil, &body)

	require.Equal(t, 2, len(result))
	require.Equal(t, "self", result[0].RelationType)
	require.Equal(t, "https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu", result[0].Href)
	require.Equal(t, "application/rss+xml", result[0].MediaType)

	require.Equal(t, "hub", result[1].RelationType)
	require.Equal(t, "https://websub.rocks/blog/103/JXKfevIPFu6PFRdErTIu/hub", result[1].Href)
}
*/
