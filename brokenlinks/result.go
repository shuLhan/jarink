// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks

import (
	"slices"
	"strings"
)

// Broken store the broken link, HTTP status code, and the error message that
// cause it.
type Broken struct {
	Link  string `json:"link"`
	Error string `json:"error,omitempty"`
	Code  int    `json:"code"`
}

// Result store the result of scanning for broken links.
type Result struct {
	// BrokenLinks store the page and its broken links.
	BrokenLinks map[string][]Broken `json:"broken_links"`
}

func newResult() *Result {
	return &Result{
		BrokenLinks: map[string][]Broken{},
	}
}

func (result *Result) sort() {
	for _, listBroken := range result.BrokenLinks {
		slices.SortFunc(listBroken, func(a, b Broken) int {
			return strings.Compare(a.Link, b.Link)
		})
	}
}
