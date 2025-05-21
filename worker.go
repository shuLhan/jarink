// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"golang.org/x/net/html"
)

type worker struct {
	// seenPage store the page URL that has been scanned and its HTTP status
	// code.
	seenPage map[string]int

	// pageq contains queue of page URL to be scanned.
	pageq chan string

	// errq contains error from scanning a page URL.
	errq chan error

	// result contains map of page URL and its list of broken link.
	result *Result

	// wg sync the goroutine scanner.
	wg sync.WaitGroup

	// seenPageMtx guard the seenPage field from concurrent read/write.
	seenPageMtx sync.Mutex
}

func newWorker(baseUrl string) (wrk *worker) {
	wrk = &worker{
		pageq:    make(chan string, 1),
		errq:     make(chan error, 1),
		seenPage: map[string]int{},
		result:   newResult(),
	}
	wrk.pageq <- baseUrl
	return wrk
}

func (wrk *worker) run() (result *Result, err error) {
	for len(wrk.pageq) > 0 {
		select {
		case page := <-wrk.pageq:
			wrk.wg.Add(1)
			go wrk.scan(page)

		case err = <-wrk.errq:
			close(wrk.pageq)
			return nil, err

		default:
			wrk.wg.Wait()
		}
	}
	return wrk.result, nil
}

// scan the function that fetch the HTML page and scan for broken links.
func (wrk *worker) scan(pageUrl string) {
	var logp = `scan`

	defer wrk.wg.Done()

	if wrk.hasSeen(pageUrl) {
		return
	}
	wrk.markSeen(pageUrl, http.StatusProcessing)

	var (
		httpResp *http.Response
		err      error
	)
	httpResp, err = http.Get(pageUrl)
	if err != nil {
		wrk.errq <- err
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		wrk.errq <- fmt.Errorf(`%s %s: return HTTP status code %d`,
			logp, pageUrl, httpResp.StatusCode)
		return
	}

	defer httpResp.Body.Close()

	err = wrk.parseHTML(pageUrl, httpResp.Body)
	if err != nil {
		wrk.errq <- fmt.Errorf(`%s %s: %w`, logp, pageUrl, err)
		return
	}
}

func (wrk *worker) parseHTML(pageUrl string, body io.Reader) (err error) {
	var (
		logp = `parseHTML`
		doc  *html.Node
	)
	doc, err = html.Parse(body)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	var node *html.Node
	for node = range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
	}
	return nil
}

// hasSeen return true if the pageUrl has been scanned.
func (wrk *worker) hasSeen(pageUrl string) (ok bool) {
	wrk.seenPageMtx.Lock()
	_, ok = wrk.seenPage[pageUrl]
	wrk.seenPageMtx.Unlock()
	return ok
}

func (wrk *worker) markSeen(pageUrl string, httpStatusCode int) {
	wrk.seenPageMtx.Lock()
	wrk.seenPage[pageUrl] = httpStatusCode
	wrk.seenPageMtx.Unlock()
}
