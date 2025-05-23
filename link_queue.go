// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package deadlinks

import "net/url"

type linkQueue struct {
	parentUrl *url.URL
	url       string
}
