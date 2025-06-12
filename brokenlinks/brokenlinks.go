// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks

import (
	"fmt"
)

const Version = `0.1.0`

// StatusBadLink status for link that is not parseable by [url.Parse] or not
// reachable during GET or HEAD, either timeout or IP or domain not exist.
const StatusBadLink = 700

// Options define the options for scanning broken links.
type Options struct {
	Url            string
	PastResultFile string
	IsVerbose      bool
}

// Scan the URL for broken links.
func Scan(opts Options) (result *Result, err error) {
	var logp = `brokenlinks`
	var wrk *worker

	wrk, err = newWorker(opts)
	if err != nil {
		return nil, fmt.Errorf(`%s: %s`, logp, err)
	}

	result, err = wrk.run()
	if err != nil {
		return nil, fmt.Errorf(`%s: %s`, logp, err)
	}

	return result, nil
}
