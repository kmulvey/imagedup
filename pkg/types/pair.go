package types

// Pair represents two images, their paths and their element # in the files list
// 48 bytes each
type Pair struct {
	I   int
	J   int
	One string
	Two string
}
