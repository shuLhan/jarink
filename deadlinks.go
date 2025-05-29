// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import (
	"fmt"
)

// Scan the baseUrl for dead links.
func Scan(opts ScanOptions) (result *Result, err error) {
	var logp = `Scan`
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
