# ImageDup
[![ImageDup](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml/badge.svg)](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml) [![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://vshymanskyy.github.io/StandWithUkraine)

Got a lot of images with many duplicates? Maybe of different sizes? `imagedup` uses [perceptual hashing](https://en.wikipedia.org/wiki/Perceptual_hashing) to find images that are close in appearance but not exact. Once `imagedup` is finished the `verify` tool can be used to read the delete log and open images in pairs so you can double check them before they are deleted. This step is necessary as perceptual hashing is not perfect and will sometimes show two completely different images.

## Run
```
./imagedup -threads=2 -distance=10 -dir=/path/to/images

# this will create delete.log which will be used by the verify tool.

./verify
```

print help:

`imagedup -h`
