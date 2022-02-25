package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		fmt.Println(prefix, err)
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}

/*
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
*/
