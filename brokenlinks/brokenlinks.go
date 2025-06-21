// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package brokenlinks

import (
	"fmt"
	"log"
)

const Version = `0.1.0`

// StatusBadLink status for link that is not parseable by [url.Parse] or not
// reachable during GET or HEAD, either timeout or IP or domain not exist.
const StatusBadLink = 700

// Scan the URL for broken links.
func Scan(opts Options) (result *Result, err error) {
	var logp = `Scan`

	err = opts.init()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	var wrk *worker

	wrk, err = newWorker(opts)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	result, err = wrk.run()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	err = wrk.cache.Save()
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
	}

	return result, nil
}
