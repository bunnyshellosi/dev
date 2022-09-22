.DEFAULT_GOAL := build

.PHONY: build

build:
	goreleaser release --snapshot --rm-dist
