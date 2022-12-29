package service

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestRSSLinks(t *testing.T) {

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

	spew.Dump(discoverLinks_RSS(nil, &body))

}
