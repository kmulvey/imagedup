package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os/signal"
	"syscall"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const hashCacheFile = "hashcache.json"

// pair represents two images, their paths and thier element # in the files list
type pair struct {
	I   int
	J   int
	One string
	Two string
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
}

func main() {

	var start = time.Now()

	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)

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

	var files, imageHashCache, deleteLogger = setup(rootDir)

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
			var err = shutdown(imageHashCache)
			if err != nil {
				log.Fatal("error shutting down", err)
			}
			break Loop
		default:
			select {
			case p, open := <-pairChan:
				if !open {
					close(killChan)
					break Loop
				}
				diff(imageHashCache, p, deleteLogger)
			default:
			}
		}
	}

	var err = shutdown(imageHashCache)
	if err != nil {
		log.Fatal("error shutting down", err)
	}

	fmt.Println("Total time taken:", time.Since(start))
}

func setup(rootDir string) ([]string, *hashCache, *logrus.Logger) {

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

	return files, imageHashCache, NewDeleteLogger()
}

func diff(cache *hashCache, p pair, deleteLogger *logrus.Logger) {
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
