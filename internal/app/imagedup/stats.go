package imagedup

import (
	"runtime"
	"sync"
	"time"

	"github.com/RoaringBitmap/roaring/roaring64"
	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/prometheus/client_golang/prometheus"
)

type stats struct {
	PairTotal           prometheus.Counter
	GCTime              prometheus.Gauge
	TotalFiles          prometheus.Gauge
	TotalComparisons    prometheus.Gauge
	ImageCacheBytes     prometheus.Gauge
	ImageCacheNumImages prometheus.Gauge
	FileMapBytes        prometheus.Gauge
	PromNamespace       string
}

func newStats(promNamespace string) *stats {
	var s = new(stats)
	s.PromNamespace = promNamespace

	s.GCTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "gc_time_nano",
			Help:      "how long a gc sweep took",
		},
	)
	s.TotalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_comparisons",
			Help:      "how many comparisons need to be done",
		},
	)
	s.TotalFiles = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_files",
			Help:      "how many files we were give to compare",
		},
	)
	s.PairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	s.ImageCacheBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_size_bytes",
			Help:      "disk size of the cache",
		},
	)
	s.ImageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_num_images",
			Help:      "how many images are in the cache",
		},
	)
	s.FileMapBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "file_map_bytes",
			Help:      "size of the file dedup map",
		},
	)
	prometheus.MustRegister(s.PairTotal)
	prometheus.MustRegister(s.GCTime)
	prometheus.MustRegister(s.TotalComparisons)
	prometheus.MustRegister(s.TotalFiles)
	prometheus.MustRegister(s.ImageCacheBytes)
	prometheus.MustRegister(s.ImageCacheNumImages)
	prometheus.MustRegister(s.FileMapBytes)

	return s
}

// publishStats publishes go GC stats + cache size to prom every 10 seconds
func (s *stats) publishStats(imageCache *hash.Cache, fileMap *roaring64.Bitmap, dedupPairs bool, bitmapLock *sync.RWMutex) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		s.GCTime.Set(float64(stats.PauseTotalNs))

		var numImages, cacheBytes = imageCache.Stats()
		s.ImageCacheNumImages.Set(float64(numImages))
		s.ImageCacheBytes.Set(float64(cacheBytes))

		if dedupPairs {
			bitmapLock.Lock()
			var b, _ = fileMap.ToBytes()
			s.FileMapBytes.Set(float64(len(b)))
			bitmapLock.Unlock()
		}

		time.Sleep(10 * time.Second)
	}
}
