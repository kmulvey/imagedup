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

// Cache stores an array of image hashes from corona10/goimagehash
type Cache struct {
	imageCacheHits   prometheus.Counter
	imageCacheMisses prometheus.Counter
	storeFileName    string
	store            map[string]*Image
	lock             sync.RWMutex
}

// Image is the minimal data needed to compare images and is held in-memory by HashCache.Cache
type Image struct {
	*goimagehash.ImageHash
	image.Config `json:"-"`
}

// NewCache reads the given file to rebuild its map from the last time it was run.
// If the file does not exist, it will be created.
func NewCache(cacheFileName, promNamespace string, numFiles int) (*Cache, error) {
	var c = new(Cache)
	c.store = make(map[string]*Image, numFiles)
	c.storeFileName = cacheFileName
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
	var f, err = os.OpenFile(cacheFileName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("HashCache error opening file: %s, err: %w", cacheFileName, err)
	}
	if info, err := f.Stat(); err != nil {
		return c, fmt.Errorf("HashCache error stating file: %s, err: %w", cacheFileName, err)
	} else if info.Size() == 0 {
		return c, nil
	}

	// load array from file
	var m = make(map[string]uint64, len(c.store))
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("HashCache error decoding json file: %s, err: %w", cacheFileName, err)
	}

	if len(m) > 0 {
		for imageName, hash := range m {
			c.store[imageName] = &Image{goimagehash.NewImageHash(hash, goimagehash.PHash), image.Config{}}
		}
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("HashCache error closing file: %s, err: %w", cacheFileName, err)
	}

	return c, nil
}

// Stats returns the number of images in the cache
func (c *Cache) Stats() (int, int) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var length = len(c.store)
	return length, length * 48
}

// GetHash gets the hash from cache or if it does not exist it calcs it
func (c *Cache) GetHash(fileName string) (*Image, error) {

	c.lock.RLock()
	var imgData = c.store[fileName]
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
		c.store[fileName] = imgCache
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
	var m = make(map[string]uint64, len(c.store))
	for file, hash := range c.store {
		m[file] = hash.GetHash()
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
