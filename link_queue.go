// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"net/url"

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
