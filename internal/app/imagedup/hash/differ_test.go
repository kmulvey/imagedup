package hash

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kmulvey/imagedup/v2/pkg/imagedup/types"
	"github.com/stretchr/testify/assert"
)

func TestDiffer(t *testing.T) {
	t.Parallel()

	var cacheFile = "testdiffer.json"
	var inputImages = make(chan types.Pair)

	var cache, err = NewCache(cacheFile, "testdiffer", 3)
	assert.NoError(t, err)

	var differ = NewDiffer(2, 10, inputImages, cache, "testdiffer")
	assert.NoError(t, err)

	var results, errors = differ.Run(context.Background())
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
				assert.False(t, strings.Contains(diff.One, "trees"))
				assert.False(t, strings.Contains(diff.Two, "trees"))
				i++
			}
		}
		assert.Equal(t, 2, i)
		close(done)
	}()

	inputImages <- types.Pair{I: 0, One: "../testimages/iceland.jpg", J: 0, Two: "../testimages/iceland.jpg"}
	inputImages <- types.Pair{I: 1, One: "../testimages/iceland-small.jpg", J: 0, Two: "../testimages/iceland.jpg"}
	inputImages <- types.Pair{I: 1, One: "../testimages/iceland-small.jpg", J: 2, Two: "../testimages/trees.jpg"}
	close(inputImages)

	<-done

	err = os.RemoveAll(cacheFile)
	assert.NoError(t, err)
}
