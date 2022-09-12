package hash

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDiffer(t *testing.T) {
	t.Parallel()

	var cacheFile = "testdiffer.json"
	var inputImages = make(chan types.Pair)

	var cache, err = NewCache(cacheFile, "testdiffer")
	assert.NoError(t, err)

	var differ = NewDiffer(2, 10, inputImages, cache, "testdiffer")
	assert.NoError(t, err)

	var results, errors = differ.Run(context.Background())
	var done = make(chan struct{})
	go func() {
		var r = true
		var e = true
		var i int
		for r && e {
			select {
			case err, open := <-errors:
				if !open {
					e = false
					continue
				}
				assert.NoError(t, err)
			case diff, open := <-results:
				if !open {
					r = false
					continue
				}
				assert.Equal(t, "../testimages/iceland-small.jpg", diff.One)
				assert.Equal(t, "../testimages/trees.jpg", diff.Two)
				i++
			}
		}
		assert.Equal(t, 1, i)
		close(done)
	}()
	time.Sleep(500 * time.Millisecond)

	inputImages <- types.Pair{One: "../testimages/iceland.jpg", Two: "../testimages/iceland.jpg"}
	inputImages <- types.Pair{One: "../testimages/iceland-small.jpg", Two: "../testimages/iceland.jpg"}
	inputImages <- types.Pair{One: "../testimages/iceland-small.jpg", Two: "../testimages/trees.jpg"}
	close(inputImages)

	<-done

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}
