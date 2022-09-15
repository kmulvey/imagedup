package hash

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/imagedup/pkg/imagedup/types"
	"github.com/prometheus/client_golang/prometheus"
)

// Differ
type Differ struct {
	diffTime             prometheus.Gauge
	comparisonsCompleted prometheus.Gauge
	inputImages          chan types.Pair
	cache                *Cache
	numWorkers           int
	distanceThreshold    int
}

// DiffResult are two images that are the "same", i.e. within the given distance
type DiffResult struct {
	One     string
	Two     string
	OneArea int
	TwoArea int
}

func NewDiffer(numWorkers, distanceThreshold int, inputImages chan types.Pair, cache *Cache, promNamespace string) *Differ {

	if numWorkers <= 0 || numWorkers > runtime.GOMAXPROCS(0)-1 {
		numWorkers = 1
	}

	var dp = &Differ{
		inputImages:       inputImages,
		cache:             cache,
		distanceThreshold: distanceThreshold,
		numWorkers:        numWorkers,
		diffTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: promNamespace,
				Name:      "diff_time_nano",
				Help:      "How long it takes to diff two images, in nanoseconds.",
			}),
		comparisonsCompleted: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: promNamespace,
				Name:      "comparisons_completed",
			}),
	}
	prometheus.MustRegister(dp.diffTime)
	prometheus.MustRegister(dp.comparisonsCompleted)

	return dp
}

func (dp *Differ) Run(ctx context.Context) (chan DiffResult, chan error) {
	var errorChans = make([]chan error, dp.numWorkers)
	var resultChans = make([]chan DiffResult, dp.numWorkers)

	for i := 0; i < dp.numWorkers; i++ {
		var errors = make(chan error)
		var results = make(chan DiffResult)
		errorChans[i] = errors
		resultChans[i] = results
		go dp.diffWorker(ctx, results, errors)
	}

	return goutils.MergeChannels(resultChans...), goutils.MergeChannels(errorChans...)
}

func (dp *Differ) diffWorker(ctx context.Context, results chan DiffResult, errors chan error) {

	// declare these here to reduce allocations in the loop
	var start time.Time
	var imgCacheOne, imgCacheTwo *Image
	var err error
	var distance int

	for {
		select {
		case <-ctx.Done():
			close(errors)
			close(results)
			return
		default:
			p, open := <-dp.inputImages
			if !open {
				close(errors)
				close(results)
				return
			}
			start = time.Now()

			imgCacheOne, err = dp.cache.GetHash(p.One)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.One, err)
				continue
			}

			imgCacheTwo, err = dp.cache.GetHash(p.Two)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			distance, err = imgCacheOne.ImageHash.Distance(imgCacheTwo.ImageHash)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			if distance <= dp.distanceThreshold {
				results <- DiffResult{One: p.One, OneArea: imgCacheOne.Config.Height * imgCacheOne.Config.Width, Two: p.Two, TwoArea: imgCacheTwo.Config.Height * imgCacheTwo.Config.Width}
			}

			dp.diffTime.Set(float64(time.Since(start)))
			dp.comparisonsCompleted.Inc()
		}
	}
}
