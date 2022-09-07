package imagedup

import (
	"runtime"
	"time"

	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/prometheus/client_golang/prometheus"
)

type Stats struct {
	PromNamespace        string
	DiffTime             prometheus.Gauge
	PairTotal            prometheus.Counter
	GCTime               prometheus.Gauge
	TotalComparisons     prometheus.Gauge
	ComparisonsCompleted prometheus.Gauge
	ImageCacheSize       prometheus.Gauge
	ImageCacheNumImages  prometheus.Gauge
	PairCacheSize        prometheus.Gauge
}

func NewStats(promNamespace string) *Stats {
	var s = new(Stats)
	s.PromNamespace = promNamespace

	s.DiffTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "diff_time_nano",
			Help:      "How long it takes to diff two images, in nanoseconds.",
		},
	)
	s.PairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	s.GCTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "gc_time_nano",
		},
	)
	s.TotalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_comparisons",
		},
	)
	s.ComparisonsCompleted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "comparisons_completed",
		},
	)
	s.ImageCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_size_bytes",
		},
	)
	s.ImageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_num_images",
		},
	)
	s.PairCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "pair_cache_size",
		},
	)
	prometheus.MustRegister(s.DiffTime)
	prometheus.MustRegister(s.PairTotal)
	prometheus.MustRegister(s.GCTime)
	prometheus.MustRegister(s.TotalComparisons)
	prometheus.MustRegister(s.ComparisonsCompleted)
	prometheus.MustRegister(s.ImageCacheSize)
	prometheus.MustRegister(s.ImageCacheNumImages)
	prometheus.MustRegister(s.PairCacheSize)

	return s
}

// PublishStats publishes go GC stats + cache size to prom every 10 seconds
func (s *Stats) PublishStats(imageCache *cache.HashCache) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		s.GCTime.Set(float64(stats.PauseTotalNs))

		s.ImageCacheNumImages.Set(float64(imageCache.NumImages()))

		time.Sleep(10 * time.Second)
	}
}
