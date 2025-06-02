// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink_test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	libnet "git.sr.ht/~shulhan/pakakeh.go/lib/net"
)

// The test run two web servers that serve content on "testdata/web/".
// The first web server is the one that we want to scan.
// The second web server is external web server, where HTML pages should not
// be parsed.

const testAddress = `127.0.0.1:11836`
const testExternalAddress = `127.0.0.1:11900`

func TestMain(m *testing.M) {
	log.SetFlags(0)
	var httpDirWeb = http.Dir(`testdata/web`)
	var fshandle = http.FileServer(httpDirWeb)

	go func() {
		var mux = http.NewServeMux()
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
	}()
	go func() {
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
	}()

	var err = libnet.WaitAlive(`tcp`, testAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	err = libnet.WaitAlive(`tcp`, testExternalAddress, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}
