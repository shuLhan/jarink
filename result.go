// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"slices"
	"strings"
)

// Broken store the link with its HTTP status.
type Broken struct {
	Link string
	Code int
}

// Result store the result of Scan.
type Result struct {
	// PageLinks store the page and its broken links.
	PageLinks map[string][]Broken
}

func newResult() *Result {
	return &Result{
		PageLinks: map[string][]Broken{},
	}
}

func (result *Result) sort() {
	for _, listBroken := range result.PageLinks {
		slices.SortFunc(listBroken, func(a, b Broken) int {
			return strings.Compare(a.Link, b.Link)
		})
	}
}
