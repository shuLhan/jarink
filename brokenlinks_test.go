// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"git.sr.ht/~shulhan/jarink"
	"git.sr.ht/~shulhan/pakakeh.go/lib/test"
)

func TestBrokenlinks(t *testing.T) {
	var testUrl = `http://` + testAddress

	type testCase struct {
		exp      map[string][]jarink.Broken
		scanUrl  string
		expError string
	}

	listCase := []testCase{{
		scanUrl:  `127.0.0.1:14594`,
		expError: `brokenlinks: invalid URL "127.0.0.1:14594"`,
	}, {
		scanUrl:  `http://127.0.0.1:14594`,
		expError: `brokenlinks: Get "http://127.0.0.1:14594": dial tcp 127.0.0.1:14594: connect: connection refused`,
	}, {
		scanUrl: testUrl,
		exp: map[string][]jarink.Broken{
			testUrl: []jarink.Broken{
				{
					Link: testUrl + `/broken.png`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/brokenPage`,
					Code: http.StatusNotFound,
				}, {
					Link:  `http://127.0.0.1:abc`,
					Error: `parse "http://127.0.0.1:abc": invalid port ":abc" after host`,
					Code:  jarink.StatusBadLink,
				}, {
					Link:  `http:/127.0.0.1:11836`,
					Error: `Get "http:/127.0.0.1:11836": http: no Host in request URL`,
					Code:  jarink.StatusBadLink,
				},
			},
			testUrl + `/broken.html`: []jarink.Broken{
				{
					Link: testUrl + `/brokenPage`,
					Code: http.StatusNotFound,
				},
			},
			testUrl + `/page2`: []jarink.Broken{
				{
					Link: testUrl + `/broken.png`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken/relative`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken2.png`,
					Code: http.StatusNotFound,
				},
			},
		},
	}, {
		// Scanning on "/path" should not scan the the "/" or other
		// pages other than below of "/path" itself.
		scanUrl: testUrl + `/page2`,
		exp: map[string][]jarink.Broken{
			testUrl + `/page2`: []jarink.Broken{
				{
					Link: testUrl + `/broken.png`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken/relative`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken2.png`,
					Code: http.StatusNotFound,
				},
			},
		},
	}}

	var (
		result *jarink.BrokenlinksResult
		err    error
	)
	for _, tcase := range listCase {
		t.Logf(`--- brokenlinks: %s`, tcase.scanUrl)
		var brokenlinksOpts = jarink.BrokenlinksOptions{
			Url: tcase.scanUrl,
		}
		result, err = jarink.Brokenlinks(brokenlinksOpts)
		if err != nil {
			test.Assert(t, tcase.scanUrl+` error`,
				tcase.expError, err.Error())
			continue
		}
		//got, _ := json.MarshalIndent(result.BrokenLinks, ``, `  `)
		//t.Logf(`got=%s`, got)
		test.Assert(t, tcase.scanUrl, tcase.exp, result.BrokenLinks)
	}
}

// Test running Brokenlinks with file PastResultFile is set.
// The PastResultFile is modified to only report errors on "/page2".
func TestBrokenlinks_pastResult(t *testing.T) {
	var testUrl = `http://` + testAddress

	type testCase struct {
		exp      map[string][]jarink.Broken
		expError string
		opts     jarink.BrokenlinksOptions
	}

	listCase := []testCase{{
		// With invalid file.
		opts: jarink.BrokenlinksOptions{
			Url:            testUrl,
			PastResultFile: `testdata/invalid`,
		},
		expError: `brokenlinks: open testdata/invalid: no such file or directory`,
	}, {
		// With valid file.
		opts: jarink.BrokenlinksOptions{
			Url:            testUrl,
			PastResultFile: `testdata/past_result.json`,
		},
		exp: map[string][]jarink.Broken{
			testUrl + `/page2`: []jarink.Broken{
				{
					Link: testUrl + `/broken.png`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken/relative`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/page2/broken2.png`,
					Code: http.StatusNotFound,
				},
			},
		},
	}}

	var (
		result *jarink.BrokenlinksResult
		err    error
	)
	for _, tcase := range listCase {
		t.Logf(`--- brokenlinks: %s`, tcase.opts.Url)
		result, err = jarink.Brokenlinks(tcase.opts)
		if err != nil {
			test.Assert(t, tcase.opts.Url+` error`,
				tcase.expError, err.Error())
			continue
		}
		got, _ := json.MarshalIndent(result.BrokenLinks, ``, `  `)
		t.Logf(`got=%s`, got)
		test.Assert(t, tcase.opts.Url, tcase.exp, result.BrokenLinks)
	}
}
