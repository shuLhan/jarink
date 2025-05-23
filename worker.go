// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type worker struct {
	// seenLink store the page URL that has been scanned and its HTTP status
	// code.
	seenLink map[string]int

	// linkq contains queue of page URL to be scanned.
	linkq chan linkQueue

	// errq contains error from scanning a page URL.
	errq chan error

	// result contains map of page URL and its list of broken link.
	result *Result

	// The base URL to scan that will be joined to relative or absolute
	// links or image.
	baseUrl *url.URL

	// wg sync the goroutine scanner.
	wg sync.WaitGroup

	// seenLinkMtx guard the seenLink field from concurrent read/write.
	seenLinkMtx sync.Mutex
}

func newWorker(baseUrl string) (wrk *worker, err error) {
	wrk = &worker{
		seenLink: map[string]int{},
		linkq:    make(chan linkQueue, 1000),
		errq:     make(chan error, 1),
		result:   newResult(),
	}
	wrk.baseUrl, err = url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	wrk.baseUrl = wrk.baseUrl.JoinPath(`/`)
	wrk.linkq <- linkQueue{
		parentUrl: nil,
		url:       wrk.baseUrl.String(),
	}
	return wrk, nil
}

func (wrk *worker) run() (result *Result, err error) {
	var ever bool = true
	for ever {
		select {
		case linkq := <-wrk.linkq:
			wrk.wg.Add(1)
			go wrk.scan(linkq)

		case err = <-wrk.errq:
			return nil, err

		default:
			wrk.wg.Wait()
			if len(wrk.linkq) == 0 {
				ever = false
			}
		}
	}
	wrk.result.sort()
	return wrk.result, nil
}

// scan the function that fetch the HTML page and scan for broken links.
func (wrk *worker) scan(linkq linkQueue) {
	var logp = `scan`

	defer wrk.wg.Done()

	wrk.seenLinkMtx.Lock()
	statusCode, seen := wrk.seenLink[linkq.url]
	wrk.seenLinkMtx.Unlock()
	if seen {
		if statusCode >= http.StatusBadRequest {
			wrk.markDead(linkq, statusCode)
		}
		return
	}
	wrk.seenLinkMtx.Lock()
	wrk.seenLink[linkq.url] = http.StatusProcessing
	wrk.seenLinkMtx.Unlock()

	var (
		httpResp *http.Response
		err      error
	)
	httpResp, err = http.Get(linkq.url)
	if err != nil {
		wrk.errq <- err
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		wrk.markDead(linkq, httpResp.StatusCode)
		return
	}

	defer httpResp.Body.Close()

	err = wrk.parseHTML(linkq.url, httpResp.Body)
	if err != nil {
		wrk.errq <- fmt.Errorf(`%s %s: %w`, logp, linkq.url, err)
		return
	}
}

func (wrk *worker) markDead(linkq linkQueue, httpStatusCode int) {
	var parentUrl = linkq.parentUrl.String()

	wrk.seenLinkMtx.Lock()
	var listBroken = wrk.result.PageLinks[parentUrl]
	listBroken = append(listBroken, Broken{
		Link: linkq.url,
		Code: httpStatusCode,
	})
	wrk.result.PageLinks[parentUrl] = listBroken
	wrk.seenLink[linkq.url] = httpStatusCode
	wrk.seenLinkMtx.Unlock()
}

func (wrk *worker) parseHTML(linkUrl string, body io.Reader) (err error) {
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
		if node.DataAtom == atom.A {
			for _, attr := range node.Attr {
				if attr.Key != `href` {
					continue
				}
				wrk.processLink(linkUrl, attr.Val)
			}
		}
		if node.DataAtom == atom.Img {
			for _, attr := range node.Attr {
				if attr.Key != `src` {
					continue
				}
				wrk.processLink(linkUrl, attr.Val)
			}
		}
	}
	return nil
}

func (wrk *worker) processLink(rawParentUrl string, val string) {
	if len(val) == 0 {
		return
	}
	var parentUrl *url.URL
	var err error
	parentUrl, err = url.Parse(rawParentUrl)
	if err != nil {
		log.Fatal(err)
	}
	if val[0] == '/' {
		// Link to the same domain will queued for scanning.
		var newUrl = wrk.baseUrl.JoinPath(val)
		wrk.linkq <- linkQueue{
			parentUrl: parentUrl,
			url:       newUrl.String(),
		}
		return
	}
	if strings.HasPrefix(val, `http`) {
		var newUrl *url.URL
		newUrl, err = url.Parse(val)
		if err != nil {
			var linkq = linkQueue{
				parentUrl: parentUrl,
				url:       val,
			}
			wrk.markDead(linkq, 700)
			return
		}
		wrk.linkq <- linkQueue{
			parentUrl: parentUrl,
			url:       newUrl.String(),
		}
		return
	}
	// val is relative to parent URL.
	var newUrl = parentUrl.JoinPath(`/`, val)
	wrk.linkq <- linkQueue{
		parentUrl: parentUrl,
		url:       newUrl.String(),
	}
}
