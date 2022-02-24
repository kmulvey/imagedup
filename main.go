package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	_ "net/http/pprof"

	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type pair struct {
	I   int
	J   int
	One string
	Two string
}

const PromNamespace = "imagedup"

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
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// prom
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":5000", nil))
	}()

	var start = time.Now()

	var rootDir string
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.Parse()
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}

	// get points from where we left off last time
	var startI, startJ = getCheckpoints()

	// list all the files
	var files, err = listFiles(rootDir)
	handleErr("listfiles", err)

	totalComparisons.Set(float64(len(files) * (len(files) - 1)))
	comparisonsCompleted.Set(float64(startI*len(files) + startJ))

	// spin up the diff workers
	var threads = 2
	var checkpoints = make(chan pair)
	go cacheCheckpoint(checkpoints)
	var fileChans = make([]chan pair, threads)
	var doneChans = make([]chan struct{}, threads)
	var hashCache = NewHashCache()
	go publishStats(hashCache)
	for i := 0; i < threads; i++ {
		fileChans[i] = make(chan pair, 10)
		doneChans[i] = make(chan struct{})
		go diff(hashCache, rootDir, fileChans[i], checkpoints, doneChans[i])
	}

	// db to store pairs that are alreay done
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)
	pairDB, err := badger.Open(badger.DefaultOptions("pairs").WithLogger(dbLogger))
	handleErr("pairs db open", err)

	fmt.Println("started, go to grafana to monitor")

	// feed the files into the diff workers
	var started bool
	for i, one := range files {
		for j, two := range files {
			// trying to find where we left off last time
			if !started {
				if i == startI && j == startJ {
					started = true
				} else {
					continue
				}
			}

			if i != j {
				if !getPair(pairDB, one, two) {
					fileChans[j%threads] <- pair{One: one, Two: two, I: i, J: j}
					pairTotal.Inc()
					setPair(pairDB, one, two)
				}
			}
		}
	}

	for _, c := range fileChans {
		close(c)
	}
	err = pairDB.Close()
	handleErr("close db", err)

	<-merge(doneChans...)
	fmt.Println(time.Since(start))
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
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

func diff(cache *hashCache, rootDir string, pairs, checkpoints chan pair, done chan struct{}) {
	for p := range pairs {
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

		checkpoints <- p
		diffTime.Set(float64(time.Since(start)))
		comparisonsCompleted.Inc()
	}
	close(done)
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
func publishStats(hashCache *hashCache) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		gcOpTotal.Set(float64(stats.NumGC))
		gcTime.Set(float64(stats.PauseTotalNs))

		imageCacheSize.Set(float64(hashCache.Size()))
		imageCacheNumImages.Set(float64(hashCache.NumImages()))

		time.Sleep(10 * time.Second)
	}
}
