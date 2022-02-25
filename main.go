package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os/signal"

	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const PromNamespace = "imagedup"
const hashCacheFile = "hashcache.json"
const lastCheckpointFile = "checkpoint.json"

var (
	diffTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "diff_time_nano",
			Help:      "How long it takes to diff two images, in nanoseconds.",
		},
	)
	pairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	gcTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "gc_time_nano",
		},
	)
	gcOpTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "gc_op_total",
		},
	)
	totalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "total_comparisons",
		},
	)
	comparisonsCompleted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "comparisons_completed",
		},
	)
	imageCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_size_bytes",
		},
	)
	imageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_num_images",
		},
	)
	pairCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "pair_cache_size",
		},
	)
)

var deleteLogger *logrus.Logger

type DeleteLogFormatter struct {
}

func (f *DeleteLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf = new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s: %s		", "big", entry.Data["big"].(string)))
	buf.WriteString(fmt.Sprintf("%s: %s\n", "small", entry.Data["small"].(string)))

	var js, _ = json.Marshal(entry.Data)
	return append(js, '\n'), nil
}

func init() {
	prometheus.MustRegister(diffTime)
	prometheus.MustRegister(pairTotal)
	prometheus.MustRegister(gcOpTotal)
	prometheus.MustRegister(gcTime)
	prometheus.MustRegister(totalComparisons)
	prometheus.MustRegister(comparisonsCompleted)
	prometheus.MustRegister(imageCacheSize)
	prometheus.MustRegister(imageCacheNumImages)
	prometheus.MustRegister(pairCacheSize)

	log.SetFormatter(&log.TextFormatter{})
	var file, err = os.OpenFile("delete.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
		os.Exit(1)
	}

	deleteLogger = logrus.New()
	deleteLogger.SetFormatter(new(DeleteLogFormatter))
	deleteLogger.SetOutput(file)
}

func main() {

	var start = time.Now()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, os.Kill)

	// prom
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":5000", nil))
	}()

	var rootDir string
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.Parse()
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}

	var pairCache, files, imageHashCache = setup(rootDir)

	// db to store pairs that are alreay done
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)

	var pairChan = make(chan pair)
	var killChan = make(chan struct{})
	go streamFiles(pairCache, files, pairChan, killChan)

	fmt.Println("started, go to grafana to monitor")

	// feed the files into the diff workers
Loop:
	for {
		select {
		case <-gracefulShutdown:
			fmt.Println("shutting down")
			close(killChan)
			shutdown(pairCache, imageHashCache)
			break Loop
		default:
			select {
			case p, open := <-pairChan:
				if !open {
					break Loop
				}
				diff(imageHashCache, p)
				pairCache.Drain(p)
			}
		}
	}
	fmt.Println("Total time taken:", time.Since(start))
}

func setup(rootDir string) (*pairCache, []string, *hashCache) {

	// get points from where we left off last time
	var pairCache, err = NewPairFromCache(lastCheckpointFile)
	handleErr("NewPairFromCache", err)
	log.Infof("Loaded %d pairs from disk cache", len(pairCache.Cache))

	// list all the files
	files, err := listFiles(rootDir)
	handleErr("listFiles", err)

	// init the image cache
	imageHashCache, err := NewHashCache(hashCacheFile)
	handleErr("NewHashCache", err)
	log.Infof("Loaded %d image hashes from disk cache", len(imageHashCache.Cache))

	go publishStats(imageHashCache, pairCache)

	// starter stats
	totalComparisons.Set(float64(len(files) * (len(files) - 1)))
	comparisonsCompleted.Set(float64(pairCache.LastPair.I*len(files) + pairCache.LastPair.J))

	return pairCache, files, imageHashCache
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		fmt.Println(prefix, err)
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}

// listFiles recursivly traverses the root directory and adds every .jpg to a string slice and returns it
func listFiles(root string) ([]string, error) {
	var allFiles []string
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return allFiles, err
	}
	for _, file := range files {
		if file.IsDir() {
			var subFiles, err = listFiles(path.Join(root, file.Name()))
			if err != nil {
				return allFiles, err
			}
			allFiles = append(allFiles, subFiles...)
		} else {
			if strings.HasSuffix(file.Name(), ".jpg") {
				allFiles = append(allFiles, path.Join(root, file.Name()))
			}
		}
	}
	return allFiles, nil
}

// n = total # of files
// total comparisons = n^2 - n
func streamFiles(pc *pairCache, files []string, pairChan chan pair, killChan chan struct{}) {
	var started bool
	for i, one := range files {
		for j, two := range files {
			// trying to find where we left off last time
			if !started {
				if i == pc.LastPair.I && j == pc.LastPair.J {
					started = true
				} else {
					continue
				}
			}

			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case _, open := <-killChan:
					if !open {
						close(pairChan)
						return
					}
				default:
					//if !pc.Get(one, two) {
					pairChan <- pair{One: one, Two: two, I: i, J: j}
					pairTotal.Inc()
					//	pc.Set(one, two)
					//}
				}
			}
		}
	}
	close(pairChan)
}

func diff(cache *hashCache, p pair) {
	var start = time.Now()

	var imgCacheOne, err = cache.GetHash(p.One)
	handleErr("get hash: "+p.One, err)

	imgCacheTwo, err := cache.GetHash(p.Two)
	handleErr("get hash: "+p.One, err)

	distance, err := imgCacheOne.ImageHash.Distance(imgCacheTwo.ImageHash)
	handleErr("distance", err)

	if distance < 10 {
		if (imgCacheOne.Config.Height * imgCacheOne.Config.Width) > (imgCacheTwo.Config.Height * imgCacheTwo.Config.Width) {
			deleteLogger.WithFields(log.Fields{
				"big":   p.One,
				"small": p.Two,
			}).Info("delete")
		} else {
			deleteLogger.WithFields(log.Fields{
				"big":   p.Two,
				"small": p.One,
			}).Info("delete")
		}
	}

	//		checkpoints <- p
	diffTime.Set(float64(time.Since(start)))
	comparisonsCompleted.Inc()
}

// mergeStructs is a concurrent merge function that combines all input chans
func merge(cs ...chan struct{}) <-chan struct{} {
	var wg sync.WaitGroup
	out := make(chan struct{})

	output := func(c <-chan struct{}) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// publishStats publishes go GC stats + cache size to prom
func publishStats(imageCache *hashCache, pc *pairCache) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		gcOpTotal.Set(float64(stats.NumGC))
		gcTime.Set(float64(stats.PauseTotalNs))

		imageCacheSize.Set(float64(imageCache.Size()))
		imageCacheNumImages.Set(float64(imageCache.NumImages()))
		pairCacheSize.Set(float64(pc.Size()))

		time.Sleep(10 * time.Second)
	}
}
