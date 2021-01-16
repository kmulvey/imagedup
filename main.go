package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
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

var (
	diffTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "diff_time_nano",
			Help: "How long it takes to diff two images, in nanoseconds.",
		},
	)
	pairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "pair_total",
			Help: "How many pairs we read.",
		},
	)
	gcTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gc_time_nano",
		},
	)
	gcOpTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gc_op_total",
		},
	)
)

func init() {
	prometheus.MustRegister(diffTime)
	prometheus.MustRegister(pairTotal)
	prometheus.MustRegister(gcOpTotal)
	prometheus.MustRegister(gcTime)
}

func main() {
	var rootDir string
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.Parse()
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":5000", nil))
	}()
	go publishStats()

	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)
	var opts = badger.DefaultOptions("checkpoints").WithLogger(dbLogger)

	var db, err = badger.Open(opts)
	handleErr("badger open", err)
	var valBytes []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("checkpoint"))
		if errors.Is(err, badger.ErrKeyNotFound) {
			valBytes = []byte("0 0")
			return nil
		}
		handleErr("tnx get", err)

		valBytes, err = item.ValueCopy(valBytes)
		handleErr("Value copy", err)
		return nil
	})
	handleErr("db view", err)

	var valSlice = strings.Split(string(valBytes), " ")
	startI, err := strconv.Atoi(valSlice[0])
	handleErr("atoi: "+valSlice[0], err)
	startJ, err := strconv.Atoi(valSlice[1])
	handleErr("atoi: "+valSlice[1], err)
	err = db.Close()
	handleErr("db close", err)

	files, err := listFiles(rootDir)
	handleErr("listfiles", err)

	var threads = 4
	var checkpoints = make(chan pair)
	go cacheCheckpoint(checkpoints)
	var fileChans = make([]chan pair, threads)
	var doneChans = make([]chan struct{}, threads)
	for i := 0; i < threads; i++ {
		fileChans[i] = make(chan pair, 10)
		doneChans[i] = make(chan struct{})
		go diff(rootDir, fileChans[i], checkpoints, doneChans[i])
	}

	var started bool
	for i, one := range files {
		for j, two := range files {
			if !started {
				if i == startI && j == startJ {
					fmt.Println("picking up at", i, j)
					started = true
				} else {
					continue
				}
			}

			if i != j {
				fileChans[j%threads] <- pair{One: one, Two: two, I: i, J: j}
				pairTotal.Inc()
			}
		}
	}

	<-merge(doneChans...)
}

func handleErr(prefix string, err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}
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

func diff(rootDir string, pairs, checkpoints chan pair, done chan struct{}) {
	for p := range pairs {
		var start = time.Now()
		file1, err := os.Open(p.One)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		handleErr("file open: "+file1.Name(), err)
		file2, err := os.Open(p.Two)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		handleErr("file open: "+file1.Name(), err)

		img1, err := jpeg.Decode(file1)
		handleErr("jpeg.Decode: "+file1.Name(), err)
		img2, err := jpeg.Decode(file2)
		handleErr("jpeg.Decode: "+file2.Name(), err)
		hash1, err := goimagehash.PerceptionHash(img1)
		handleErr("PerceptionHash: "+file1.Name(), err)
		hash2, err := goimagehash.PerceptionHash(img2)
		handleErr("PerceptionHash: "+file2.Name(), err)
		distance, err := hash1.Distance(hash2)
		handleErr("distance", err)

		if distance == 0 {
			oneDimensions, _, err := image.DecodeConfig(file1)
			handleErr("DecodeConfig: "+file1.Name(), err)
			twoDimensions, _, err := image.DecodeConfig(file2)
			handleErr("DecodeConfig: "+file2.Name(), err)

			if (oneDimensions.Height * oneDimensions.Width) > (twoDimensions.Height * twoDimensions.Width) {
				fmt.Println("delete:", file2)
			} else {
				fmt.Println("delete:", file1)
			}
		}
		file1.Close()
		file2.Close()

		checkpoints <- p
		diffTime.Set(float64(time.Since(start)))
	}
	close(done)
}

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

func cacheCheckpoint(checkpoints chan pair) {
	var dbLogger = logrus.New()
	dbLogger.SetLevel(log.WarnLevel)
	var opts = badger.DefaultOptions("checkpoints").WithLogger(dbLogger)
	var db, err = badger.Open(opts)
	handleErr("badger open", err)

	txn := db.NewTransaction(true) // Read-write txn
	var i int
	for cp := range checkpoints {
		i++
		err = txn.Set([]byte("checkpoint"), []byte(strconv.Itoa(cp.I)+" "+strconv.Itoa(cp.J)))
		handleErr("txn.set", err)

		if i%50 == 0 {
			err = txn.Commit()
			handleErr("txn commit", err)
			txn = db.NewTransaction(true)
		}
	}
	err = txn.Commit()
	handleErr("txn commit", err)

	err = db.Close()
	handleErr("db close", err)
}

func publishStats() {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		gcOpTotal.Set(float64(stats.NumGC))
		gcTime.Set(float64(stats.PauseTotalNs))

		time.Sleep(10 * time.Second)
	}
}
