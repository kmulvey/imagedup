package main

import (
	"encoding/json"
	"os"
)

// pair represents two images, their paths and thier element # in the files list
type pair struct {
	I   int
	J   int
	One string
	Two string
}

// getCheckpoints is used on startup to get the last pair of images compared
// if its the first time running it will just return 0,0
func NewCheckpoints(file string) (int, int) {

	var f, err = os.Open(file)
	if err != nil {
		return 0, 0
	}
	defer f.Close() // dont really care about this error

	// dump map to file
	var pair = new(pair)
	err = json.NewDecoder(f).Decode(pair)
	if err != nil {
		return 0, 0
	}

	return pair.I, pair.J
}

// Drain reads from the chan and caches the lastest pair, used to save to disk later
func (p *pair) Drain(completedPairs chan pair) {
	for newPair := range completedPairs {
		p = &newPair
	}
}

// Save stores the last pair we diff'd so we know where to start next time
func (p *pair) Save(file string) error {

	var f, err = os.Create(lastCheckpointFile)
	if err != nil {
		return err
	}

	// dump map to file
	err = json.NewEncoder(f).Encode(p)
	if err != nil {
		return err
	}

	return f.Close()
}
