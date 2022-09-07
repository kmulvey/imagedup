package cache

import (
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"sync"

	"github.com/corona10/goimagehash"
	"github.com/prometheus/client_golang/prometheus"
)

// HashCache stores a map of image hashes from corona10/goimagehash
type HashCache struct {
	Cache            map[string]*imageCache
	lock             sync.RWMutex
	imageCacheHits   prometheus.Counter
	imageCacheMisses prometheus.Counter
}

type imageCache struct {
	*goimagehash.ImageHash
	image.Config `json:"-"`
}

// hashExportType is a stripped down type with just the necessary data which is intended to be
// persisted to disk so we dont need to calculate the hash again.
type hashExportType struct {
	Hash uint64
	Kind goimagehash.Kind
}

// NewHashCache reads the given file to rebuild its map from the last time it was run.
// If the file does not exist, it will be created.
func NewHashCache(file, promNamespace string) (*HashCache, error) {
	var hc = new(HashCache)
	hc.Cache = make(map[string]*imageCache)
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
	var f, err = os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(file)
			return hc, err
		} else {
			return nil, err
		}
	}
	if info, err := f.Stat(); err != nil {
		return hc, err
	} else if info.Size() == 0 {
		return hc, nil
	}

	// load map to file
	var m = make(map[string]hashExportType)
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, err
	}

	for name, hash := range m {
		hc.Cache[name] = &imageCache{goimagehash.NewImageHash(hash.Hash, hash.Kind), image.Config{}}
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return hc, nil
}

// NumImages returns the number of images in the cache
func (h *HashCache) NumImages() int {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return len(h.Cache)
}

// GetHash gets the hash from cache or if it does not exist it calcs it
func (h *HashCache) GetHash(file string) (*imageCache, error) {

	h.lock.RLock()
	var imgCache, ok = h.Cache[file]
	h.lock.RUnlock()
	if ok {
		h.imageCacheHits.Inc()
		return imgCache, nil
	} else {
		h.imageCacheMisses.Inc()
		var imgCache = new(imageCache)

		var fileHandle, err = os.Open(file)
		if err != nil {
			return imgCache, err
		}

		img, err := jpeg.Decode(fileHandle)
		if err != nil {
			return imgCache, err
		}

		imgCache.ImageHash, err = goimagehash.PerceptionHash(img)
		if err != nil {
			return imgCache, err
		}

		_, err = fileHandle.Seek(0, 0) // reset file reader
		if err != nil {
			return imgCache, err
		}

		imgCache.Config, err = jpeg.DecodeConfig(fileHandle)
		if err != nil {
			return imgCache, err
		}

		h.lock.Lock()
		h.Cache[file] = imgCache
		h.lock.Unlock()
		return imgCache, fileHandle.Close()
	}
}

// Persist writes the cache to disk
// https://pkg.go.dev/github.com/corona10/goimagehash#ImageHash.Dump
func (h *HashCache) Persist(file string) error {

	var f, err = os.Create(file)
	if err != nil {
		return err
	}

	// dump map to file
	h.lock.Lock()
	var m = make(map[string]hashExportType)
	for name, hash := range h.Cache {
		m[name] = hashExportType{Hash: hash.GetHash(), Kind: hash.GetKind()}
	}
	h.lock.Unlock()

	err = json.NewEncoder(f).Encode(m)
	if err != nil {
		return err
	}

	return f.Close()
}
