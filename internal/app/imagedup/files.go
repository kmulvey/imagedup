package imagedup

import (
	"context"
	"math"
	"regexp"

	"github.com/kmulvey/imagedup/v2/pkg/imagedup/types"
)

// ImageExtensionRegex captures file extensions we can work with.
var ImageExtensionRegex = regexp.MustCompile(".*.jpg$|.*.jpeg$|.*.png$|.*.webp$|.*.JPG$|.*.JPEG$|.*.PNG$|.*.WEBP$")

// streamFiles generates roughly n^2 comparisons and writes them to a channel that
// is read by the diff workers.
func (id *ImageDup) streamFiles(ctx context.Context, files []string) {
	var numImages = float64(len(files))
	if id.dedupPairs {
		id.stats.TotalComparisons.Set((math.Pow(numImages, 2) - numImages) / 2)
	} else {
		id.stats.TotalComparisons.Set((math.Pow(numImages, 2) - numImages))
	}

	for i, one := range files {
		for j, two := range files {
			if i != j { // dont diff yourself
				select {
				case <-ctx.Done():
					close(id.images)
					return
				default:
					if id.dedupPairs {
						id.bitmapLock.Lock()

						if _, found := id.dedupCache[one+" "+two]; !found {
							id.images <- types.Pair{One: one, Two: two}
							id.dedupCache[two+" "+one] = struct{}{} // we set the opposite pair so we skip it next time
							id.stats.PairTotal.Inc()
							id.stats.FileMapMisses.Inc()
						}
						id.stats.FileMapHits.Inc()

						id.bitmapLock.Unlock()
					} else {
						id.images <- types.Pair{One: one, Two: two}
						id.stats.PairTotal.Inc()
					}
				}
			}
		}
	}
	close(id.images)
}
