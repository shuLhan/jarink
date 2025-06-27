// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink

import (
	_ "embed"
)

// Version of jarink program and module.
var Version = `0.2.0`

// GoEmbedReadme embed the README for showing the usage of program.
//
//go:embed README
var GoEmbedReadme string
