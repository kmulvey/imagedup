package diffpool

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

/* this type of test will be flappy in ci/cd unfortunatly
func TestDiff(t *testing.T) {
	var result = testing.Benchmark(BenchmarkDiff)
	assert.True(t, result.NsPerOp() < 15000)
	assert.True(t, float64(result.N)/result.T.Seconds() > 75000)
}
*/

func BenchmarkDiff(b *testing.B) {
	// setup deps
	var pairChan = make(chan pair)
	var cache, err = NewHashCache("BenchmarkCacheFull.json")
	assert.NoError(b, err)
	defer assert.NoError(b, os.Remove("BenchmarkCacheFull.json"))

	var logger = logrus.New()
	logger.SetOutput(new(bytes.Buffer))
	var dp = NewDiffPool(context.Background(), 1, 10, pairChan, cache, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pairChan <- pair{One: "testimages/iceland.jpg", Two: "testimages/iceland.jpg"}
	}

	dp.wait()
}
