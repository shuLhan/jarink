// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Options define the options for scanning broken links.
type Options struct {
	// The URL to be scanned.
	Url     string
	scanUrl *url.URL

	PastResultFile string

	// IgnoreStatus comma separated list HTTP status code that will be
	// ignored on scan.
	// Page that return one of the IgnoreStatus will be assumed as
	// passed and not get processed.
	// The status code must in between 100-511.
	IgnoreStatus string
	ignoreStatus []int

	IsVerbose bool

	// Insecure do not report error on server with invalid certificates.
	Insecure bool
}

func (opts *Options) init() (err error) {
	var logp = `Options`

	opts.scanUrl, err = url.Parse(opts.Url)
	if err != nil {
		return fmt.Errorf(`%s: invalid URL %q`, logp, opts.Url)
	}
	opts.scanUrl.Path = strings.TrimSuffix(opts.scanUrl.Path, `/`)
	opts.scanUrl.Fragment = ""
	opts.scanUrl.RawFragment = ""

	var listCode = strings.Split(opts.IgnoreStatus, ",")
	var val string

	for _, val = range listCode {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		var code int64
		code, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf(`%s: invalid status code %q`, logp, val)
		}
		if code < http.StatusContinue ||
			code > http.StatusNetworkAuthenticationRequired {
			return fmt.Errorf(`%s: unknown status code %q`, logp, val)
		}
		opts.ignoreStatus = append(opts.ignoreStatus, int(code))
	}
	return nil
}
