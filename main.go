package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/corona10/goimagehash/etcs"
	"github.com/corona10/goimagehash/transforms"
	"github.com/dgraph-io/badger/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/draw"
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
	totalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "total_comparisons",
		},
	)
	comparisonsCompleted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "comparisons_completed",
		},
	)
)

var deleteLogger *logrus.Logger

type DeleteLogFormatter struct {
}

func (f *DeleteLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf = new(bytes.Buffer)
	buf.WriteString(entry.Data["cmd"].(string))
	buf.WriteString(fmt.Sprintf("\n%s: %s		", "big", entry.Data["big"].(string)))
	buf.WriteString(fmt.Sprintf("%s: %s\n", "small", entry.Data["small"].(string)))

	return buf.Bytes(), nil
}

func init() {
	prometheus.MustRegister(diffTime)
	prometheus.MustRegister(pairTotal)
	prometheus.MustRegister(gcOpTotal)
	prometheus.MustRegister(gcTime)
	prometheus.MustRegister(totalComparisons)
	prometheus.MustRegister(comparisonsCompleted)

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

	// get points from where we left off last time
	var startI, startJ = getCheckpoints()

	// list all the files
	var files, err = listFiles(rootDir)
	handleErr("listfiles", err)
	totalComparisons.Set(math.Pow(float64(len(files)), 2))
	comparisonsCompleted.Set(float64(startI*len(files) + startJ))

	// spin up the diff workers
	var threads = 6
	var checkpoints = make(chan pair)
	go cacheCheckpoint(checkpoints)
	var fileChans = make([]chan pair, threads)
	var doneChans = make([]chan struct{}, threads)
	for i := 0; i < threads; i++ {
		fileChans[i] = make(chan pair, 10)
		doneChans[i] = make(chan struct{})
		go diff(rootDir, fileChans[i], checkpoints, doneChans[i])
	}

	fmt.Println("started, go to grafana to monitor")

	// feed the files into the diff workers
	var started bool
	for i, one := range files {
		for j, two := range files {
			if !started {
				if i == startI && j == startJ {
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

	for _, c := range fileChans {
		close(c)
	}

	<-merge(doneChans...)
	fmt.Println(time.Since(start))
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
		//hash1, err := goimagehash.PerceptionHash(img1)
		hash1, err := Hash(img1)
		handleErr("PerceptionHash: "+file1.Name(), err)
		//hash2, err := goimagehash.PerceptionHash(img2)
		hash2, err := Hash(img2)
		handleErr("PerceptionHash: "+file2.Name(), err)
		distance, err := hash1.Distance(hash2)
		handleErr("distance", err)

		if distance < 15 {
			file1.Seek(0, 0) // reset file reader
			oneDimensions, err := jpeg.DecodeConfig(file1)
			handleErr("DecodeConfig: "+file1.Name(), err)
			file2.Seek(0, 0)
			twoDimensions, err := jpeg.DecodeConfig(file2)
			handleErr("DecodeConfig: "+file2.Name(), err)
			var oneStr = fmt.Sprintf("%d, %d, %s", oneDimensions.Height, oneDimensions.Width, file1.Name())
			handleErr("printf: ", err)
			var twoStr = fmt.Sprintf("%d, %d, %s", twoDimensions.Height, twoDimensions.Width, file2.Name())
			handleErr("printf: ", err)

			if (oneDimensions.Height * oneDimensions.Width) > (twoDimensions.Height * twoDimensions.Width) {
				deleteLogger.WithFields(log.Fields{
					"cmd":   "rm " + file2.Name(),
					"big":   oneStr,
					"small": twoStr,
				}).Info("delete")
			} else {
				deleteLogger.WithFields(log.Fields{
					"cmd":   "rm " + file1.Name(),
					"big":   twoStr,
					"small": oneStr,
				}).Info("delete")
			}
		}
		err = file1.Close()
		handleErr("file close: "+file1.Name(), err)
		err = file2.Close()
		handleErr("file close: "+file1.Name(), err)

		checkpoints <- p
		diffTime.Set(float64(time.Since(start)))
		comparisonsCompleted.Inc()
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

func getCheckpoints() (int, int) {
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

	return startI, startJ
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

func Hash(img image.Image) (*goimagehash.ImageHash, error) {
	if img == nil {
		return nil, errors.New("Image object can not be nil")
	}

	phash := goimagehash.NewImageHash(0, 2)

	// resize
	sr := img.Bounds()
	dr := image.Rect(0, 0, 64, 64)
	dst := image.NewRGBA(dr)
	draw.NearestNeighbor.Scale(dst, dr, img, sr, draw.Src, nil)

	// gray
	pixels := transforms.Rgb2Gray(dst)

	dct := transforms.DCT2D(pixels, 64, 64)
	flattens := transforms.FlattenPixels(dct, 8, 8)
	median := etcs.MedianOfPixels(flattens)

	for idx, p := range flattens {
		if p > median {
			phash.LeftShiftSet(len(flattens) - idx - 1)
		}
	}
	return phash, nil
}
