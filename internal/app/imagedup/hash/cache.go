package hash

import (
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
	globPattern      string
	storeFileName    string
	store            []*Image
	lock             sync.RWMutex
}

// Image is the minimal data needed to compare images and is held in-memory by HashCache.Cache
type Image struct {
	*goimagehash.ImageHash
	image.Config `json:"-"`
}

// hashExportType is a stripped down type with just the necessary data which is intended to be
// persisted to disk so we dont need to calculate the hash again.
type hashExportType struct {
	GlobPattern string
	Hashes      []uint64
}

// NewCache reads the given file to rebuild its map from the last time it was run.
// If the file does not exist, it will be created.
func NewCache(file, globPattern, promNamespace string, numFiles int) (*Cache, error) {
	var c = new(Cache)
	c.store = make([]*Image, numFiles)
	c.storeFileName = file
	c.globPattern = globPattern
	c.imageCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "image_hash_cache_hits",
		},
	)
	c.imageCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "image_hash_cache_misses",
		},
	)
	prometheus.MustRegister(c.imageCacheHits)
	prometheus.MustRegister(c.imageCacheMisses)

	// try to open the file, if it doesnt exist, create it
	var f, err = os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0755)
	if err != nil {
		return nil, fmt.Errorf("HashCache error opening file: %s, err: %w", file, err)
	}
	if info, err := f.Stat(); err != nil {
		return c, fmt.Errorf("HashCache error stating file: %s, err: %w", file, err)
	} else if info.Size() == 0 {
		return c, nil
	}

	// load array from file
	var m = new(hashExportType)
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("HashCache error decoding json file: %s, err: %w", file, err)
	}

	if len(m.Hashes) > 0 {
		if globPattern != m.GlobPattern {
			return nil, fmt.Errorf("Previous glob: %s from file: %s does not match new glob: %s, please specify a new cache file", m.GlobPattern, file, globPattern)
		}

		for i, hash := range m.Hashes {
			c.store[i] = &Image{goimagehash.NewImageHash(hash, goimagehash.PHash), image.Config{}}
		}
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("HashCache error closing file: %s, err: %w", file, err)
	}

	return c, nil
}

// NumImages returns the number of images in the cache
func (c *Cache) Stats() (int, int) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var length = len(c.store)
	return length, length * 48
}

// GetHash gets the hash from cache or if it does not exist it calcs it
func (c *Cache) GetHash(fileIndex int, fileName string) (*Image, error) {

	c.lock.RLock()
	var imgData = c.store[fileIndex]
	c.lock.RUnlock()

	if imgData != nil {
		c.imageCacheHits.Inc()
		return imgData, nil
	} else {
		c.imageCacheMisses.Inc()
		var imgCache = new(Image)

		var fileHandle, err = os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("HashCache error opening file: %s, err: %w", fileName, err)
		}

		img, err := jpeg.Decode(fileHandle)
		if err != nil {
			return nil, fmt.Errorf("HashCache error decoding jpeg file: %s, err: %w", fileName, err)
		}

		imgCache.ImageHash, err = goimagehash.PerceptionHash(img)
		if err != nil {
			return nil, fmt.Errorf("HashCache error calculating hash for file: %s, err: %w", fileName, err)
		}

		_, err = fileHandle.Seek(0, 0) // reset file reader
		if err != nil {
			return nil, fmt.Errorf("HashCache error rewinding file: %s, err: %w", fileName, err)
		}

		imgCache.Config, err = jpeg.DecodeConfig(fileHandle)
		if err != nil {
			return nil, fmt.Errorf("HashCache error decoding jpeg config file: %s, err: %w", fileName, err)
		}

		c.lock.Lock()
		c.store[fileIndex] = imgCache
		c.lock.Unlock()

		if err = fileHandle.Close(); err != nil {
			return nil, fmt.Errorf("HashCache error closing file: %s, err: %w", fileName, err)
		}

		return imgCache, nil
	}
}

// Persist writes the cache to disk
func (c *Cache) Persist() error {

	var f, err = os.Create(c.storeFileName)
	if err != nil {
		return fmt.Errorf("HashCache error creating file: %s, err: %w", c.storeFileName, err)
	}

	// dump map to file
	c.lock.Lock()
	var m = new(hashExportType)
	m.GlobPattern = c.globPattern
	m.Hashes = make([]uint64, len(c.store))
	for i, hash := range c.store {
		m.Hashes[i] = hash.GetHash()
	}
	c.lock.Unlock()

	err = json.NewEncoder(f).Encode(m)
	if err != nil {
		return fmt.Errorf("HashCache error json encoding file: %s, err: %w", c.storeFileName, err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("HashCache error closing file: %s, err: %w", c.storeFileName, err)
	}

	prometheus.Unregister(c.imageCacheHits)
	prometheus.Unregister(c.imageCacheMisses)

	return nil
}
