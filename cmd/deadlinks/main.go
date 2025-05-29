// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"git.sr.ht/~shulhan/deadlinks"
)

func main() {
	var optVerbose bool

	flag.BoolVar(&optVerbose, `verbose`, false,
		`print additional information while running`)

	flag.Parse()

	var cmd = flag.Arg(0)
	if cmd == "" {
		goto invalid_command
	}

	cmd = strings.ToLower(cmd)
	if cmd == "scan" {
		var scanOpts = deadlinks.ScanOptions{
			Url:       flag.Arg(1),
			IsVerbose: optVerbose,
		}
		if scanOpts.Url == "" {
			goto invalid_command
		}

		var result *deadlinks.Result
		var err error
		result, err = deadlinks.Scan(scanOpts)
		if err != nil {
			log.Fatal(err.Error())
		}

		var page string
		var listBroken []deadlinks.Broken
		for page, listBroken = range result.PageLinks {
			fmt.Printf("Page: %s\n", page)
			for _, broken := range listBroken {
				fmt.Printf("\tDead: %s (%d)\n", broken.Link,
					broken.Code)
			}
		}
		return
	}

invalid_command:
	usage()
	os.Exit(1)
}

func usage() {
	log.Println(`
deadlinks <COMMAND> <args...>

Deadlinks is a program to scan for invalid links inside HTML page on the live
web server.
Invalid links will be scanned on anchor href attribute ("<a href=...>") or on
the image src attribute ("<img src=...").

== Usage

scan URL [OPTIONS]

	Start scanning for deadlinks on the web server pointed by URL.
	Once finished it will print the page and list of dead links inside
	that page.
	This command accept the following options,

	-verbose : print the page that being scanned.

	Example,

	$ deadlinks scan https://kilabit.info
	Page: https://kilabit.info/some/page
		Dead: https://kilabit.info/some/page/image.png (404)
		Dead: https://external.com/link (500)
	Page: https://kilabit.info/another/page
		Dead: https://kilabit.info/another/page/image.png (404)
		Dead: https://external.org/link (500)`)
}
