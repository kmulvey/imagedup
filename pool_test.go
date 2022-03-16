package main

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	var result = testing.Benchmark(BenchmarkDiff)
	assert.True(t, result.NsPerOp() < 800)
	assert.True(t, float64(result.N)/result.T.Seconds() > 1e6)
}

func BenchmarkDiff(b *testing.B) {
	// setup deps
	var ctx, _ = context.WithCancel(context.Background())
	var pairChan = make(chan pair)
	var cache, err = NewHashCache("BenchmarkCacheFull.json")
	assert.NoError(b, err)
	defer assert.NoError(b, os.Remove("BenchmarkCacheFull.json"))

	var dp = NewDiffPool(ctx, 1, pairChan, cache, logrus.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pairChan <- pair{One: "testimages/iceland.jpg", Two: "testimages/iceland.jpg"}
	}

	dp.wait()
}
