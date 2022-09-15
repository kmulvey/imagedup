# ImageDup
[![ImageDup](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml/badge.svg)](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml) [![codecov](https://codecov.io/gh/kmulvey/imagedup/branch/main/graph/badge.svg?token=wp6NcwDC5k)](https://codecov.io/gh/kmulvey/imagedup) [![Go Report Card](https://goreportcard.com/badge/github.com/kmulvey/imagedup)](https://goreportcard.com/report/github.com/kmulvey/imagedup) [![Go Reference](https://pkg.go.dev/badge/github.com/kmulvey/imagedup.svg)](https://pkg.go.dev/github.com/kmulvey/imagedup)

Got a lot of images with many duplicates? Maybe of different sizes? `imagedup` uses [perceptual hashing](https://en.wikipedia.org/wiki/Perceptual_hashing) to find images that are close in appearance but not exact. Once `imagedup` is finished the `verify` tool can be used to read the delete log and open images in pairs so you can double check them before they are deleted. This step is necessary as perceptual hashing is not perfect and will sometimes show two completely different images.

## Run
```
./nsquared -cache-file cache.json -output-file delete.log -dir /path/to/images -threads 5 -dedup-file-pairs true

# this will create delete.log which will be used by the verify tool.

./verify -delete-file delete.log
```

print help:

`imagedup -h`
