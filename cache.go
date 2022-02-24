package main

import (
	"image"
	"image/jpeg"
	"os"
	"sync"
	"unsafe"

	"github.com/kmulvey/goimagehash"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	imageCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_hits",
		},
	)
	imageCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_misses",
		},
	)
)

func init() {
	prometheus.MustRegister(imageCacheHits)
	prometheus.MustRegister(imageCacheMisses)
}

type imageCache struct {
	*goimagehash.ImageHash
	image.Config
}

type hashCache struct {
	cache map[string]imageCache
	lock  sync.RWMutex
}

func NewHashCache() *hashCache {
	return &hashCache{cache: make(map[string]imageCache)}
}

func (h *hashCache) Size() int {
	h.lock.Lock()
	defer h.lock.Unlock()

	var total int
	for _, img := range h.cache {
		total += int(unsafe.Sizeof(img.ImageHash))
		total += int(unsafe.Sizeof(img.Config.ColorModel))
		total += int(unsafe.Sizeof(img.Config.Height))
		total += int(unsafe.Sizeof(img.Config.Width))
	}

	return total
}

func (h *hashCache) NumImages() int {
	h.lock.Lock()
	defer h.lock.Unlock()

	return len(h.cache)
}

func (h *hashCache) GetHash(file string) (imageCache, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	var imgCache = imageCache{}
	var ok bool

	imgCache, ok = h.cache[file]
	if ok {
		imageCacheHits.Inc()
		return imgCache, nil
	} else {
		imageCacheMisses.Inc()
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

		h.cache[file] = imgCache
		return imgCache, fileHandle.Close()
	}
}
