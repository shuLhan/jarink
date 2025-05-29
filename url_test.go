// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"net/url"
	"testing"

	"git.sr.ht/~shulhan/pakakeh.go/lib/test"
)

func TestUrlString(t *testing.T) {
	type testCase struct {
		rawUrl string
		exp    string
	}
	var listCase = []testCase{{
		rawUrl: `http://127.0.0.1`,
		exp:    `http://127.0.0.1`,
	}, {
		rawUrl: `http://127.0.0.1/`,
		exp:    `http://127.0.0.1/`,
	}, {
		rawUrl: `http://127.0.0.1/page`,
		exp:    `http://127.0.0.1/page`,
	}, {
		rawUrl: `http://127.0.0.1/page/`,
		exp:    `http://127.0.0.1/page/`,
	}}
	for _, tcase := range listCase {
		gotUrl, err := url.Parse(tcase.rawUrl)
		if err != nil {
			t.Fatal(err)
		}
		var got = gotUrl.String()
		test.Assert(t, tcase.rawUrl, tcase.exp, got)
	}
}
