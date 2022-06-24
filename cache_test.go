package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/* this type of test will be flappy in ci/cd unfortunatly
func TestCacheFull(t *testing.T) {
	var result = testing.Benchmark(BenchmarkCacheFull)
	assert.True(t, result.NsPerOp() < 500)
	assert.True(t, float64(result.N)/result.T.Seconds() > 4e7)
}
*/

func BenchmarkCacheFull(b *testing.B) {
	var c, err = NewHashCache("BenchmarkCacheFull.json")
	assert.NoError(b, err)
	defer assert.NoError(b, os.Remove("BenchmarkCacheFull.json"))

	_, err = c.GetHash("testimages/iceland.jpg")
	assert.NoError(b, err)

	assert.Equal(b, 1, c.NumImages())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = c.GetHash("testimages/iceland.jpg")
		assert.NoError(b, err)
	}
}

func TestCacheEmpty(t *testing.T) {
	var c, err = NewHashCache("TestCacheEmpty.json")
	assert.NoError(t, err)
	defer assert.NoError(t, os.Remove("TestCacheEmpty.json"))

	var start = time.Now()
	_, err = c.GetHash("testimages/iceland.jpg")
	assert.NoError(t, err)
	assert.True(t, time.Since(start) < 300*time.Millisecond)

	assert.Equal(t, 1, c.NumImages())
}
