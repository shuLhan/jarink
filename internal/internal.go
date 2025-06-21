// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

// CacheFile return the path to cache file under [os.UserCacheDir] +
// "jarink" directory.
// This variable defined here so the test file can override it.
var CacheFile = DefaultCacheFile

func DefaultCacheFile() (cacheFile string, err error) {
	var logp = `DefaultCacheFile`
	var cacheDir string

	cacheDir, err = os.UserCacheDir()
	if err != nil {
		return ``, fmt.Errorf(`%s: %w`, logp, err)
	}
	cacheDir = filepath.Join(cacheDir, `jarink`)

	err = os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return ``, fmt.Errorf(`%s: %w`, logp, err)
	}

	cacheFile = filepath.Join(cacheDir, `cache.json`)
	return cacheFile, nil
}
