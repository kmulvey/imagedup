package hash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"sync"

	"github.com/corona10/goimagehash"
	"github.com/prometheus/client_golang/prometheus"
)

// Cache stores a map of image hashes from corona10/goimagehash
type Cache struct {
	imageCacheHits   prometheus.Counter
	imageCacheMisses prometheus.Counter
	store            map[string]*Image
	storeFileName    string
	lock             sync.RWMutex
}

// Image is the minimal data needed to compare images and is held in-memory by HashCache.Cache
type Image struct {
	*goimagehash.ImageHash
	image.Config `json:"-"`
}

// hashExportType is a stripped down type with just the necessary data which is intended to be
// persisted to disk so we dont need to calculate the hash again.
type hashExportType map[string]uint64

// NewCache reads the given file to rebuild its map from the last time it was run.
// If the file does not exist, it will be created.
func NewCache(file, promNamespace string) (*Cache, error) {
	var hc = new(Cache)
	hc.store = make(map[string]*Image)
	hc.storeFileName = file
	hc.imageCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "image_hash_cache_hits",
		},
	)
	hc.imageCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "image_hash_cache_misses",
		},
	)
	prometheus.MustRegister(hc.imageCacheHits)
	prometheus.MustRegister(hc.imageCacheMisses)

	// try to open the file, if it doesnt exist, create it
	var f, err = os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0755)
	if err != nil {
		return nil, fmt.Errorf("HashCache error opening file: %s, err: %w", file, err)
	}
	if info, err := f.Stat(); err != nil {
		return hc, fmt.Errorf("HashCache error stating file: %s, err: %w", file, err)
	} else if info.Size() == 0 {
		return hc, nil
	}

	// load map to file
	var m = make(hashExportType)
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("HashCache error decoding json file: %s, err: %w", file, err)
	}

	for name, hash := range m {
		hc.store[name] = &Image{goimagehash.NewImageHash(hash, goimagehash.PHash), image.Config{}}
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("HashCache error closing file: %s, err: %w", file, err)
	}

	return hc, nil
}

// NumImages returns the number of images in the cache
func (h *Cache) Stats() (int, int) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	// encoding to get the size of the store is a bit clunky but it only
	// takes tens of millis for a len(store) of ~50k.
	b := new(bytes.Buffer)
	_ = json.NewEncoder(b).Encode(h.store) // dont care about errors, its just a stat
	return len(h.store), b.Len()
}

// GetHash gets the hash from cache or if it does not exist it calcs it
func (h *Cache) GetHash(file string) (*Image, error) {

	h.lock.RLock()
	var imgCache, ok = h.store[file]
	h.lock.RUnlock()

	if ok {
		h.imageCacheHits.Inc()
		return imgCache, nil
	} else {
		h.imageCacheMisses.Inc()
		var imgCache = new(Image)

		var fileHandle, err = os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("HashCache error opening file: %s, err: %w", file, err)
		}

		img, err := jpeg.Decode(fileHandle)
		if err != nil {
			return nil, fmt.Errorf("HashCache error decoding jpeg file: %s, err: %w", file, err)
		}

		imgCache.ImageHash, err = goimagehash.PerceptionHash(img)
		if err != nil {
			return nil, fmt.Errorf("HashCache error calculating hash for file: %s, err: %w", file, err)
		}

		_, err = fileHandle.Seek(0, 0) // reset file reader
		if err != nil {
			return nil, fmt.Errorf("HashCache error rewinding file: %s, err: %w", file, err)
		}

		imgCache.Config, err = jpeg.DecodeConfig(fileHandle)
		if err != nil {
			return nil, fmt.Errorf("HashCache error decoding jpeg config file: %s, err: %w", file, err)
		}

		h.lock.Lock()
		h.store[file] = imgCache
		h.lock.Unlock()

		if err = fileHandle.Close(); err != nil {
			return nil, fmt.Errorf("HashCache error closing file: %s, err: %w", file, err)
		}

		return imgCache, nil
	}
}

// Persist writes the cache to disk
// https://pkg.go.dev/github.com/corona10/goimagehash#ImageHash.Dump
func (h *Cache) Persist() error {

	var f, err = os.Create(h.storeFileName)
	if err != nil {
		return fmt.Errorf("HashCache error creating file: %s, err: %w", h.storeFileName, err)
	}

	// dump map to file
	h.lock.Lock()
	var m = make(hashExportType)
	for name, hash := range h.store {
		m[name] = hash.GetHash()
	}
	h.lock.Unlock()

	err = json.NewEncoder(f).Encode(m)
	if err != nil {
		return fmt.Errorf("HashCache error json encoding file: %s, err: %w", h.storeFileName, err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("HashCache error closing file: %s, err: %w", h.storeFileName, err)
	}

	prometheus.Unregister(h.imageCacheHits)
	prometheus.Unregister(h.imageCacheMisses)

	return nil
}
