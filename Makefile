REPOPATH = github.com/kmulvey/imagedup
BUILDS := nsquared uniqdirs verify

build: 
	for target in $(BUILDS); do \
		go build -v -ldflags="-s -w" -o ./cmd/$$target ./cmd/$$target; \
	done
