package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pairCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "pair_cache_hits",
		},
	)
	pairCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "pair_cache_misses",
		},
	)
)

func init() {
	prometheus.MustRegister(pairCacheHits)
	prometheus.MustRegister(pairCacheMisses)
}

// emptyStruct is just that, we delcare it once to save allocs
var emptyStruct = struct{}{}

// pairCache is all the file pairs we have already diff'd
// as well as the last pair to get diffed
type pairCache struct {
	Cache    map[string]struct{}
	LastPair pair
	Lock     sync.RWMutex
}

// pair represents two images, their paths and thier element # in the files list
type pair struct {
	I   int
	J   int
	One string
	Two string
}

func (pc *pairCache) Get(fileOne, fileTwo string) bool {
	pc.Lock.RLock()
	defer pc.Lock.RUnlock()

	var _, found = pc.Cache[fileOne+fileTwo]
	if !found {
		// try the other way around
		_, found = pc.Cache[fileTwo+fileOne]
		if !found {
			pairCacheMisses.Inc()
		} else {
			pairCacheHits.Inc()
		}
	}

	return found
}

func (pc *pairCache) Set(fileOne, fileTwo string) {
	pc.Lock.Lock()
	defer pc.Lock.Unlock()

	pc.Cache[fileOne+fileTwo] = emptyStruct
	pc.Cache[fileTwo+fileOne] = emptyStruct
}

// getCheckpoints is used on startup to get the last pair of images compared
// if its the first time running it will just return 0,0
func NewPairFromCache(file string) (*pairCache, error) {

	var p = new(pairCache)

	var f, err = os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(file)
			p.Cache = make(map[string]struct{})
			return p, err
		} else {
			return nil, err
		}
	}

	// dump map to file
	err = json.NewDecoder(f).Decode(p)
	if err != nil {
		return p, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Drain reads from the chan and caches the lastest pair, used to save to disk later
func (pc *pairCache) Drain(completedPair pair) {
	pc.LastPair = completedPair
}

/*
func (p *pair) Drain(completedPairs chan pair) {
	for newPair := range completedPairs {
		p = &newPair
	}
}
*/

// Save stores the last pair we diff'd so we know where to start next time
func (pc *pairCache) Save(file string) error {

	var f, err = os.Create(lastCheckpointFile)
	if err != nil {
		return err
	}

	// dump map to file
	err = json.NewEncoder(f).Encode(pc)
	if err != nil {
		return err
	}

	return f.Close()
}
