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
	"git.sr.ht/~shulhan/jarink/brokenlinks"
)

func main() {
	log.SetFlags(0)

	var (
		optIgnoreStatus string
		optInsecure     bool
		optIsVerbose    bool
		optPastResult   string
	)

	flag.StringVar(&optIgnoreStatus, `ignore-status`, ``,
		`Comma separated HTTP response status code to be ignored.`)

	flag.BoolVar(&optInsecure, `insecure`, false,
		`Do not report as error on server with invalid certificates.`)

	flag.BoolVar(&optIsVerbose, `verbose`, false,
		`Print additional information while running.`)

	flag.StringVar(&optPastResult, `past-result`, ``,
		`Scan only pages with broken links from the past JSON result.`)

	flag.Parse()

	var cmd = flag.Arg(0)
	cmd = strings.ToLower(cmd)
	switch cmd {
	case `brokenlinks`:
		var opts = brokenlinks.Options{
			IgnoreStatus:   optIgnoreStatus,
			Insecure:       optInsecure,
			IsVerbose:      optIsVerbose,
			PastResultFile: optPastResult,
		}

		opts.Url = flag.Arg(1)
		if opts.Url == "" {
			log.Printf(`Missing argument URL to be scanned.`)
			goto invalid_command
		}

		var (
			result *brokenlinks.Result
			err    error
		)
		result, err = brokenlinks.Scan(opts)
		if err != nil {
			log.Fatal(err.Error())
		}

		var resultJson []byte
		resultJson, err = json.MarshalIndent(result, ``, `  `)
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("%s\n", resultJson)
		return

	case `help`:
		log.Println(jarink.GoEmbedReadme)
		return

	case `version`:
		log.Println(jarink.Version)
		return

	default:
		log.Printf(`Missing or invalid command %q`, cmd)
	}

invalid_command:
	log.Printf(`Run "jarink help" for usage.`)
	os.Exit(1)
}
