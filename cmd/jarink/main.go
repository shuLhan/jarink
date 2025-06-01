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
	cmd = strings.ToLower(cmd)
	switch cmd {
	case `brokenlinks`:
		var brokenlinksOpts = jarink.BrokenlinksOptions{
			Url:       flag.Arg(1),
			IsVerbose: optVerbose,
		}
		if brokenlinksOpts.Url == "" {
			log.Printf(`Missing argument URL to be scanned.`)
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

	case `help`:
		log.Println(jarink.GoEmbedReadme)
		return

	default:
		log.Printf(`Missing or invalid command %q`, cmd)
	}

invalid_command:
	log.Printf(`Run "jarink help" for usage.`)
	os.Exit(1)
}
