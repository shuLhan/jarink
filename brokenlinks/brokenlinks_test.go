// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks_test

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	libnet "git.sr.ht/~shulhan/pakakeh.go/lib/net"
	"git.sr.ht/~shulhan/pakakeh.go/lib/test"

	"git.sr.ht/~shulhan/jarink/brokenlinks"
)

// The test run four web servers.
//
// The first web server is the one that we want to scan, it serve content on
// "testdata/web".
//
// The second web server is an external web server, where HTML pages should not
// be parsed.
//
// The third web server is with insecure, self-signed TLS, for testing
// "insecure" option.
// Commands to generate certificate:
//	$ openssl genrsa -out 127.0.0.1.key
//	$ openssl x509 -new -key=127.0.0.1.key -subj="/CN=shulhan" \
//		-days=3650 -out=127.0.0.1.pem
//
// The fourth web server is slow response web server.

const testAddress = `127.0.0.1:11836`
const testExternalAddress = `127.0.0.1:11900`
const testInsecureAddress = `127.0.0.1:11838`
const testAddressSlow = `127.0.0.1:11839`

func TestMain(m *testing.M) {
	log.SetFlags(0)
	var httpDirWeb = http.Dir(`testdata/web`)
	var fshandle = http.FileServer(httpDirWeb)

	go testServer(fshandle)
	go testExternalServer(fshandle)
	go testInsecureServer(fshandle)
	go runServerSlow(testAddressSlow)

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
	err = libnet.WaitAlive(`tcp`, testAddressSlow, 5*time.Second)
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

func runServerSlow(addr string) {
	var mux = http.NewServeMux()
	mux.HandleFunc(`/`, func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		var body = []byte(`<html><body>
			<a href="/slow1">Slow 1</a>
			<a href="/slow2">Slow 2</a>
			<a href="/slow3">Slow 3</a>
			</body></html>`)
		resp.Write(body)
	})

	mux.HandleFunc(`/slow1`,
		func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusOK)
			var body = []byte(`<html><body>
				<a href="/slow1/sub">Slow 1, sub</a>
				<a href="/slow2/sub">Slow 2, sub</a>
				<a href="/slow3/sub">Slow 3, sub</a>
				</body></html>`)
			resp.Write(body)
		})
	mux.HandleFunc(`/slow2`,
		func(resp http.ResponseWriter, req *http.Request) {
			time.Sleep(1 * time.Second)
			resp.WriteHeader(http.StatusOK)
			var body = []byte(`<html><body>
				<a href="/slow1/sub">Slow 1, sub</a>
				<a href="/slow2/sub">Slow 2, sub</a>
				<a href="/slow3/sub">Slow 3, sub</a>
				</body></html>`)
			resp.Write(body)
		})
	mux.HandleFunc(`/slow3`,
		func(resp http.ResponseWriter, req *http.Request) {
			time.Sleep(2 * time.Second)
			resp.WriteHeader(http.StatusOK)
			var body = []byte(`<html><body>
				<a href="/slow1/sub">Slow 1, sub</a>
				<a href="/slow2/sub">Slow 2, sub</a>
				<a href="/slow3/sub">Slow 3, sub</a>
				</body></html>`)
			resp.Write(body)
		})

	mux.HandleFunc(`/slow1/sub`,
		func(resp http.ResponseWriter, req *http.Request) {
			time.Sleep(4 * time.Second)
			resp.WriteHeader(http.StatusOK)
		})
	mux.HandleFunc(`/slow2/sub`,
		func(resp http.ResponseWriter, req *http.Request) {
			time.Sleep(5 * time.Second)
			resp.WriteHeader(http.StatusOK)
		})
	mux.HandleFunc(`/slow3/sub`,
		func(resp http.ResponseWriter, req *http.Request) {
			time.Sleep(6 * time.Second)
			resp.WriteHeader(http.StatusForbidden)
		})

	var httpServer = &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func TestScan(t *testing.T) {
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
		expError: `Scan: Options: invalid URL "127.0.0.1:14594"`,
	}, {
		opts: brokenlinks.Options{
			Url: `http://127.0.0.1:14594`,
		},
		expError: `Scan: Get "http://127.0.0.1:14594": dial tcp 127.0.0.1:14594: connect: connection refused`,
	}, {
		opts: brokenlinks.Options{
			Url:          testUrl,
			IgnoreStatus: "abc",
		},
		expError: `Scan: Options: invalid status code "abc"`,
	}, {
		opts: brokenlinks.Options{
			Url:          testUrl,
			IgnoreStatus: "50",
		},
		expError: `Scan: Options: unknown status code "50"`,
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
		// Scanning on "/page2" should not scan the the "/" or other
		// pages other than below of "/page2" itself.
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

// Test scanning with [Options.PastResultFile] is set.
// The PastResultFile is modified to only report errors on "/page2".
func TestScan_pastResult(t *testing.T) {
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

func TestScan_slow(t *testing.T) {
	const testUrl = `http://` + testAddressSlow

	var opts = brokenlinks.Options{
		Url: testUrl,
	}

	var gotResult *brokenlinks.Result
	var err error
	gotResult, err = brokenlinks.Scan(opts)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := json.MarshalIndent(gotResult, ``, `  `)
	t.Logf(`got=%s`, got)

	var expResult = &brokenlinks.Result{
		BrokenLinks: map[string][]brokenlinks.Broken{
			testUrl + `/slow1`: []brokenlinks.Broken{{
				Link: testUrl + `/slow3/sub`,
				Code: http.StatusForbidden,
			}},
			testUrl + `/slow2`: []brokenlinks.Broken{{
				Link: testUrl + `/slow3/sub`,
				Code: http.StatusForbidden,
			}},
			testUrl + `/slow3`: []brokenlinks.Broken{{
				Link: testUrl + `/slow3/sub`,
				Code: http.StatusForbidden,
			}},
		},
	}
	test.Assert(t, `TestScan_slow`, expResult, gotResult)
}
