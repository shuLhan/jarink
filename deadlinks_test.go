// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks_test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"git.sr.ht/~shulhan/deadlinks"
	libnet "git.sr.ht/~shulhan/pakakeh.go/lib/net"
	"git.sr.ht/~shulhan/pakakeh.go/lib/test"
)

const testListenAddress = `127.0.0.1:11836`

func TestMain(m *testing.M) {
	var httpDirWeb = http.Dir(`testdata/web`)
	var fshandle = http.FileServer(httpDirWeb)

	http.Handle(`/`, fshandle)
	go func() {
		var err = http.ListenAndServe(testListenAddress, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	var err = libnet.WaitAlive(`tcp`, testListenAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestDeadLinks_Scan(t *testing.T) {
	var testUrl = `http://` + testListenAddress

	type testCase struct {
		exp      map[string][]deadlinks.Broken
		scanUrl  string
		expError string
	}

	listCase := []testCase{{
		scanUrl:  `127.0.0.1:14594`,
		expError: `Scan: invalid URL "127.0.0.1:14594"`,
	}, {
		scanUrl:  `http://127.0.0.1:14594`,
		expError: `Scan: Get "http://127.0.0.1:14594": dial tcp 127.0.0.1:14594: connect: connection refused`,
	}, {
		scanUrl: testUrl,
		exp: map[string][]deadlinks.Broken{
			testUrl: []deadlinks.Broken{{
				Link: testUrl + `/broken.png`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/brokenPage`,
				Code: http.StatusNotFound,
			}},
			testUrl + `/broken.html`: []deadlinks.Broken{{
				Link: testUrl + `/brokenPage`,
				Code: http.StatusNotFound,
			}},
			testUrl + `/page2`: []deadlinks.Broken{{
				Link: testUrl + `/broken.png`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/page2/broken/relative`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/page2/broken2.png`,
				Code: http.StatusNotFound,
			}},
		},
	}, {
		scanUrl: testUrl + `/page2`,
		exp: map[string][]deadlinks.Broken{
			testUrl: []deadlinks.Broken{{
				Link: testUrl + `/broken.png`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/brokenPage`,
				Code: http.StatusNotFound,
			}},
			testUrl + `/broken.html`: []deadlinks.Broken{{
				Link: testUrl + `/brokenPage`,
				Code: http.StatusNotFound,
			}},
			testUrl + `/page2`: []deadlinks.Broken{{
				Link: testUrl + `/broken.png`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/page2/broken/relative`,
				Code: http.StatusNotFound,
			}, {
				Link: testUrl + `/page2/broken2.png`,
				Code: http.StatusNotFound,
			}},
		},
	}}

	var (
		result *deadlinks.Result
		err    error
	)
	for _, tcase := range listCase {
		result, err = deadlinks.Scan(tcase.scanUrl)
		if err != nil {
			test.Assert(t, tcase.scanUrl+` error`,
				tcase.expError, err.Error())
			continue
		}
		//got, _ := json.MarshalIndent(result.PageLinks, ``, `  `)
		//t.Logf(`got=%s`, got)
		test.Assert(t, tcase.scanUrl, tcase.exp, result.PageLinks)
	}
}
