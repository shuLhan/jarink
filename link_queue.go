// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink

import (
	"net/url"
	"strings"

	"golang.org/x/net/html/atom"
)

type linkQueue struct {
	parentUrl *url.URL

	// The error from scan.
	errScan error

	// url being scanned.
	url string

	// kind of url, its either an anchor or image.
	// It set to 0 if url is the first URL being scanned.
	kind atom.Atom

	// isExternal if true the scan will issue HTTP method HEAD instead of
	// GET.
	isExternal bool

	// Status of link after scan, its mostly used the HTTP status code.
	// 0: link is the result of scan, not processed yet.
	// StatusBadLink: link is invalid, not parseable or unreachable.
	// 200 - 211: OK.
	// 400 - 511: Error.
	status int
}

// checkExternal set the isExternal field to be true if
//
// (1) [linkQueue.url] does not start with [brokenlinksWorker.scanUrl]
//
// (2) linkQueue is from scanPastResult, indicated by non-nil
// [brokenlinksWorker.pastResult].
// In this case, we did not want to scan the other pages from the same scanUrl
// domain.
func (linkq *linkQueue) checkExternal(wrk *brokenlinksWorker) {
	if !strings.HasPrefix(linkq.url, wrk.scanUrl.String()) {
		linkq.isExternal = true
		return
	}
	if wrk.pastResult != nil {
		linkq.isExternal = true
		return
	}
}
