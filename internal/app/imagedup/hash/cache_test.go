package hash

import (
	"os"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, "TestCache")
	assert.NoError(t, err)

	files, err := path.ListFiles("../testimages")
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(path.OnlyFiles(files))

	for _, file := range fileNames {
		_, err = cache.GetHash(file)
		assert.NoError(t, err)
	}
	assert.Equal(t, 3, cache.NumImages())

	_, err = cache.GetHash(fileNames[0])
	assert.NoError(t, err)

	err = cache.Persist()
	assert.NoError(t, err)

	// do it again
	cache, err = NewCache(cacheFile, "TestCache2")
	assert.NoError(t, err)
	assert.Equal(t, 3, cache.NumImages())

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}