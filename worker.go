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

	// The base URL that will be joined to relative or absolute
	// links or image.
	baseUrl *url.URL

	// The URL to scan.
	scanUrl *url.URL

	opts ScanOptions

	// wg sync the goroutine scanner.
	wg sync.WaitGroup

	// seenLinkMtx guard the seenLink field from concurrent read/write.
	seenLinkMtx sync.Mutex
}

func newWorker(opts ScanOptions) (wrk *worker, err error) {
	wrk = &worker{
		opts:     opts,
		seenLink: map[string]int{},
		linkq:    make(chan linkQueue, 10000),
		errq:     make(chan error, 1),
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

	wrk.linkq <- linkQueue{
		parentUrl: nil,
		url:       wrk.scanUrl.String(),
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

		default:
			wrk.wg.Wait()

			select {
			case err = <-wrk.errq:
				return nil, err
			default:
				if len(wrk.linkq) == 0 {
					ever = false
				}
			}
		}
	}
	wrk.result.sort()
	return wrk.result, nil
}

// scan fetch the HTML page or image to check if its valid.
func (wrk *worker) scan(linkq linkQueue) {
	defer wrk.wg.Done()

	wrk.seenLinkMtx.Lock()
	statusCode, seen := wrk.seenLink[linkq.url]
	wrk.seenLinkMtx.Unlock()
	if seen {
		if statusCode >= http.StatusBadRequest {
			wrk.markDead(linkq, statusCode)
		}
		if wrk.opts.IsVerbose {
			fmt.Printf("scan: %s %d\n", linkq.url, statusCode)
		}
		return
	}
	wrk.seenLinkMtx.Lock()
	wrk.seenLink[linkq.url] = http.StatusProcessing
	wrk.seenLinkMtx.Unlock()

	if wrk.opts.IsVerbose {
		fmt.Printf("scan: %s %d\n", linkq.url, http.StatusProcessing)
	}

	var (
		httpResp *http.Response
		err      error
	)
	if linkq.kind == atom.Img {
		httpResp, err = http.Head(linkq.url)
	} else {
		httpResp, err = http.Get(linkq.url)
	}
	if err != nil {
		if linkq.parentUrl == nil {
			wrk.errq <- err
		} else {
			wrk.markDead(linkq, http.StatusNotFound)
		}
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		wrk.markDead(linkq, httpResp.StatusCode)
		return
	}
	wrk.seenLinkMtx.Lock()
	wrk.seenLink[linkq.url] = http.StatusOK
	wrk.seenLinkMtx.Unlock()

	if linkq.kind == atom.Img {
		return
	}
	if !strings.HasPrefix(linkq.url, wrk.baseUrl.String()) {
		// Do not parse the page from external domain.
		return
	}
	wrk.parseHTML(linkq.url, httpResp.Body)
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

func (wrk *worker) parseHTML(linkUrl string, body io.Reader) {
	var doc *html.Node

	doc, _ = html.Parse(body)

	// After we check the code and test for [html.Parse] there are
	// no case actual cases where HTML content will return an error.
	// The only possible error is when reading from body (io.Reader), and
	// that is also almost impossible.
	//
	// [html.Parse]: https://go.googlesource.com/net/+/refs/tags/v0.40.0/html/parse.go#2347

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
				wrk.processLink(linkUrl, attr.Val, atom.A)
			}
		}
		if node.DataAtom == atom.Img {
			for _, attr := range node.Attr {
				if attr.Key != `src` {
					continue
				}
				wrk.processLink(linkUrl, attr.Val, atom.Img)
			}
		}
	}
}

func (wrk *worker) processLink(rawParentUrl, val string, kind atom.Atom) {
	if len(val) == 0 {
		return
	}

	var parentUrl *url.URL
	var err error

	parentUrl, err = url.Parse(rawParentUrl)
	if err != nil {
		log.Fatal(err)
	}

	var newUrl *url.URL
	newUrl, err = url.Parse(val)
	if err != nil {
		var linkq = linkQueue{
			parentUrl: parentUrl,
			url:       val,
			kind:      kind,
		}
		wrk.markDead(linkq, 700)
		return
	}
	newUrl.Fragment = ""
	newUrl.RawFragment = ""

	var newUrlStr = strings.TrimSuffix(newUrl.String(), `/`)

	if kind == atom.A && val[0] == '#' {
		// Ignore link to ID, like `href="#element_id"`.
		return
	}

	// val is absolute to parent URL.
	if val[0] == '/' {
		// Link to the same domain will queued for scanning.
		newUrl = wrk.baseUrl.JoinPath(newUrl.Path)
		newUrlStr = strings.TrimSuffix(newUrl.String(), `/`)
		wrk.linkq <- linkQueue{
			parentUrl: parentUrl,
			url:       newUrlStr,
			kind:      kind,
		}
		return
	}
	if strings.HasPrefix(val, `http`) {
		wrk.linkq <- linkQueue{
			parentUrl: parentUrl,
			url:       newUrlStr,
			kind:      kind,
		}
		return
	}
	// val is relative to parent URL.
	newUrl = parentUrl.JoinPath(`/`, newUrl.Path)
	newUrlStr = strings.TrimSuffix(newUrl.String(), `/`)
	wrk.linkq <- linkQueue{
		parentUrl: parentUrl,
		url:       newUrlStr,
		kind:      kind,
	}
}
