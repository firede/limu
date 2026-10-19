package main

import (
	"net/url"
	"strings"
)

// siteURL is the address of the playground on the website, which a link copied on a
// desktop opens.
const siteURL = "https://limu.firede.com/playground"

// maxLinkHash is the longest fragment a link may have, as the website's playgroundHref
// allows: a link longer than that is not one to share.
const maxLinkHash = 32_000

// shareLink is a link to page that opens src: the text in the URL's fragment, encoded as a
// query is, as the website's links to the playground and Pele's have it. It reports
// whether the text is short enough for one.
func shareLink(page, src string) (string, bool) {
	hash := url.Values{"source": {src}}.Encode()
	if len(hash) > maxLinkHash {
		return "", false
	}
	return page + "#" + hash, true
}

// linkedSource is the text a URL's fragment, with its #, holds, and whether it holds one.
// A fragment that is not a query, or has no source in it, holds none.
func linkedSource(hash string) (string, bool) {
	if !strings.HasPrefix(hash, "#") {
		return "", false
	}
	// A part of the query that does not decode is left out, and the rest is read.
	q, _ := url.ParseQuery(hash[1:])
	if !q.Has("source") {
		return "", false
	}
	return q.Get("source"), true
}
