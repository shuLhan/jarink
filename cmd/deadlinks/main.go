// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"encoding/json"
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

		var resultJson []byte
		resultJson, err = json.MarshalIndent(result.PageLinks, ``, `  `)
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("%s\n", resultJson)
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

[OPTIONS] scan URL

	Start scanning for deadlinks on the web server pointed by URL.
	Once finished it will print the page and list of dead links inside
	that page in JSON format.
	This command accept the following options,

	-verbose : print the page that being scanned.

	Example,

	$ deadlinks scan https://kilabit.info
	{
	  "https://kilabit.info/some/page": [
	    {
	      "Link": "https://kilabit.info/some/page/image.png",
	      "Code": 404
	    },
	    {
	      "Link": "https://external.com/link",
	      "Code": 500
	    }
	  ],
	  "https://kilabit.info/another/page": [
	    {
	      "Link": "https://kilabit.info/another/page/image.png",
	      "Code": 404
	    },
	    {
	      "Link": "https://external.org/link",
	      "Code": 500
	    }
	  ]
	}

--
deadlinks v` + deadlinks.Version)
}
