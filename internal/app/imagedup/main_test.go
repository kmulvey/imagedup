package imagedup

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kmulvey/path"
	"github.com/stretchr/testify/assert"
)

func TestNewImageDup(t *testing.T) {
	t.Parallel()

	var cacheFile = "TestNewImageDup.json"

	var dup, err = NewImageDup("TestNewImageDup", cacheFile, "glob", 2, 1, 10, true)
	assert.Equal(t, "Skipping glob because there are only 1 files", err.Error())
	assert.Nil(t, dup)

	dup, err = NewImageDup("TestNewImageDup", cacheFile, "glob", 2, 3, 10, true)
	assert.NoError(t, err)

	dirs, err := path.List("./testimages", path.NewFileListFilter())
	assert.NoError(t, err)
	var fileNames = path.OnlyNames(dirs)

	var results, errors = dup.Run(context.Background(), fileNames)
	var done = make(chan struct{})

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
				assert.True(t, strings.Contains(diff.One, "iceland"))
				i++
			}
		}
		assert.Equal(t, i, i)
		close(done)
	}()

	<-done

	assert.NoError(t, dup.Shutdown())

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}
