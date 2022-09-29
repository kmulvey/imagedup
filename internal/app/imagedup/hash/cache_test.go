package hash

import (
	"os"
	"testing"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, "TestCache", 3)
	assert.NoError(t, err)

	files, err := path.ListFiles("../testimages")
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(path.OnlyFiles(files))

	for _, file := range fileNames {
		_, err = cache.GetHash(file)
		assert.NoError(t, err)
	}
	var numImages, _ = cache.Stats()
	assert.Equal(t, 3, numImages)

	_, err = cache.GetHash(fileNames[0])
	assert.NoError(t, err)

	err = cache.Persist()
	assert.NoError(t, err)

	// do it again
	cache, err = NewCache(cacheFile, "TestCache2", 3)
	assert.NoError(t, err)
	numImages, _ = cache.Stats()
	assert.Equal(t, 3, numImages)

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}

func BenchmarkGetHash(b *testing.B) {

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, goutils.RandomString(5), 3)
	assert.NoError(b, err)

	files, err := path.ListFiles("../testimages")
	assert.NoError(b, err)
	var fileNames = path.OnlyNames(path.OnlyFiles(files))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = cache.GetHash(fileNames[0])
		assert.NoError(b, err)
	}
}
