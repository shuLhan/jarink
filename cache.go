// SPDX-FileCopyrightText: 2025 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-only

package jarink

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"git.sr.ht/~shulhan/jarink/internal"
)

// ScannedLink store information about the link.
type ScannedLink struct {
	Url          string `json:"url"`
	Size         int64  `json:"size"`
	ResponseCode int    `json:"response_code"`
}

// Cache store external links that has been scanned, to minize
// request to the same URL in the future.
// The cache is stored as JSON file under user's cache directory, inside
// "jarink" directory.
// For example, in Linux it should be "$HOME/.cache/jarink/cache.json".
// See [os.UserCacheDir] for location specific to operating system.
type Cache struct {
	ScannedLinks map[string]*ScannedLink `json:"scanned_links"`
	file         string
	mtx          sync.Mutex
}

// LoadCache from local storage.
func LoadCache() (cache *Cache, err error) {
	var logp = `LoadCache`

	cache = &Cache{
		ScannedLinks: map[string]*ScannedLink{},
	}

	cache.file, err = internal.CacheFile()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	var cacheJson []byte
	cacheJson, err = os.ReadFile(cache.file)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	err = json.Unmarshal(cacheJson, &cache)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return cache, nil
}

// Get return the scanned link information by url.
func (cache *Cache) Get(url string) (scannedLink *ScannedLink) {
	cache.mtx.Lock()
	scannedLink = cache.ScannedLinks[url]
	cache.mtx.Unlock()
	return scannedLink
}

// Save the cache into local storage.
func (cache *Cache) Save() (err error) {
	var logp = `Save`
	var cacheJson []byte
	cacheJson, err = json.MarshalIndent(cache, ``, `  `)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	cacheJson = append(cacheJson, '\n')

	err = os.WriteFile(cache.file, cacheJson, 0600)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}
	return nil
}

func (cache *Cache) Set(url string, respCode int, size int64) {
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	var scannedLink = cache.ScannedLinks[url]
	if scannedLink != nil {
		return
	}
	scannedLink = &ScannedLink{
		Url:          url,
		Size:         size,
		ResponseCode: respCode,
	}
	cache.ScannedLinks[url] = scannedLink
}
