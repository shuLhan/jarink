// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks_test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	libnet "git.sr.ht/~shulhan/pakakeh.go/lib/net"
	"git.sr.ht/~shulhan/pakakeh.go/lib/test"

	"git.sr.ht/~shulhan/jarink/brokenlinks"
)

// The test run three web servers that serve content on "testdata/web/".
// The first web server is the one that we want to scan.
// The second web server is external web server, where HTML pages should not
// be parsed.
// The third web server is with insecure, self-signed TLS, for testing
// "insecure" option.
//
// Command to generate certificate:
//	$ openssl genrsa -out 127.0.0.1.key
//	$ openssl x509 -new -key=127.0.0.1.key -subj="/CN=shulhan" \
//		-days=3650 -out=127.0.0.1.pem

const testAddress = `127.0.0.1:11836`
const testExternalAddress = `127.0.0.1:11900`
const testInsecureAddress = `127.0.0.1:11838`

func TestMain(m *testing.M) {
	log.SetFlags(0)
	var httpDirWeb = http.Dir(`testdata/web`)
	var fshandle = http.FileServer(httpDirWeb)

	go testServer(fshandle)
	go testExternalServer(fshandle)
	go testInsecureServer(fshandle)

	var err = libnet.WaitAlive(`tcp`, testAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	err = libnet.WaitAlive(`tcp`, testExternalAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	err = libnet.WaitAlive(`tcp`, testInsecureAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func testServer(fshandle http.Handler) {
	var mux = http.NewServeMux()
	mux.HandleFunc(`/page403`, page403)
	mux.Handle(`/`, fshandle)
	var testServer = &http.Server{
		Addr:           testAddress,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var err = testServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func page403(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusForbidden)
}

func testExternalServer(fshandle http.Handler) {
	var mux = http.NewServeMux()
	mux.Handle(`/`, fshandle)
	var testServer = &http.Server{
		Addr:           testExternalAddress,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var err = testServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func testInsecureServer(fshandle http.Handler) {
	var mux = http.NewServeMux()
	mux.Handle(`/`, fshandle)
	var testServer = &http.Server{
		Addr:           testInsecureAddress,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var certFile = `testdata/127.0.0.1.pem`
	var keyFile = `testdata/127.0.0.1.key`
	var err = testServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
}

func TestBrokenlinks(t *testing.T) {
	var testUrl = `http://` + testAddress

	type testCase struct {
		exp      map[string][]brokenlinks.Broken
		expError string
		opts     brokenlinks.Options
	}

	listCase := []testCase{{
		opts: brokenlinks.Options{
			Url: `127.0.0.1:14594`,
		},
		expError: `Scan: invalid URL "127.0.0.1:14594"`,
	}, {
		opts: brokenlinks.Options{
			Url: `http://127.0.0.1:14594`,
		},
		expError: `Scan: Get "http://127.0.0.1:14594": dial tcp 127.0.0.1:14594: connect: connection refused`,
	}, {
		opts: brokenlinks.Options{
			Url:          testUrl,
			IgnoreStatus: `403`,
			Insecure:     true,
		},
		exp: map[string][]brokenlinks.Broken{
			testUrl: []brokenlinks.Broken{
				{
					Link: testUrl + `/broken.png`,
					Code: http.StatusNotFound,
				}, {
					Link: testUrl + `/brokenPage`,
					Code: http.StatusNotFound,
				}, {
					Link:  `http://127.0.0.1:abc`,
					Error: `parse "http://127.0.0.1:abc": invalid port ":abc" after host`,
					Code:  brokenlinks.StatusBadLink,
				}, {
					Link:  `http:/127.0.0.1:11836`,
					Error: `Get "http:/127.0.0.1:11836": http: no Host in request URL`,
					Code:  brokenlinks.StatusBadLink,
				},
			},
			testUrl + `/broken.html`: []brokenlinks.Broken{
				{
					Link: testUrl + `/brokenPage`,
					Code: http.StatusNotFound,
				},
			},
			testUrl + `/page2`: []brokenlinks.Broken{
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
		opts: brokenlinks.Options{
			Url: testUrl + `/page2`,
		},
		exp: map[string][]brokenlinks.Broken{
			testUrl + `/page2`: []brokenlinks.Broken{
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
		result *brokenlinks.Result
		err    error
	)
	for _, tcase := range listCase {
		t.Logf(`--- brokenlinks: %s`, tcase.opts.Url)
		result, err = brokenlinks.Scan(tcase.opts)
		if err != nil {
			test.Assert(t, tcase.opts.Url+` error`,
				tcase.expError, err.Error())
			continue
		}
		//got, _ := json.MarshalIndent(result.BrokenLinks, ``, `  `)
		//t.Logf(`got=%s`, got)
		test.Assert(t, tcase.opts.Url, tcase.exp, result.BrokenLinks)
	}
}

// Test running Brokenlinks with file PastResultFile is set.
// The PastResultFile is modified to only report errors on "/page2".
func TestBrokenlinks_pastResult(t *testing.T) {
	var testUrl = `http://` + testAddress

	type testCase struct {
		exp      map[string][]brokenlinks.Broken
		expError string
		opts     brokenlinks.Options
	}

	listCase := []testCase{{
		// With invalid file.
		opts: brokenlinks.Options{
			Url:            testUrl,
			PastResultFile: `testdata/invalid`,
		},
		expError: `Scan: open testdata/invalid: no such file or directory`,
	}, {
		// With valid file.
		opts: brokenlinks.Options{
			Url:            testUrl,
			PastResultFile: `testdata/past_result.json`,
			IgnoreStatus:   `403`,
		},
		exp: map[string][]brokenlinks.Broken{
			testUrl + `/page2`: []brokenlinks.Broken{
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
		result *brokenlinks.Result
		err    error
	)
	for _, tcase := range listCase {
		t.Logf(`--- brokenlinks: %s`, tcase.opts.Url)
		result, err = brokenlinks.Scan(tcase.opts)
		if err != nil {
			test.Assert(t, tcase.opts.Url+` error`,
				tcase.expError, err.Error())
			continue
		}
		//got, _ := json.MarshalIndent(result.BrokenLinks, ``, `  `)
		//t.Logf(`got=%s`, got)
		test.Assert(t, tcase.opts.Url, tcase.exp, result.BrokenLinks)
	}
}
