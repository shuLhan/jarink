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

	"git.sr.ht/~shulhan/jarink"
)

func main() {
	log.SetFlags(0)

	var optVerbose bool

	flag.BoolVar(&optVerbose, `verbose`, false,
		`print additional information while running`)

	flag.Parse()

	var cmd = flag.Arg(0)
	if cmd == "" {
		goto invalid_command
	}

	cmd = strings.ToLower(cmd)
	if cmd == "brokenlinks" {
		var brokenlinksOpts = jarink.BrokenlinksOptions{
			Url:       flag.Arg(1),
			IsVerbose: optVerbose,
		}
		if brokenlinksOpts.Url == "" {
			goto invalid_command
		}

		var result *jarink.BrokenlinksResult
		var err error
		result, err = jarink.Brokenlinks(brokenlinksOpts)
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
	log.Println(`= Jarink

Jarink is a program to help web administrator to maintains their website.

== Synopsis

	jarink [OPTIONS] <COMMAND> <args...>

Available commands,

	brokenlinks - scan the website for broken links (page and images).

== Usage

[OPTIONS] brokenlinks URL

	Start scanning for broken links on the web server pointed by URL.
	Invalid links will be scanned on anchor href attribute
	("<a href=...>") or on the image src attribute ("<img src=...").

	Once finished it will print the page and list of broken links inside
	that page in JSON format.

	This command accept the following options,

		-verbose : print the page that being scanned.

	Example,

	$ jarink scan https://kilabit.info
	{
	  "https://kilabit.info/some/page": [
	    {
	      "Link": "https://kilabit.info/some/page/image.png",
	      "Code": 404
	    },
	    {
	      "Link": "https://external.com/link",
	      "Error": "Internal server error",
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
	      "Error": "Internal server error",
	      "Code": 500
	    }
	  ]
	}

--
jarink v` + jarink.Version)
}
