package types

// Pair represents two images, their paths and their element # in the files list
// Each Pair is 48 bytes
type Pair struct {
	One string
	Two string
	I   int
	J   int
}
