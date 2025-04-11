package imagedup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func testStreamFilesHelper(t *testing.T, cacheFile string, dedupPairs bool, expectedPairs map[string]struct{}) {
	t.Helper()

	var id, err = NewImageDup(cacheFile, cacheFile, 2, 3, 10, dedupPairs)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		for img := range id.images {
			delete(expectedPairs, filepath.Base(img.One)+filepath.Base(img.Two))
		}
		assert.Empty(t, expectedPairs)

		close(done)
	}()

	files, err := path.List("./testimages", 1, false, path.NewFileEntitiesFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(files)

	id.streamFiles(t.Context(), fileNames)

	<-done

	assert.NoError(t, os.RemoveAll(cacheFile))
}

func TestStreamFiles(t *testing.T) {
	t.Parallel()

	var expectedPairs = map[string]struct{}{
		"iceland-small.jpgiceland.jpg": {},
		"iceland-small.jpgtrees.jpg":   {},
		"iceland.jpgtrees.jpg":         {},
	}
	testStreamFilesHelper(t, "TestStreamFiles", false, expectedPairs)
}

func TestStreamFilesDedup(t *testing.T) {
	t.Parallel()

	var expectedPairs = map[string]struct{}{
		"iceland-small.jpgiceland.jpg": {},
		"iceland-small.jpgtrees.jpg":   {},
		"iceland.jpgtrees.jpg":         {},
	}
	testStreamFilesHelper(t, "TestStreamFilesDedup", true, expectedPairs)
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
		assert.Less(t, numImages, 100) // kind of arbitrary, it basically just needs to be small

		close(done)
	}()

	var ctx, cancel = context.WithCancel(t.Context())
	go id.streamFiles(ctx, make([]string, 100))
	cancel()

	<-done
	assert.NoError(t, os.RemoveAll(cacheFile))
}
