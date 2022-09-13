package imagedup

import (
	"context"
	"os"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestNewImageDup(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestNewImageDup.json"

	var dup, err = NewImageDup("TestNewImageDup", cacheFile, 2, 10)
	assert.NoError(t, err)

	files, err := path.ListFiles("./testimages")
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(path.OnlyFiles(files))

	var results, errors = dup.Run(context.Background(), fileNames)
	var done = make(chan struct{})

	var expectedOneFiles = map[string]struct{}{
		"testimages/iceland-small.jpg": {},
		"testimages/iceland.jpg":       {},
	}
	go func() {
		var i int
		for results != nil || errors != nil {
			select {
			case err, open := <-errors:
				if !open {
					errors = nil
					continue
				}
				assert.NoError(t, err)
			case diff, open := <-results:
				if !open {
					results = nil
					continue
				}
				delete(expectedOneFiles, diff.One)
				i++
			}
		}
		assert.Equal(t, 2, i)
		assert.Equal(t, 0, len(expectedOneFiles))
		close(done)
	}()

	<-done

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}
