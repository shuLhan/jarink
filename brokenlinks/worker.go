// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type worker struct {
	// seenLink store the URL being or has been scanned and its HTTP
	// status code.
	seenLink map[string]int

	// resultq channel that collect result from scanning.
	resultq chan map[string]linkQueue

	// result contains the final result after all of the pages has been
	// scanned.
	result *Result

	// pastResult containts the past scan result, loaded from file
	// [Options.PastResultFile].
	pastResult *Result

	// The base URL that will be joined to relative or absolute
	// links or image.
	baseUrl *url.URL

	log *log.Logger

	httpc *http.Client

	opts Options

	// wg sync the goroutine scanner.
	wg sync.WaitGroup
}

func newWorker(opts Options) (wrk *worker, err error) {
	var netDial = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	var tlsConfig = &tls.Config{
		InsecureSkipVerify: opts.Insecure,
	}

	wrk = &worker{
		opts:     opts,
		seenLink: map[string]int{},
		resultq:  make(chan map[string]linkQueue, 100),
		result:   newResult(),
		log:      log.New(os.Stderr, ``, log.LstdFlags),
		httpc: &http.Client{
			Transport: &http.Transport{
				DialContext:           netDial.DialContext,
				ExpectContinueTimeout: 1 * time.Second,
				ForceAttemptHTTP2:     true,
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConns:          100,
				TLSClientConfig:       tlsConfig,
				TLSHandshakeTimeout:   10 * time.Second,
			},
		},
	}

	wrk.baseUrl = &url.URL{
		Scheme: wrk.opts.scanUrl.Scheme,
		Host:   wrk.opts.scanUrl.Host,
	}

	if opts.PastResultFile == "" {
		// Run with normal scan.
		return wrk, nil
	}

	pastresult, err := os.ReadFile(opts.PastResultFile)
	if err != nil {
		return nil, err
	}

	wrk.pastResult = newResult()
	err = json.Unmarshal(pastresult, &wrk.pastResult)
	if err != nil {
		return nil, err
	}

	return wrk, nil
}

func (wrk *worker) run() (result *Result, err error) {
	if wrk.pastResult == nil {
		result, err = wrk.scanAll()
	} else {
		result, err = wrk.scanPastResult()
	}
	return result, err
}

// scanAll scan all pages start from [Options.Url].
func (wrk *worker) scanAll() (result *Result, err error) {
	// Scan the first URL to make sure that the server is reachable.
	var firstLinkq = linkQueue{
		parentUrl: nil,
		url:       wrk.opts.scanUrl.String(),
		status:    http.StatusProcessing,
	}
	wrk.seenLink[firstLinkq.url] = http.StatusProcessing

	wrk.wg.Add(1)
	go wrk.scan(firstLinkq)
	wrk.wg.Wait()

	var resultq = <-wrk.resultq
	for _, linkq := range resultq {
		if linkq.url == firstLinkq.url {
			if linkq.errScan != nil {
				return nil, linkq.errScan
			}
			wrk.seenLink[linkq.url] = linkq.status
			continue
		}
		if linkq.status >= http.StatusBadRequest {
			wrk.markBroken(linkq)
			continue
		}

		wrk.seenLink[linkq.url] = http.StatusProcessing
		wrk.wg.Add(1)
		go wrk.scan(linkq)
	}

	var tick = time.NewTicker(500 * time.Millisecond)
	var listWaitStatus []linkQueue
	var isScanning = true
	for isScanning {
		select {
		case resultq := <-wrk.resultq:
			listWaitStatus = wrk.processResult(resultq, listWaitStatus)

		case <-tick.C:
			wrk.wg.Wait()
			if len(wrk.resultq) != 0 {
				continue
			}
			if len(listWaitStatus) != 0 {
				// There are links that still waiting for
				// scanning to be completed.
				continue
			}
			isScanning = false
		}
	}
	wrk.result.sort()
	return wrk.result, nil
}

// scanPastResult scan only pages reported inside
// [Result.BrokenLinks].
func (wrk *worker) scanPastResult() (
	result *Result, err error,
) {
	go func() {
		for page := range wrk.pastResult.BrokenLinks {
			var linkq = linkQueue{
				parentUrl: nil,
				url:       page,
				status:    http.StatusProcessing,
			}
			wrk.seenLink[linkq.url] = http.StatusProcessing
			wrk.wg.Add(1)
			go wrk.scan(linkq)
		}
	}()

	var tick = time.NewTicker(500 * time.Millisecond)
	var listWaitStatus []linkQueue
	var isScanning = true
	for isScanning {
		select {
		case resultq := <-wrk.resultq:
			listWaitStatus = wrk.processResult(resultq, listWaitStatus)

		case <-tick.C:
			wrk.wg.Wait()
			if len(wrk.resultq) != 0 {
				continue
			}
			if len(listWaitStatus) != 0 {
				// There are links that still waiting for
				// scanning to be completed.
				continue
			}
			isScanning = false
		}
	}
	wrk.result.sort()
	return wrk.result, nil
}

// processResult the resultq contains the original URL being scanned
// and its child links.
// For example, scanning "http://example.tld" result in
//
//	"http://example.tld": {status=200}
//	"http://example.tld/page": {status=0}
//	"http://example.tld/image.png": {status=0}
//	"http://bad:domain/image.png": {status=700}
func (wrk *worker) processResult(
	resultq map[string]linkQueue, listWaitStatus []linkQueue,
) (
	newList []linkQueue,
) {
	for _, linkq := range resultq {
		if linkq.status >= http.StatusBadRequest {
			wrk.markBroken(linkq)
			continue
		}
		if linkq.status != 0 {
			// linkq is the result of scan with
			// non error status.
			wrk.seenLink[linkq.url] = linkq.status
			continue
		}

		seenStatus, seen := wrk.seenLink[linkq.url]
		if !seen {
			wrk.seenLink[linkq.url] = http.StatusProcessing
			wrk.wg.Add(1)
			go wrk.scan(linkq)
			continue
		}
		if seenStatus >= http.StatusBadRequest {
			linkq.status = seenStatus
			wrk.markBroken(linkq)
			continue
		}
		if seenStatus >= http.StatusOK {
			// The link has been processed and its
			// not an error.
			continue
		}
		// The link being processed by other goroutine.
		linkq.status = seenStatus
		newList = append(newList, linkq)
	}
	for _, linkq := range listWaitStatus {
		seenStatus := wrk.seenLink[linkq.url]
		if seenStatus >= http.StatusBadRequest {
			linkq.status = seenStatus
			wrk.markBroken(linkq)
			continue
		}
		if seenStatus >= http.StatusOK {
			continue
		}
		if seenStatus == http.StatusProcessing {
			// Scanning still in progress.
			newList = append(newList, linkq)
			continue
		}
	}
	return newList
}

func (wrk *worker) markBroken(linkq linkQueue) {
	var parentUrl = linkq.parentUrl.String()
	var listBroken = wrk.result.BrokenLinks[parentUrl]
	var brokenLink = Broken{
		Link: linkq.url,
		Code: linkq.status,
	}
	if linkq.errScan != nil {
		brokenLink.Error = linkq.errScan.Error()
	}
	listBroken = append(listBroken, brokenLink)
	wrk.result.BrokenLinks[parentUrl] = listBroken

	wrk.seenLink[linkq.url] = linkq.status
}

// scan fetch the HTML page or image to check if its valid.
func (wrk *worker) scan(linkq linkQueue) {
	defer func() {
		if wrk.opts.IsVerbose && linkq.errScan != nil {
			wrk.log.Printf("error: %d %s error=%v\n", linkq.status,
				linkq.url, linkq.errScan)
		}
		wrk.wg.Done()
	}()

	var (
		resultq  = map[string]linkQueue{}
		httpResp *http.Response
		err      error
	)
	httpResp, err = wrk.fetch(linkq)
	if err != nil {
		linkq.status = StatusBadLink
		linkq.errScan = err
		resultq[linkq.url] = linkq
		go wrk.pushResult(resultq)
		return
	}
	defer httpResp.Body.Close()

	linkq.status = httpResp.StatusCode
	resultq[linkq.url] = linkq

	if slices.Contains(wrk.opts.ignoreStatus, httpResp.StatusCode) {
		return
	}

	if httpResp.StatusCode >= http.StatusBadRequest {
		go wrk.pushResult(resultq)
		return
	}
	if linkq.kind == atom.Img || linkq.isExternal {
		go wrk.pushResult(resultq)
		return
	}

	var doc *html.Node
	doc, _ = html.Parse(httpResp.Body)

	// After we check the code and test for [html.Parse] there are
	// no case actual cases where HTML content will return an error.
	// The only possible error is when reading from body (io.Reader), and
	// that is also almost impossible.
	//
	// [html.Parse]: https://go.googlesource.com/net/+/refs/tags/v0.40.0/html/parse.go#2347

	var scanUrl *url.URL

	scanUrl, err = url.Parse(linkq.url)
	if err != nil {
		log.Fatal(err)
	}

	var node *html.Node
	for node = range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		var nodeLink *linkQueue
		if node.DataAtom == atom.A {
			for _, attr := range node.Attr {
				if attr.Key != `href` {
					continue
				}
				nodeLink = wrk.processLink(scanUrl, attr.Val, atom.A)
				break
			}
		} else if node.DataAtom == atom.Img {
			for _, attr := range node.Attr {
				if attr.Key != `src` {
					continue
				}
				nodeLink = wrk.processLink(scanUrl, attr.Val, atom.Img)
				break
			}
		} else {
			continue
		}
		if nodeLink == nil {
			continue
		}
		_, seen := resultq[nodeLink.url]
		if !seen {
			nodeLink.checkExternal(wrk)
			resultq[nodeLink.url] = *nodeLink
		}
	}
	go wrk.pushResult(resultq)
}

func (wrk *worker) fetch(linkq linkQueue) (
	httpResp *http.Response,
	err error,
) {
	const maxRetry = 5
	var retry int
	for retry < 5 {
		if linkq.kind == atom.Img {
			if wrk.opts.IsVerbose {
				wrk.log.Printf("scan: HEAD %s\n", linkq.url)
			}
			httpResp, err = wrk.httpc.Head(linkq.url)
		} else {
			if wrk.opts.IsVerbose {
				wrk.log.Printf("scan: GET %s\n", linkq.url)
			}
			httpResp, err = wrk.httpc.Get(linkq.url)
		}
		if err == nil {
			return httpResp, nil
		}
		var errDNS *net.DNSError
		if !errors.As(err, &errDNS) {
			return nil, err
		}
		if errDNS.Timeout() {
			retry++
		}
	}
	return nil, err
}

func (wrk *worker) processLink(parentUrl *url.URL, val string, kind atom.Atom) (
	linkq *linkQueue,
) {
	if len(val) == 0 {
		return nil
	}

	var newUrl *url.URL
	var err error
	newUrl, err = url.Parse(val)
	if err != nil {
		return &linkQueue{
			parentUrl: parentUrl,
			errScan:   err,
			url:       val,
			kind:      kind,
			status:    StatusBadLink,
		}
	}
	newUrl.Fragment = ""
	newUrl.RawFragment = ""

	if kind == atom.A && val[0] == '#' {
		// Ignore link to ID, like `href="#element_id"`.
		return nil
	}
	if strings.HasPrefix(val, `http`) {
		return &linkQueue{
			parentUrl: parentUrl,
			url:       strings.TrimSuffix(newUrl.String(), `/`),
			kind:      kind,
		}
	}
	if val[0] == '/' {
		// val is absolute to parent URL.
		newUrl = wrk.baseUrl.JoinPath(newUrl.Path)
	} else {
		// val is relative to parent URL.
		newUrl = parentUrl.JoinPath(`/`, newUrl.Path)
	}
	linkq = &linkQueue{
		parentUrl: parentUrl,
		url:       strings.TrimSuffix(newUrl.String(), `/`),
		kind:      kind,
	}
	return linkq
}

func (wrk *worker) pushResult(resultq map[string]linkQueue) {
	var tick = time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case wrk.resultq <- resultq:
			tick.Stop()
			return
		case <-tick.C:
		}
	}
}
