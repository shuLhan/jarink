// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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

	// The base URL that will be joined to relative or absolute
	// links or image.
	baseUrl *url.URL

	// The URL to scan.
	scanUrl *url.URL

	opts ScanOptions

	// wg sync the goroutine scanner.
	wg sync.WaitGroup
}

func newWorker(opts ScanOptions) (wrk *worker, err error) {
	wrk = &worker{
		opts:     opts,
		seenLink: map[string]int{},
		resultq:  make(chan map[string]linkQueue, 100),
		result:   newResult(),
	}

	wrk.scanUrl, err = url.Parse(opts.Url)
	if err != nil {
		return nil, fmt.Errorf(`invalid URL %q`, opts.Url)
	}

	wrk.scanUrl.Path = strings.TrimSuffix(wrk.scanUrl.Path, `/`)
	wrk.scanUrl.Fragment = ""
	wrk.scanUrl.RawFragment = ""

	wrk.baseUrl = &url.URL{
		Scheme: wrk.scanUrl.Scheme,
		Host:   wrk.scanUrl.Host,
	}

	return wrk, nil
}

func (wrk *worker) run() (result *Result, err error) {
	// Scan the first URL to make sure that the server is reachable.
	var firstLinkq = linkQueue{
		parentUrl: nil,
		url:       wrk.scanUrl.String(),
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
			wrk.markDead(linkq)
			continue
		}

		wrk.seenLink[linkq.url] = http.StatusProcessing
		wrk.wg.Add(1)
		go wrk.scan(linkq)
	}

	var listWaitStatus []linkQueue
	var isScanning = true
	for isScanning {
		select {
		case resultq := <-wrk.resultq:

			// The resultq contains the original URL being scanned
			// and its child links.
			// For example, scanning "http://example.tld" result
			// in
			//
			//	"http://example.tld": {status=200}
			//	"http://example.tld/page": {status=0}
			//	"http://example.tld/image.png": {status=0}
			//	"http://bad:domain/image.png": {status=700}

			for _, linkq := range resultq {
				if linkq.status >= http.StatusBadRequest {
					wrk.markDead(linkq)
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
					wrk.markDead(linkq)
					continue
				}
				if seenStatus >= http.StatusOK {
					// The link has been processed and its
					// not an error.
					continue
				}
				if linkq.status != 0 {
					// linkq is the result of scan with
					// non error status.
					wrk.seenLink[linkq.url] = linkq.status
					continue
				}

				// The link being processed by other
				// goroutine.
				listWaitStatus = append(listWaitStatus, linkq)
			}

		default:
			wrk.wg.Wait()
			if len(wrk.resultq) != 0 {
				continue
			}
			var newList []linkQueue
			for _, linkq := range listWaitStatus {
				seenStatus := wrk.seenLink[linkq.url]
				if seenStatus == http.StatusProcessing {
					// Scanning still in progress.
					newList = append(newList, linkq)
					continue
				}
				if seenStatus >= http.StatusBadRequest {
					linkq.status = seenStatus
					wrk.markDead(linkq)
					continue
				}
			}
			if len(newList) != 0 {
				// There are link that still waiting for
				// scanning to be completed.
				listWaitStatus = newList
				continue
			}
			isScanning = false
		}
	}
	wrk.result.sort()
	return wrk.result, nil
}

func (wrk *worker) markDead(linkq linkQueue) {
	var parentUrl = linkq.parentUrl.String()
	var listBroken = wrk.result.PageLinks[parentUrl]
	var brokenLink = Broken{
		Link: linkq.url,
		Code: linkq.status,
	}
	listBroken = append(listBroken, brokenLink)
	wrk.result.PageLinks[parentUrl] = listBroken
	wrk.seenLink[linkq.url] = linkq.status
}

// scan fetch the HTML page or image to check if its valid.
func (wrk *worker) scan(linkq linkQueue) {
	defer func() {
		if wrk.opts.IsVerbose {
			fmt.Printf("  done: %d %s\n", linkq.status, linkq.url)
		}
		wrk.wg.Done()
	}()

	if wrk.opts.IsVerbose {
		fmt.Printf("scan: %d %s\n", linkq.status, linkq.url)
	}

	var (
		resultq  = map[string]linkQueue{}
		httpResp *http.Response
		err      error
	)
	if linkq.kind == atom.Img {
		httpResp, err = http.Head(linkq.url)
	} else {
		httpResp, err = http.Get(linkq.url)
	}
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

	if httpResp.StatusCode >= http.StatusBadRequest {
		go wrk.pushResult(resultq)
		return
	}
	if linkq.kind == atom.Img {
		go wrk.pushResult(resultq)
		return
	}
	if !strings.HasPrefix(linkq.url, wrk.baseUrl.String()) {
		// Do not parse the HTML page from external domain, only need
		// its HTTP status code.
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
	var link string
	var status int
	for node = range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		link = ""
		if node.DataAtom == atom.A {
			for _, attr := range node.Attr {
				if attr.Key != `href` {
					continue
				}
				link, status = wrk.processLink(scanUrl, attr.Val, atom.A)
				break
			}
		} else if node.DataAtom == atom.Img {
			for _, attr := range node.Attr {
				if attr.Key != `src` {
					continue
				}
				link, status = wrk.processLink(scanUrl, attr.Val, atom.Img)
				break
			}
		} else {
			continue
		}
		if link == "" {
			continue
		}
		resultq[link] = linkQueue{
			parentUrl: scanUrl,
			url:       link,
			kind:      node.DataAtom,
			status:    status,
		}
	}
	go wrk.pushResult(resultq)
}

func (wrk *worker) processLink(parentUrl *url.URL, val string, kind atom.Atom) (
	link string, status int,
) {
	if len(val) == 0 {
		return "", 0
	}

	var newUrl *url.URL
	var err error
	newUrl, err = url.Parse(val)
	if err != nil {
		return val, StatusBadLink
	}
	newUrl.Fragment = ""
	newUrl.RawFragment = ""

	if kind == atom.A && val[0] == '#' {
		// Ignore link to ID, like `href="#element_id"`.
		return "", 0
	}
	if strings.HasPrefix(val, `http`) {
		link = strings.TrimSuffix(newUrl.String(), `/`)
		return link, 0
	}
	if val[0] == '/' {
		// val is absolute to parent URL.
		newUrl = wrk.baseUrl.JoinPath(newUrl.Path)
	} else {
		// val is relative to parent URL.
		newUrl = parentUrl.JoinPath(`/`, newUrl.Path)
	}
	link = strings.TrimSuffix(newUrl.String(), `/`)
	return link, 0
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
