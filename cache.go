package main

import (
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"sync"

	"github.com/kmulvey/goimagehash"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	imageCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "image_hash_cache_hits",
		},
	)
	imageCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "image_hash_cache_misses",
		},
	)
)

func init() {
	prometheus.MustRegister(imageCacheHits)
	prometheus.MustRegister(imageCacheMisses)
}

type imageCache struct {
	*goimagehash.ImageHash
	image.Config `json:"-"`
}

type hashCache struct {
	Cache map[string]*imageCache
	lock  sync.RWMutex
}

type HashExportType struct {
	Hash uint64
	Kind goimagehash.Kind
}

func NewHashCache(file string) (*hashCache, error) {
	var hc = new(hashCache)
	hc.Cache = make(map[string]*imageCache)

	// try to open the file, if it doesnt exist, create it
	var f, err = os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(file)
			return hc, err
		} else {
			return nil, err
		}
	}

	// load map to file
	var m = make(map[string]HashExportType)
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

func (h *hashCache) NumImages() int {
	h.lock.Lock()
	defer h.lock.Unlock()

	return len(h.Cache)
}

func (h *hashCache) GetHash(file string) (*imageCache, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	var imgCache, ok = h.Cache[file]
	if ok {
		imageCacheHits.Inc()
		return imgCache, nil
	} else {
		imageCacheMisses.Inc()
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

		fileHandle.Seek(0, 0) // reset file reader
		imgCache.Config, err = jpeg.DecodeConfig(fileHandle)
		if err != nil {
			return imgCache, err
		}

		h.Cache[file] = imgCache
		return imgCache, fileHandle.Close()
	}
}

// https://pkg.go.dev/github.com/corona10/goimagehash#ImageHash.Dump
func (h *hashCache) Persist(file string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	var f, err = os.Create(file)
	if err != nil {
		return err
	}

	// dump map to file
	var m = make(map[string]HashExportType)
	for name, hash := range h.Cache {
		m[name] = HashExportType{Hash: hash.GetHash(), Kind: hash.GetKind()}
	}

	err = json.NewEncoder(f).Encode(m)
	if err != nil {
		return err
	}

	return f.Close()
}
