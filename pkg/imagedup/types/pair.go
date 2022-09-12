package types

// Pair represents two images, their paths and their element # in the files list
// 48 bytes each
type Pair struct {
	One string
	Two string
	I   int
	J   int
}
