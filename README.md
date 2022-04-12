# ImageDup
[![ImageDup](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml/badge.svg)](https://github.com/kmulvey/imagedup/actions/workflows/release_build.yml) [![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://vshymanskyy.github.io/StandWithUkraine)

Got a lot of images with many duplicates, maybe of different sizesimagedup uses [perceptual hashing](https://en.wikipedia.org/wiki/Perceptual_hashing) to find images that 

## Run
```
./imagedup -threads=2 -distance=10 -dir=/path/to/images

# this will create delete.log which will be used by the verify tool.

./verify
```

print help:

`imagedup -h`
