package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"testing"

	corona "github.com/corona10/goimagehash"

	"github.com/kmulvey/goimagehash"
	"github.com/kmulvey/goimagehash/transforms"
	"github.com/nfnt/resize"
	"github.com/stretchr/testify/assert"
	"golang.org/x/image/draw"
)

func TestDupNames(t *testing.T) {
	var dir = "/home/kmulvey/Documents"
	var files, err = listFiles(dir)
	assert.NoError(t, err)

	var static bool
	var fmap = make(map[string]bool)
	for _, f := range files {
		//var filename = strings.Replace(f, dir, "", 1)
		if _, ok := fmap[f]; !ok {
			fmap[f] = static
		} else {
			fmt.Println(f)
		}
	}
}

func BenchmarkCoronaPerceptionHash(b *testing.B) {

	var file, err = os.Open("/home/kmulvey/Documents/Valeria Mavrin.jpg")
	assert.NoError(b, err)

	img, err := jpeg.Decode(file)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		hash, err := corona.PerceptionHash(img)
		assert.NoError(b, err)
		assert.Equal(b, uint64(0x9c99caa3dc4a3476), hash.GetHash())
	}
}

func BenchmarkMyPerceptionHash(b *testing.B) {

	var file, err = os.Open("/home/kmulvey/Documents/Valeria Mavrin.jpg")
	assert.NoError(b, err)

	img, err := jpeg.Decode(file)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		hash, err := goimagehash.PerceptionHash(img)
		assert.NoError(b, err)
		assert.Equal(b, uint64(0x9c994aa3de4a7076), hash.GetHash())
	}
}

func BenchmarkNfntScale(b *testing.B) {
	var file, err = os.Open("/home/kmulvey/Documents/Valeria Mavrin.jpg")
	assert.NoError(b, err)

	img, err := jpeg.Decode(file)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		var small = resize.Resize(64, 64, img, resize.NearestNeighbor)
		transforms.Rgb2Gray(small)
	}
}

func BenchmarkXScale(b *testing.B) {
	var file, err = os.Open("/home/kmulvey/Documents/Valeria Mavrin.jpg")
	assert.NoError(b, err)

	img, err := jpeg.Decode(file)
	assert.NoError(b, err)

	sr := img.Bounds()
	dr := image.Rect(0, 0, 64, 64)
	dst := image.NewRGBA(dr)

	for i := 0; i < b.N; i++ {
		draw.NearestNeighbor.Scale(dst, dr, img, sr, draw.Src, nil)
		transforms.Rgb2Gray(dst)
	}
}
