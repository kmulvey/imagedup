package hash

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/imagedup/v2/pkg/imagedup/types"
	"github.com/prometheus/client_golang/prometheus"
)

// Differ diffs images
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

// NewDiffer is the constructor, Run() must be called to start diffing
func NewDiffer(numWorkers, distanceThreshold int, inputImages chan types.Pair, cache *Cache, promNamespace string) *Differ {

	if numWorkers <= 0 || numWorkers > runtime.GOMAXPROCS(0)-1 {
		numWorkers = 1
	}

	var d = &Differ{
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
	prometheus.MustRegister(d.diffTime)
	prometheus.MustRegister(d.comparisonsCompleted)

	return d
}

// Shutdown unregisters prom stats
func (d *Differ) Shutdown() {
	prometheus.Unregister(d.diffTime)
	prometheus.Unregister(d.comparisonsCompleted)
}

// Run starts the diff workers
func (d *Differ) Run(ctx context.Context) (chan DiffResult, chan error) {
	var errorChans = make([]chan error, d.numWorkers)
	var resultChans = make([]chan DiffResult, d.numWorkers)

	for i := 0; i < d.numWorkers; i++ {
		var errors = make(chan error)
		var results = make(chan DiffResult)
		errorChans[i] = errors
		resultChans[i] = results
		go d.diffWorker(ctx, results, errors)
	}

	return goutils.MergeChannels(resultChans...), goutils.MergeChannels(errorChans...)
}

// diffWorker compares two imamges to determine if they are similar.
func (d *Differ) diffWorker(ctx context.Context, results chan DiffResult, errors chan error) {

	// declare these here to reduce allocations in the loop
	var start time.Time
	var imgCacheOne, imgCacheTwo *Image
	var err error
	var distance int
	var p types.Pair
	var open bool

	for {
		select {
		case <-ctx.Done():
			close(errors)
			close(results)
			return
		default:
			p, open = <-d.inputImages
			if !open {
				close(errors)
				close(results)
				return
			}
			start = time.Now()

			imgCacheOne, err = d.cache.GetHash(p.I, p.One)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.One, err)
				continue
			}

			imgCacheTwo, err = d.cache.GetHash(p.J, p.Two)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			distance, err = imgCacheOne.ImageHash.Distance(imgCacheTwo.ImageHash)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			if distance <= d.distanceThreshold {
				results <- DiffResult{One: p.One, OneArea: imgCacheOne.Config.Height * imgCacheOne.Config.Width, Two: p.Two, TwoArea: imgCacheTwo.Config.Height * imgCacheTwo.Config.Width}
			}

			d.diffTime.Set(float64(time.Since(start)))
			d.comparisonsCompleted.Inc()
		}
	}
}
