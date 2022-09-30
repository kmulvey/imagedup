package imagedup

import (
	"context"
	"os"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

var expectedPairs = map[string]struct{}{
	"testimages/iceland-small.jpgtestimages/iceland.jpg": {},
	"testimages/iceland-small.jpgtestimages/trees.jpg":   {},
	"testimages/iceland.jpgtestimages/trees.jpg":         {},
}

func TestStreamFiles(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestStreamFiles"
	var id, err = NewImageDup("TestStreamFiles", cacheFile, "glob", 2, 3, 10, true)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		for img := range id.images {
			delete(expectedPairs, img.One+img.Two)
		}
		assert.Equal(t, 0, len(expectedPairs))

		close(done)
	}()

	files, err := path.ListFiles("./testimages")
	assert.NoError(t, err)
	files = path.OnlyFiles(files)
	var fileNames = path.OnlyNames(files)

	id.streamFiles(context.Background(), fileNames)

	<-done

	assert.NoError(t, os.RemoveAll(cacheFile))
}

func TestStreamFilesCancel(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestStreamFiles"
	var id, err = NewImageDup("TestStreamFilesCancel", cacheFile, "glob", 2, 3, 10, true)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		var numImages int
		for range id.images {
			numImages++
		}
		assert.True(t, numImages < 100) // kind of arbitrary, it basically just needs to be small

		close(done)
	}()

	var ctx, cancel = context.WithCancel(context.Background())
	go id.streamFiles(ctx, make([]string, 100))
	cancel()

	<-done
	assert.NoError(t, os.RemoveAll(cacheFile))
}
