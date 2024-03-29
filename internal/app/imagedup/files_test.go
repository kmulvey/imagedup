package imagedup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestStreamFiles(t *testing.T) {
	t.Parallel()

	var expectedPairs = map[string]struct{}{
		"iceland-small.jpgiceland.jpg": {},
		"iceland-small.jpgtrees.jpg":   {},
		"iceland.jpgtrees.jpg":         {},
	}
	var cacheFile = "TestStreamFiles"
	var id, err = NewImageDup("TestStreamFiles", cacheFile, 2, 3, 10, false)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		for img := range id.images {
			delete(expectedPairs, filepath.Base(img.One)+filepath.Base(img.Two))
		}
		assert.Equal(t, 0, len(expectedPairs))

		close(done)
	}()

	files, err := path.List("./testimages", 1, false, path.NewFileEntitiesFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(files)

	id.streamFiles(context.Background(), fileNames)

	<-done

	assert.NoError(t, os.RemoveAll(cacheFile))
}

func TestStreamFilesDedup(t *testing.T) {
	t.Parallel()

	var expectedPairs = map[string]struct{}{
		"iceland-small.jpgiceland.jpg": {},
		"iceland-small.jpgtrees.jpg":   {},
		"iceland.jpgtrees.jpg":         {},
	}
	var cacheFile = "TestStreamFilesDedup"
	var id, err = NewImageDup("TestStreamFilesDedup", cacheFile, 2, 3, 10, true)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		for img := range id.images {
			delete(expectedPairs, filepath.Base(img.One)+filepath.Base(img.Two))
		}
		assert.Equal(t, 0, len(expectedPairs))

		close(done)
	}()

	files, err := path.List("./testimages", 1, false, path.NewFileEntitiesFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(files)

	id.streamFiles(context.Background(), fileNames)

	<-done

	assert.NoError(t, os.RemoveAll(cacheFile))
}

func TestStreamFilesCancel(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestStreamFiles"
	var id, err = NewImageDup("TestStreamFilesCancel", cacheFile, 2, 3, 10, true)
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
