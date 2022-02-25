package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os/signal"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const PromNamespace = "imagedup"
const hashCacheFile = "hashcache.json"
const lastCheckpointFile = "checkpoint.json"

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

// pair represents two images, their paths and thier element # in the files list
type pair struct {
	I   int
	J   int
	One string
	Two string
}

func init() {

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

	var files, imageHashCache = setup(rootDir)

	// db to store pairs that are alreay done
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)

	var pairChan = make(chan pair)
	var killChan = make(chan struct{})
	go streamFiles(files, pairChan, killChan)

	fmt.Println("started, go to grafana to monitor")

	// feed the files into the diff workers
Loop:
	for {
		select {
		case <-gracefulShutdown:
			fmt.Println("shutting down")
			close(killChan)
			shutdown(imageHashCache)
			break Loop
		default:
			select {
			case p, open := <-pairChan:
				if !open {
					break Loop
				}
				diff(imageHashCache, p)
			}
		}
	}

	close(killChan)
	shutdown(imageHashCache)
	fmt.Println("Total time taken:", time.Since(start))
}

func setup(rootDir string) ([]string, *hashCache) {

	// list all the files
	files, err := listFiles(rootDir)
	handleErr("listFiles", err)

	// init the image cache
	imageHashCache, err := NewHashCache(hashCacheFile)
	handleErr("NewHashCache", err)
	log.Infof("Loaded %d image hashes from disk cache", len(imageHashCache.Cache))

	go publishStats(imageHashCache)

	// starter stats
	totalComparisons.Set(float64(len(files) * (len(files) - 1)))

	return files, imageHashCache
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

	diffTime.Set(float64(time.Since(start)))
	comparisonsCompleted.Inc()
}
