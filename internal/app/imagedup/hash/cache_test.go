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
	assert.NotNil(t, cache)

	dirs, err := path.List("../testimages", 1, false, path.NewFileEntitiesFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(dirs)

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

func TestBadCacheFile(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestBadCacheFile.json"
	assert.NoError(t, os.WriteFile(cacheFile, []byte("not json"), 0600))

	var cache, err = NewCache(cacheFile, "TestBadCacheFile", 3)
	assert.Equal(t, "HashCache error decoding json file: TestBadCacheFile.json, err: invalid character 'o' in literal null (expecting 'u')", err.Error())
	assert.Nil(t, cache)

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}

func BenchmarkGetHash(b *testing.B) {

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, goutils.RandomString(5), 3)
	assert.NoError(b, err)

	dirs, err := path.List("../testimages", 1, false, path.NewFileEntitiesFilter())
	assert.NoError(b, err)
	var fileNames = path.OnlyNames(dirs)

	for _, file := range dirs {
		_, err = cache.GetHash(file.AbsolutePath)
		assert.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = cache.GetHash(fileNames[0])
		assert.NoError(b, err)
	}

	err = os.RemoveAll(cacheFile)
	assert.NoError(b, err)
}
