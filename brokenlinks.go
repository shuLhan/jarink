// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink

import (
	"fmt"
	"slices"
	"strings"
)

const Version = `0.1.0`

// StatusBadLink status for link that is not parseable by [url.Parse] or not
// reachable during GET or HEAD, either timeout or IP or domain not exist.
const StatusBadLink = 700

// Broken store the broken link, HTTP status code, and the error message that
// cause it.
type Broken struct {
	Link  string `json:"link"`
	Error string `json:"error,omitempty"`
	Code  int    `json:"code"`
}

// BrokenlinksOptions define the options for scanning broken links.
type BrokenlinksOptions struct {
	Url            string
	PastResultFile string
	IsVerbose      bool
}

// BrokenlinksResult store the result of scanning for broken links.
type BrokenlinksResult struct {
	// BrokenLinks store the page and its broken links.
	BrokenLinks map[string][]Broken `json:"broken_links"`
}

func newBrokenlinksResult() *BrokenlinksResult {
	return &BrokenlinksResult{
		BrokenLinks: map[string][]Broken{},
	}
}

func (result *BrokenlinksResult) sort() {
	for _, listBroken := range result.BrokenLinks {
		slices.SortFunc(listBroken, func(a, b Broken) int {
			return strings.Compare(a.Link, b.Link)
		})
	}
}

// Brokenlinks scan the URL for broken links.
func Brokenlinks(opts BrokenlinksOptions) (result *BrokenlinksResult, err error) {
	var logp = `brokenlinks`
	var wrk *brokenlinksWorker

	wrk, err = newWorker(opts)
	if err != nil {
		return nil, fmt.Errorf(`%s: %s`, logp, err)
	}

	result, err = wrk.run()
	if err != nil {
		return nil, fmt.Errorf(`%s: %s`, logp, err)
	}

	return result, nil
}
