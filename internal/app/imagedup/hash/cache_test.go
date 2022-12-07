package hash

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, "glob", "TestCache", 3)
	assert.NoError(t, err)
	assert.NotNil(t, cache)

	dirs, err := path.List("../testimages", path.NewFileListFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(dirs)

	for i, file := range fileNames {
		_, err = cache.GetHash(i, file)
		assert.NoError(t, err)
	}
	var numImages, _ = cache.Stats()
	assert.Equal(t, 3, numImages)

	_, err = cache.GetHash(0, fileNames[0])
	assert.NoError(t, err)

	err = cache.Persist()
	assert.NoError(t, err)

	// do it again
	cache, err = NewCache(cacheFile, "glob", "TestCache2", 3)
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

	var cache, err = NewCache(cacheFile, "glob", "TestBadCacheFile", 3)
	assert.Equal(t, "HashCache error decoding json file: TestBadCacheFile.json, err: invalid character 'o' in literal null (expecting 'u')", err.Error())
	assert.Nil(t, cache)

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}

func TestDifferentGlob(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestDifferentGlob.json"
	var fileData = hashExportType{
		GlobPattern: "glob",
		Hashes:      []uint64{0, 1, 2, 3, 4},
	}
	js, err := json.Marshal(fileData)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(cacheFile, js, 0600))

	cache, err := NewCache(cacheFile, "different glob", "TestDifferentGlob", 3)
	assert.Equal(t, "Previous glob: 'glob' from file: TestDifferentGlob.json does not match new glob: 'different glob', please specify a new cache file", err.Error())
	assert.Nil(t, cache)

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}

func BenchmarkGetHash(b *testing.B) {

	var cacheFile = "testcache.json"

	var cache, err = NewCache(cacheFile, "glob", goutils.RandomString(5), 3)
	assert.NoError(b, err)

	dirs, err := path.List("../testimages", path.NewFileListFilter())
	assert.NoError(b, err)
	var fileNames = path.OnlyNames(dirs)

	for i, file := range dirs {
		_, err = cache.GetHash(i, file.AbsolutePath)
		assert.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = cache.GetHash(0, fileNames[0])
		assert.NoError(b, err)
	}

	err = os.RemoveAll(cacheFile)
	assert.NoError(b, err)
}
