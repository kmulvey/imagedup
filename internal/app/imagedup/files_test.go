package imagedup

import (
	"context"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestStreamFiles(t *testing.T) {
	var id, err = NewImageDup(context.Background(), "TestStreamFiles", "cacheFile.json", "outputFile.log", 2, 10)
	assert.NoError(t, err)

	var done = make(chan struct{})
	go func() {
		for range id.images {

		}
		close(done)
	}()

	files, err := path.ListFiles("./testimages")
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(files)

	id.streamFiles(fileNames)

	<-done
}
